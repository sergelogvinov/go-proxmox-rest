//go:build e2e

package qemu_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestQemuCloneValidation verifies client-side validation of nil options
// and a missing newid, without making a request.
func TestQemuCloneValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu clone tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes().Qemu().Clone(ctx, cfg.Node, nonexistentVMID, nil)
	e2e.RequireError(t, "clone with nil options", err)

	_, err = client.Nodes().Qemu().Clone(ctx, cfg.Node, nonexistentVMID, &qemu.CloneOptions{})
	e2e.RequireError(t, "clone without newid", err)
}

// TestQemuCloneAbsentSource verifies that cloning a nonexistent source
// guest returns an error. Clone is never exercised against a real guest
// here: a successful clone creates a brand-new guest, and this client
// has no VM-destroy method yet to clean it up afterwards.
func TestQemuCloneAbsentSource(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu clone tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes().Qemu().Clone(ctx, cfg.Node, nonexistentVMID, &qemu.CloneOptions{
		NewID: nonexistentVMID - 1,
	})
	e2e.RequireError(t, "clone absent source guest", err)
}
