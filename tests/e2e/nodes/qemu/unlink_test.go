//go:build e2e

package qemu_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestQemuUnlinkValidation verifies client-side validation of nil
// options and an empty id list, without making a request.
func TestQemuUnlinkValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu unlink tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	err := client.Nodes(cfg.Node).Qemu().Unlink(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "unlink with nil options", err)

	err = client.Nodes(cfg.Node).Qemu().Unlink(ctx, nonexistentVMID, &qemu.UnlinkOptions{})
	e2e.RequireError(t, "unlink without idlist", err)
}

// TestQemuUnlinkAbsentSource verifies that unlinking a disk on a
// nonexistent guest returns an error. Unlink is never exercised against
// a real guest here: it deletes real disk images, which this suite has
// no business doing against whatever cluster PVE_E2E_URL happens to
// point at.
func TestQemuUnlinkAbsentSource(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu unlink tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	err := client.Nodes(cfg.Node).Qemu().Unlink(ctx, nonexistentVMID, &qemu.UnlinkOptions{
		IDList: []string{"unused0"},
	})
	e2e.RequireError(t, "unlink of absent guest", err)
}
