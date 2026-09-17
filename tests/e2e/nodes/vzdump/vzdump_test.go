//go:build e2e

// Package vzdump_e2e exercises the per-node backup module against a live
// Proxmox VE cluster.
//
// Create is only exercised for its client-side validation (nil options,
// neither VMID nor All set) — never against real guests. Actually
// running a backup job is expensive (spins up a real vzdump task,
// consumes backup storage) and this suite has no safe, self-contained
// guest fixture to scope it to, matching the caution already applied to
// every other job/action-launching endpoint in this client (qemu.Clone,
// ceph.Start/Stop/Restart, ...).
package vzdump_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/vzdump"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestVZDumpCreateValidation verifies client-side validation of nil
// options and a request with neither VMID nor All set, without making a
// request.
func TestVZDumpCreateValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping vzdump tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).VZDump().Create(ctx, nil)
	e2e.RequireError(t, "create backup with nil options", err)

	_, err = client.Nodes(cfg.Node).VZDump().Create(ctx, &vzdump.Options{})
	e2e.RequireError(t, "create backup without vmid or all", err)
}

// TestVZDumpDefaults verifies GET /nodes/{node}/vzdump/defaults decodes
// without error and resolves to a usable backup target (Proxmox falls
// back to the "local" storage when neither storage nor dumpdir is
// configured).
func TestVZDumpDefaults(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping vzdump tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	defaults, err := client.Nodes(cfg.Node).VZDump().Defaults(ctx, "")
	e2e.RequireNoError(t, "vzdump defaults", err)
	if defaults.Storage == "" && defaults.DumpDir == "" {
		t.Errorf("defaults: both Storage and DumpDir are empty, want at least one set")
	}
}

// TestVZDumpExtractConfigAbsent verifies that extracting a config from a
// nonexistent backup volume returns an error.
func TestVZDumpExtractConfigAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping vzdump tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).VZDump().ExtractConfig(ctx, "local:backup/vzdump-qemu-999999999-2000_01_01-00_00_00.vma.zst")
	e2e.RequireError(t, "extract config of absent backup volume", err)
}
