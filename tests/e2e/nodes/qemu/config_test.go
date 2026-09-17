//go:build e2e

package qemu_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestQemuConfigAbsent verifies that GET
// /nodes/{node}/qemu/{vmid}/config on a nonexistent guest returns an
// error.
func TestQemuConfigAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu config tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).Qemu().Config(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "config of absent guest", err)
}

// TestQemuConfigOpportunistic decodes a real guest's configuration, if
// cfg.Node happens to host one, verifying the numerically-indexed
// hardware families (net0, scsi0, ...) decode without error against
// production data rather than only synthetic fixtures. It never writes
// anything — Config is read-only.
func TestQemuConfigOpportunistic(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu config tests")
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
		t.Skip("no QEMU guest found on PVE_E2E_NODE; skipping opportunistic config check")
	}

	config, err := client.Nodes(cfg.Node).Qemu().Config(ctx, vmid, nil)
	e2e.RequireNoError(t, "get real guest config", err)
	if config.Digest == "" {
		t.Errorf("config: Digest is empty for a real guest")
	}
}

// TestQemuUpdateConfigAbsent verifies that both write paths
// (UpdateConfig/UpdateConfigAsync) reject nil options client-side and
// return an error against a nonexistent guest.
func TestQemuUpdateConfigAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu config tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	qc := client.Nodes(cfg.Node).Qemu()
	ctx := t.Context()

	err := qc.UpdateConfig(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "update config with nil options", err)

	_, err = qc.UpdateConfigAsync(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "update config async with nil options", err)

	err = qc.UpdateConfig(ctx, nonexistentVMID, &qemu.Config{Description: "e2e"})
	e2e.RequireError(t, "update config of absent guest", err)
}
