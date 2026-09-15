//go:build e2e

// Package capabilities_e2e exercises the read-only node capabilities
// module against a live Proxmox VE cluster: QEMU CPU models/flags,
// machine types, and migration capabilities.
package capabilities_e2e

import (
	"testing"

	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestCapabilitiesQemu verifies every /nodes/{node}/capabilities/qemu/*
// endpoint decodes without error.
func TestCapabilitiesQemu(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping capabilities tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	qemu := client.Nodes().Capabilities().Qemu()
	ctx := t.Context()

	models, err := qemu.CPUModels(ctx, cfg.Node, "")
	e2e.RequireNoError(t, "cpu models", err)
	if len(models) == 0 {
		t.Errorf("cpu models: got no entries, want at least the built-in models")
	}
	for _, m := range models {
		if m.Name == "" {
			t.Errorf("cpu models: entry with empty Name: %+v", m)
		}
	}

	// The host's own architecture may report zero flags (e.g.
	// aarch64), so only check that decoding succeeds.
	_, err = qemu.CPUFlags(ctx, cfg.Node, "", "")
	e2e.RequireNoError(t, "cpu flags", err)

	machines, err := qemu.Machines(ctx, cfg.Node, "")
	e2e.RequireNoError(t, "machines", err)
	for _, m := range machines {
		if m.ID == "" {
			t.Errorf("machines: entry with empty ID: %+v", m)
		}
	}

	_, err = qemu.Migration(ctx, cfg.Node)
	e2e.RequireNoError(t, "migration capabilities", err)
}
