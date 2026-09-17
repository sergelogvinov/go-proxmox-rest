//go:build e2e

package qemu_e2e

import (
	"testing"

	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestQemuTemplateAbsent verifies that converting a nonexistent guest to
// a template returns an error. Template is never exercised against a
// real guest here: it's a one-way conversion this suite cannot safely
// undo.
func TestQemuTemplateAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu template tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).Qemu().Template(ctx, nonexistentVMID, "")
	e2e.RequireError(t, "template absent guest", err)
}
