//go:build e2e

// Package replication_e2e exercises the per-node replication status module
// against a live Proxmox VE cluster.
//
// Job status/log only exist for a real, already-configured replication
// job, and ScheduleNow triggers an actual replication run — neither is
// something this suite can safely fabricate or invoke, so these tests are
// limited to what's always safe: listing, and a get/log on a syntactically
// valid but nonexistent job id.
package replication_e2e

import (
	"testing"

	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestReplicationList verifies GET /nodes/{node}/replication decodes
// without error, both unfiltered and filtered by guest.
func TestReplicationList(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping replication tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	rc := client.Nodes().Replication()
	ctx := t.Context()

	_, err := rc.List(ctx, cfg.Node, 0)
	e2e.RequireNoError(t, "list replication jobs (unfiltered)", err)

	// No guest is guaranteed to have replication jobs, so only assert
	// that the filtered call itself decodes without error.
	_, err = rc.List(ctx, cfg.Node, 999999999)
	e2e.RequireNoError(t, "list replication jobs (guest filter)", err)
}

// TestReplicationGetAbsent verifies that a syntactically valid but
// nonexistent job id returns an error from both Get and Log.
func TestReplicationGetAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping replication tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	rc := client.Nodes().Replication()
	ctx := t.Context()

	_, err := rc.Get(ctx, cfg.Node, "999999999-0")
	e2e.RequireError(t, "get absent replication job", err)

	_, err = rc.Log(ctx, cfg.Node, "999999999-0", nil)
	e2e.RequireError(t, "log absent replication job", err)
}
