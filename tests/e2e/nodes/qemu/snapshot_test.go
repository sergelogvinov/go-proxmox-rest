//go:build e2e

package qemu_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestQemuSnapshotListAbsent verifies that GET
// /nodes/{node}/qemu/{vmid}/snapshot on a nonexistent guest returns an
// error.
func TestQemuSnapshotListAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu snapshot tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).Qemu().Snapshot().List(ctx, nonexistentVMID)
	e2e.RequireError(t, "list snapshots of absent guest", err)
}

// TestQemuSnapshotListOpportunistic verifies GET
// /nodes/{node}/qemu/{vmid}/snapshot decodes without error against a
// real guest, if cfg.Node happens to host one, and always includes the
// "current" pseudo-entry. Read-only and safe.
func TestQemuSnapshotListOpportunistic(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu snapshot tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	resources, err := client.Cluster().Resources().Get(ctx, cluster.ResourceTypeVM)
	e2e.RequireNoError(t, "list cluster vm resources", err)

	var vmid int
	for _, r := range resources {
		if r.Node == cfg.Node && r.Type == "qemu" {
			vmid = r.VMID
			break
		}
	}
	if vmid == 0 {
		t.Skip("no QEMU guest found on PVE_E2E_NODE; skipping opportunistic snapshot check")
	}

	snapshots, err := client.Nodes(cfg.Node).Qemu().Snapshot().List(ctx, vmid)
	e2e.RequireNoError(t, "list snapshots of real guest", err)

	var sawCurrent bool
	for _, s := range snapshots {
		if s.Name == "current" {
			sawCurrent = true
		}
	}
	if !sawCurrent {
		t.Errorf("list: no \"current\" pseudo-entry in %+v", snapshots)
	}
}

// TestQemuSnapshotCreateValidation verifies client-side validation of
// nil options and a missing snapname, without making a request.
func TestQemuSnapshotCreateValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu snapshot tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).Qemu().Snapshot().Create(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "create snapshot with nil options", err)

	_, err = client.Nodes(cfg.Node).Qemu().Snapshot().Create(ctx, nonexistentVMID, &qemu.CreateSnapshotOptions{})
	e2e.RequireError(t, "create snapshot without snapname", err)
}
