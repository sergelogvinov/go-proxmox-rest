// Package main demonstrates node-affinity load balancing (see
// docs/node-lb.md): proxmox.WithNodeAffinity routes /nodes/{node}/...
// requests directly at the node that owns them instead of letting another
// node's pveproxy relay them, and falls back to the configured algorithm
// when a node's own endpoint is unknown or unreachable.
//
// It stands in for a real cluster with four fakeapi servers: a "delegate"
// cluster that answers for every node (as a caller would reach without
// affinity, via relay) and three single-node clusters, one per node, each
// reporting a distinct core count so a direct route is observable — the
// response can only have come from that node's own server. It then kills
// one node's server to show the circuit breaker opening and requests
// falling back to the delegate.
//
// Run it with: go run ./examples/fakeapi-loadbalancer
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	proxmox "github.com/sergelogvinov/go-proxmox-rest"
	"github.com/sergelogvinov/go-proxmox-rest/fakeapi"
)

const (
	fakeTokenID     = "fakeapi@pve!fakeapi"
	fakeTokenSecret = "fakeapi-secret"
)

func main() {
	nodeNames := []string{"pve1", "pve2", "pve3"}

	// The delegate: a single fakeapi cluster that knows every node,
	// standing in for the relay a caller would reach without affinity.
	delegate := fakeapi.NewCluster(nil, fakeapi.WithNodes(nodeNames...))
	defer delegate.Close()

	// One direct, single-node cluster per node, each reporting a distinct
	// core count so its answers are distinguishable from the delegate's.
	directClusters := map[string]*fakeapi.Cluster{}
	directCores := map[string]int{"pve1": 11, "pve2": 22, "pve3": 33}

	for _, name := range nodeNames {
		cl := fakeapi.NewCluster(nil, fakeapi.WithNodes(name))
		defer cl.Close()

		cl.Node(name).SetResources(directCores[name], 8<<30)
		directClusters[name] = cl
	}

	c := mustNewClient(delegate, directClusters)
	defer c.Close()

	demonstrate(c, nodeNames, directCores, directClusters)
}

// demonstrate drives every call shown in the package doc comment. Kept
// separate from main so its log.Fatal calls (an unreachable node is a hard
// error in this demo) don't trip gocritic's exitAfterDefer check against
// main's own cleanup defers — those still run on a normal return, since
// nothing in this function exits the process.
func demonstrate(c *proxmox.Client, nodeNames []string, directCores map[string]int, directClusters map[string]*fakeapi.Cluster) {
	ctx := context.Background()

	fmt.Println("direct routing (each answer's core count identifies its own node's server):")
	for _, name := range nodeNames {
		status, err := c.Nodes(name).Status(ctx)
		if err != nil {
			log.Fatalf("Nodes(%s).Status(): %v", name, err)
		}
		fmt.Printf("  node=%-5s cores=%d (direct server reports %d)\n", name, status.CPUInfo.Cores, directCores[name])
	}

	fmt.Println()
	fmt.Println("cluster-wide call (not node-scoped, stays on the delegate):")
	if _, err := c.Cluster().Status(ctx); err != nil {
		log.Fatalf("Cluster().Status(): %v", err)
	}
	fmt.Println("  ok")

	fmt.Println()
	fmt.Println("killing pve2's direct server to show the circuit breaker and fallback:")
	directClusters["pve2"].Close()

	fmt.Println("  first call (should fail with a bare transport error, not a Proxmox API error):")
	if _, err := c.Nodes("pve2").Status(ctx); err == nil {
		log.Fatal("expected the first call to a dead node to fail")
	} else {
		var apiErr *proxmox.APIError
		fmt.Printf("    error: %v (structured API error: %v)\n", err, errors.As(err, &apiErr))
	}

	fmt.Println("  second call (breaker is now open, falls back to the delegate):")
	if _, err := c.Nodes("pve2").Status(ctx); err != nil {
		log.Fatalf("expected the second call to fall back and succeed: %v", err)
	}
	fmt.Println("    ok, served by the delegate")
}

// mustNewClient builds the node-affinity client under test: a delegate
// balancer over the shared "relay" cluster, decorated with an explicit
// node -> direct-server endpoint map and an OnRoute hook that narrates
// every balancer decision.
func mustNewClient(delegate *fakeapi.Cluster, directClusters map[string]*fakeapi.Cluster) *proxmox.Client {
	endpoints := make(map[string]string, len(directClusters))
	for name, cl := range directClusters {
		endpoints[name] = cl.URL()
	}

	c, err := proxmox.New(proxmox.ClientConfig{},
		proxmox.WithURL(delegate.URL()),
		proxmox.WithTokenAuth(fakeTokenID, fakeTokenSecret),
		proxmox.WithLogger(quietLogger{}),
		proxmox.WithNodeAffinity(
			proxmox.WithNodeEndpoints(endpoints),
			proxmox.WithNodeMaxFailures(1),
			proxmox.WithOnRoute(func(node, baseURL string, d proxmox.RouteDecision) {
				fmt.Printf("  route  node=%-5s decision=%-16s base=%s\n", node, decisionString(d), baseURL)
			}),
		),
	)
	if err != nil {
		log.Fatalf("building client: %v", err)
	}

	return c
}

// quietLogger suppresses resty's "sensitive credentials over plain HTTP"
// warning, unconditional noise here since fakeapi's httptest.Server is
// always plain HTTP — mirrors fakeapi's own unexported quietLogger
// (fakeapi/logger.go), which Cluster.Client installs automatically but
// which this example can't reuse directly since it builds its own client
// with node-affinity options Cluster.Client doesn't take.
type quietLogger struct{}

func (quietLogger) Errorf(format string, v ...any) { log.Printf("ERROR RESTY "+format, v...) }

func (quietLogger) Warnf(format string, v ...any) {
	if strings.Contains(format, "sensitive credentials") {
		return
	}
	log.Printf("WARN RESTY "+format, v...)
}

func (quietLogger) Debugf(format string, v ...any) { log.Printf("DEBUG RESTY "+format, v...) }

func decisionString(d proxmox.RouteDecision) string {
	switch d {
	case proxmox.RouteDirect:
		return "direct"
	case proxmox.RouteUnknown:
		return "unknown-node"
	case proxmox.RouteUnhealthy:
		return "unhealthy"
	case proxmox.RouteNotNodeScoped:
		return "not-node-scoped"
	default:
		return "?"
	}
}
