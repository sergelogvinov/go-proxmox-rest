/*
Copyright 2026 Proxmox Community.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package proxmox

import (
	"context"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// defaultResolverRefresh is used when WithNodeResolver is given a
// non-positive refresh interval.
const defaultResolverRefresh = 60 * time.Second

// resolverFetchTimeout bounds a single NodeResolver.Nodes call, so a
// resolver that hangs (a stuck informer, an unreachable control plane)
// cannot leak the refresh goroutine or wedge Close.
const resolverFetchTimeout = 10 * time.Second

// NodeEndpoint binds a Proxmox node name to the base URL that reaches its
// API.
type NodeEndpoint struct {
	Node    string
	BaseURL string // scheme://host[:port], no path — LB endpoints carry no path
}

// NodeResolver resolves the cluster's node endpoints dynamically — e.g. one
// backed by a Kubernetes informer. Implementations must be safe for
// concurrent use, and must not route their own traffic through the client
// they are configuring: if an implementation does call the Proxmox API, it
// owns that (separate) client and the re-entrancy question that comes with
// it. See WithNodeResolver.
type NodeResolver interface {
	Nodes(ctx context.Context) ([]NodeEndpoint, error)
}

// registry answers "which base URL reaches node X?", consulting its sources
// in precedence order: an explicit static map (§6.1), a host template
// (§6.2), a NodeResolver (§6.4), then the pool of base URLs the delegate
// balancer was built from, matched by node name (§6.3, the zero-config
// default).
type registry struct {
	static   map[string]NodeEndpoint // exact node name -> endpoint
	template string
	pool     map[string]NodeEndpoint // lowercased node name -> endpoint

	resolver NodeResolver

	mu       sync.RWMutex
	resolved map[string]NodeEndpoint // exact node name -> endpoint; nil until the first successful refresh

	done      chan struct{}
	closeOnce sync.Once
}

// newRegistry builds a registry from cfg's configured sources and poolURLs,
// the base URLs behind the delegate balancer (used for pool-name matching).
// If cfg.resolver is set, its refresh goroutine is started here and must be
// stopped via registry.Close.
func newRegistry(cfg *nodeAffinityConfig, poolURLs []string) *registry {
	r := &registry{
		static:   make(map[string]NodeEndpoint, len(cfg.endpoints)),
		template: cfg.template,
		pool:     poolMatch(poolURLs),
		resolver: cfg.resolver,
	}

	for node, u := range cfg.endpoints {
		r.static[node] = NodeEndpoint{Node: node, BaseURL: normalizeBaseURL(u)}
	}

	if r.resolver != nil {
		refresh := cfg.resolverRefresh
		if refresh <= 0 {
			refresh = defaultResolverRefresh
		}

		r.done = make(chan struct{})

		go r.refreshLoop(refresh)
	}

	return r
}

// refreshLoop refreshes the resolved map immediately, then on every tick of
// interval, until Close. It runs on its own goroutine, started by
// newRegistry.
func (r *registry) refreshLoop(interval time.Duration) {
	r.refresh()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.refresh()
		case <-r.done:
			return
		}
	}
}

// refresh fetches the resolver's current node list and swaps it in. A
// failed fetch keeps the previous map and logs, matching how resty's own
// SRV balancer handles a failed lookup (load_balancer.go, swrr.ticker); the
// lock is never held across the resolver call itself.
func (r *registry) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), resolverFetchTimeout)
	defer cancel()

	eps, err := r.resolver.Nodes(ctx)
	if err != nil {
		log.Printf("proxmox: node affinity: resolver refresh failed, keeping previous endpoint map: %v", err)

		return
	}

	m := make(map[string]NodeEndpoint, len(eps))
	for _, ep := range eps {
		m[ep.Node] = NodeEndpoint{Node: ep.Node, BaseURL: normalizeBaseURL(ep.BaseURL)}
	}

	r.mu.Lock()
	r.resolved = m
	r.mu.Unlock()
}

// Close stops the resolver refresh goroutine, if one was started. Safe to
// call more than once, and safe to call when no resolver was configured.
func (r *registry) Close() error {
	if r.done != nil {
		r.closeOnce.Do(func() { close(r.done) })
	}

	return nil
}

// lookup resolves node to its endpoint, consulting sources in precedence
// order. The bool result is false only when every source misses.
func (r *registry) lookup(node string) (NodeEndpoint, bool) {
	if ep, ok := r.static[node]; ok {
		return ep, true
	}

	if r.template != "" {
		return NodeEndpoint{
			Node:    node,
			BaseURL: normalizeBaseURL(strings.ReplaceAll(r.template, "{node}", node)),
		}, true
	}

	if r.resolver != nil {
		r.mu.RLock()
		ep, ok := r.resolved[node]
		r.mu.RUnlock()

		if ok {
			return ep, true
		}
	}

	if ep, ok := r.pool[strings.ToLower(node)]; ok {
		return ep, true
	}

	return NodeEndpoint{}, false
}

// poolMatch derives a node-name -> endpoint map from base URLs already
// configured on the delegate balancer, matching case-insensitively against
// the full host and its first DNS label (e.g. https://pve1.example.com:8006
// matches "pve1.example.com" and "pve1"). A bare IP address never matches:
// there is nothing to match on.
func poolMatch(urls []string) map[string]NodeEndpoint {
	m := make(map[string]NodeEndpoint, len(urls)*2)

	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}

		host := u.Hostname()
		if host == "" || net.ParseIP(host) != nil {
			continue
		}

		base := normalizeBaseURL(raw)

		label := host
		if before, _, ok := strings.Cut(host, "."); ok {
			label = before
		}

		m[strings.ToLower(host)] = NodeEndpoint{Node: host, BaseURL: base}
		m[strings.ToLower(label)] = NodeEndpoint{Node: label, BaseURL: base}
	}

	return m
}

// normalizeBaseURL strips the path and query from raw, matching resty's own
// extractBaseURL (load_balancer.go), so an endpoint supplied with a path
// (e.g. a static map entry copy-pasted with /api2/json) still matches the
// bare base URLs resty's balancers key on.
func normalizeBaseURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return strings.TrimRight(raw, "/")
	}

	u.Path = ""
	u.RawQuery = ""

	return strings.TrimRight(u.String(), "/")
}
