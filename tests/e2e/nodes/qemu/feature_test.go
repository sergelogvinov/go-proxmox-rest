//go:build e2e

package qemu_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestQemuFeatureAbsent verifies that GET
// /nodes/{node}/qemu/{vmid}/feature on a nonexistent guest returns an
// error.
func TestQemuFeatureAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu feature tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).Qemu().Feature(ctx, nonexistentVMID, qemu.FeatureClone, "")
	e2e.RequireError(t, "feature check of absent guest", err)
}

// TestQemuFeatureOpportunistic checks the "clone" and "snapshot"
// features for a real guest, if cfg.Node happens to host one. Read-only
// and safe: it never mutates anything.
func TestQemuFeatureOpportunistic(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu feature tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	resources, err := client.Cluster().Resources().List(ctx, cluster.ListFilter{Type: cluster.ResourceTypeVM})
	e2e.RequireNoError(t, "list cluster vm resources", err)

	var vmid int
	for _, r := range resources {
		if r.Node == cfg.Node && r.Type == "qemu" {
			vmid = r.VMID
			break
		}
	}
	if vmid == 0 {
		t.Skip("no QEMU guest found on PVE_E2E_NODE; skipping opportunistic feature check")
	}

	qc := client.Nodes(cfg.Node).Qemu()

	_, err = qc.Feature(ctx, vmid, qemu.FeatureClone, "")
	e2e.RequireNoError(t, "feature check (clone) of real guest", err)

	_, err = qc.Feature(ctx, vmid, qemu.FeatureSnapshot, "")
	e2e.RequireNoError(t, "feature check (snapshot) of real guest", err)

	_, err = qc.Feature(ctx, vmid, qemu.FeatureCopy, "")
	e2e.RequireNoError(t, "feature check (copy) of real guest", err)
}
