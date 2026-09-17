//go:build e2e

package lxc_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestLXCCloneValidation verifies client-side validation of nil options
// and a missing newid, without making a request.
func TestLXCCloneValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc clone tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).LXC().Clone(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "clone with nil options", err)

	_, err = client.Nodes(cfg.Node).LXC().Clone(ctx, nonexistentVMID, &lxc.CloneOptions{})
	e2e.RequireError(t, "clone without newid", err)
}

// TestLXCCloneAbsentSource verifies that cloning a nonexistent source
// container returns an error. Clone is never exercised against a real
// container here: a successful clone creates a brand-new container, and
// this client has no CT-destroy method yet to clean it up afterwards.
func TestLXCCloneAbsentSource(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc clone tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).LXC().Clone(ctx, nonexistentVMID, &lxc.CloneOptions{
		NewID: nonexistentVMID - 1,
	})
	e2e.RequireError(t, "clone absent source container", err)
}
