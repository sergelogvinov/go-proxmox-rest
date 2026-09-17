//go:build e2e

package lxc_e2e

import (
	"testing"

	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestLXCTemplateAbsent verifies that converting a nonexistent container
// to a template returns an error. Template is never exercised against a
// real container here: it's a one-way conversion this suite cannot
// safely undo.
func TestLXCTemplateAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc template tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	err := client.Nodes(cfg.Node).LXC().Template(ctx, nonexistentVMID)
	e2e.RequireError(t, "template absent container", err)
}
