//go:build e2e

package qemu_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestQemuResizeValidation verifies client-side validation of nil
// options and a missing disk/size, without making a request.
func TestQemuResizeValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu resize tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes().Qemu().Resize(ctx, cfg.Node, nonexistentVMID, nil)
	e2e.RequireError(t, "resize with nil options", err)

	_, err = client.Nodes().Qemu().Resize(ctx, cfg.Node, nonexistentVMID, &qemu.ResizeOptions{})
	e2e.RequireError(t, "resize without disk or size", err)

	_, err = client.Nodes().Qemu().Resize(ctx, cfg.Node, nonexistentVMID, &qemu.ResizeOptions{Disk: "scsi0"})
	e2e.RequireError(t, "resize without size", err)
}
