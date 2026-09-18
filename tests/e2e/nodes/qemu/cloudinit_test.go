//go:build e2e

package qemu_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestQemuCloudInitPendingAbsent verifies that GET
// /nodes/{node}/qemu/{vmid}/cloudinit on a nonexistent guest returns an
// error.
func TestQemuCloudInitPendingAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu cloudinit tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).Qemu().CloudInitPending(ctx, nonexistentVMID)
	e2e.RequireError(t, "cloudinit pending of absent guest", err)
}

// TestQemuCloudInitPendingOpportunistic decodes a real guest's
// cloud-init pending state, if cfg.Node happens to host one. Read-only
// and safe: it never regenerates anything.
func TestQemuCloudInitPendingOpportunistic(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu cloudinit tests")
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
		t.Skip("no QEMU guest found on PVE_E2E_NODE; skipping opportunistic cloudinit check")
	}

	// A guest without any cloud-init drive configured simply returns an
	// empty list here — no error either way, so this only confirms the
	// response decodes without error.
	_, err = client.Nodes(cfg.Node).Qemu().CloudInitPending(ctx, vmid)
	e2e.RequireNoError(t, "cloudinit pending of real guest", err)
}

// TestQemuCloudInitUpdateAbsent verifies that regenerating the
// cloud-init drive of a nonexistent guest returns an error.
// CloudInitUpdate is never exercised against a real guest here: it
// rewrites the guest's actual cloud-init config drive image, which this
// suite has no business doing against whatever cluster PVE_E2E_URL
// happens to point at.
func TestQemuCloudInitUpdateAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu cloudinit tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	err := client.Nodes(cfg.Node).Qemu().CloudInitUpdate(ctx, nonexistentVMID)
	e2e.RequireError(t, "cloudinit update of absent guest", err)
}
