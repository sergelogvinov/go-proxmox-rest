//go:build e2e

// Package nodes_e2e exercises the per-node module against a live Proxmox VE
// cluster: version, runtime status, system log, and persistent
// configuration (get → update(Description) → get → restore).
package nodes_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/nodes"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestNodeVersion verifies GET /nodes/{node}/version decodes without error
// and reports a non-empty version.
func TestNodeVersion(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping node tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	v, err := client.Nodes().Version(ctx, cfg.Node)
	e2e.RequireNoError(t, "node version", err)

	if v.Version == "" {
		t.Errorf("version: Version is empty, want a version string")
	}
}

// TestNodeStatus verifies GET /nodes/{node}/status decodes without error
// and reports non-zero CPU/uptime information.
func TestNodeStatus(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping node tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	status, err := client.Nodes().Status(ctx, cfg.Node)
	e2e.RequireNoError(t, "node status", err)

	if status.Uptime <= 0 {
		t.Errorf("status: Uptime = %d, want > 0", status.Uptime)
	}
	if status.CPUInfo == nil || status.CPUInfo.Cores <= 0 {
		t.Errorf("status: CPUInfo = %+v, want Cores > 0", status.CPUInfo)
	}
	if status.PVEVersion == "" {
		t.Errorf("status: PVEVersion is empty, want a version string")
	}
}

// TestNodeSyslog verifies GET /nodes/{node}/syslog decodes without error
// both unfiltered and with a limit.
func TestNodeSyslog(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping node tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	nc := client.Nodes()
	ctx := t.Context()

	entries, err := nc.Syslog(ctx, cfg.Node, nil)
	e2e.RequireNoError(t, "syslog (unfiltered)", err)
	for _, e := range entries {
		if e.N == 0 || e.T == "" {
			t.Errorf("syslog: entry with empty N/T: %+v", e)
		}
	}

	limited, err := nc.Syslog(ctx, cfg.Node, &nodes.SyslogOptions{Limit: 1})
	e2e.RequireNoError(t, "syslog (limit=1)", err)
	if len(limited) > 1 {
		t.Errorf("syslog (limit=1): got %d entries, want at most 1", len(limited))
	}
}

// TestNodeConfigLifecycle verifies GET/PUT /nodes/{node}/config by setting a
// uniquely-named Description and restoring the original value (or clearing
// it via Delete, if it was unset) afterwards, so the test never leaves the
// node's persistent configuration altered.
func TestNodeConfigLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping node tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	nc := client.Nodes()
	ctx := t.Context()

	// 1. get — baseline, to restore afterwards.
	before, err := nc.Config(ctx, cfg.Node)
	e2e.RequireNoError(t, "get node config", err)

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			restore := &nodes.Config{Description: before.Description}
			if before.Description == "" {
				restore.Delete = []string{"description"}
			}

			e2e.RetryCleanup(t, "restore node config", func() error {
				return nc.UpdateConfig(cleanupCtx, cfg.Node, restore)
			})
		})
	}

	// 2. update — set a uniquely-named description.
	description := "go-proxmox-rest " + e2e.UniqueName(cfg.Prefix)
	err = nc.UpdateConfig(ctx, cfg.Node, &nodes.Config{Description: description})
	e2e.RequireNoError(t, "update node config", err)

	// 3. get — verify the description was applied. Proxmox stores
	// Description as "#"-prefixed comment lines and always appends a
	// trailing "\n" when reassembling them (PVE::JSONSchema::parse_config),
	// so a single-line description round-trips with one trailing newline.
	after, err := nc.Config(ctx, cfg.Node)
	e2e.RequireNoError(t, "get node config after update", err)

	want := description + "\n"
	if after.Description != want {
		t.Errorf("config after update: Description = %q, want %q", after.Description, want)
	}
}
