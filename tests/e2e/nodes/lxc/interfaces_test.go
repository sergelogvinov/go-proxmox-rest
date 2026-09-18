//go:build e2e

package lxc_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestLXCInterfacesAbsent verifies that GET
// /nodes/{node}/lxc/{vmid}/interfaces on a nonexistent container returns
// an error.
func TestLXCInterfacesAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc interfaces tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).LXC().Interfaces(ctx, nonexistentVMID)
	e2e.RequireError(t, "interfaces of absent container", err)
}

// TestLXCInterfacesOpportunistic lists a real, running container's
// network interfaces, if cfg.Node happens to host one — this endpoint
// reads the addresses out of a running container's network namespace, so
// a stopped container returns none. It never writes anything.
func TestLXCInterfacesOpportunistic(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc interfaces tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	resources, err := client.Cluster().Resources().List(ctx, cluster.ListFilter{Type: cluster.ResourceTypeVM})
	e2e.RequireNoError(t, "list cluster vm resources", err)

	var vmid int
	for _, r := range resources {
		if r.Node == cfg.Node && r.Type == "lxc" && r.Status == "running" {
			vmid = r.VMID
			break
		}
	}
	if vmid == 0 {
		t.Skip("no running LXC container found on PVE_E2E_NODE; skipping opportunistic interfaces check")
	}

	ifaces, err := client.Nodes(cfg.Node).LXC().Interfaces(ctx, vmid)
	e2e.RequireNoError(t, "get real container interfaces", err)
	if len(ifaces) == 0 {
		t.Errorf("interfaces: got none for a running container")
	}
}
