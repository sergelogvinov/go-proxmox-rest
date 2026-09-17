//go:build e2e

package lxc_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestLXCFeatureAbsent verifies that GET
// /nodes/{node}/lxc/{vmid}/feature on a nonexistent container returns an
// error.
func TestLXCFeatureAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc feature tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).LXC().Feature(ctx, nonexistentVMID, lxc.FeatureClone, "")
	e2e.RequireError(t, "feature check of absent container", err)
}

// TestLXCFeatureOpportunistic checks the "clone" and "snapshot" features
// for a real container, if cfg.Node happens to host one. Read-only and
// safe: it never mutates anything.
func TestLXCFeatureOpportunistic(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc feature tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	resources, err := client.Cluster().Resources().Get(ctx, cluster.ResourceTypeVM)
	e2e.RequireNoError(t, "list cluster vm resources", err)

	var vmid int
	for _, r := range resources {
		if r.Node == cfg.Node && r.Type == "lxc" {
			vmid = r.VMID
			break
		}
	}
	if vmid == 0 {
		t.Skip("no LXC container found on PVE_E2E_NODE; skipping opportunistic feature check")
	}

	lc := client.Nodes(cfg.Node).LXC()

	_, err = lc.Feature(ctx, vmid, lxc.FeatureClone, "")
	e2e.RequireNoError(t, "feature check (clone) of real container", err)

	_, err = lc.Feature(ctx, vmid, lxc.FeatureSnapshot, "")
	e2e.RequireNoError(t, "feature check (snapshot) of real container", err)

	_, err = lc.Feature(ctx, vmid, lxc.FeatureCopy, "")
	e2e.RequireNoError(t, "feature check (copy) of real container", err)
}
