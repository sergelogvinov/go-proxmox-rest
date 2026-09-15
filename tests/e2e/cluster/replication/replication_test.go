//go:build e2e

// Package replication_e2e exercises the cluster replication module against
// a live Proxmox VE cluster.
//
// A full CRUD lifecycle needs a real guest with at least one replicatable
// (ZFS-backed) volume: Create checks that the guest and target node exist,
// that the caller's node matches the guest's current node, and that the
// guest actually has replicatable volumes, all before writing the job. That
// precondition isn't something this suite can safely fabricate, so these
// tests are limited to what's always safe: listing, a get on a
// syntactically valid but nonexistent job id, and client-side validation.
package replication_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster/replication"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestReplicationList verifies GET /cluster/replication decodes without
// error.
func TestReplicationList(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Cluster().Replication().List(ctx)
	e2e.RequireNoError(t, "list replication jobs", err)
}

// TestReplicationGetAbsent verifies that a syntactically valid but
// nonexistent job id returns an error.
func TestReplicationGetAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Cluster().Replication().Get(ctx, "999999999-0")
	e2e.RequireError(t, "get absent replication job", err)
}

// TestReplicationValidation verifies client-side validation of
// Create/Update/Delete options, none of which should reach the API.
func TestReplicationValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	rc := client.Cluster().Replication()
	ctx := t.Context()

	err := rc.Create(ctx, nil)
	e2e.RequireError(t, "create with nil options", err)

	err = rc.Create(ctx, &replication.JobOptions{Type: replication.TypeLocal, Target: "does-not-matter"})
	e2e.RequireError(t, "create with missing id", err)

	err = rc.Create(ctx, &replication.JobOptions{ID: "999999999-0", Target: "does-not-matter"})
	e2e.RequireError(t, "create with missing type", err)

	err = rc.Create(ctx, &replication.JobOptions{ID: "999999999-0", Type: replication.TypeLocal})
	e2e.RequireError(t, "create with missing target", err)

	err = rc.Update(ctx, "999999999-0", nil)
	e2e.RequireError(t, "update with nil options", err)

	err = rc.Update(ctx, "999999999-0", &replication.JobOptions{})
	e2e.RequireError(t, "update with missing target", err)

	err = rc.Delete(ctx, "999999999-0", true, true)
	e2e.RequireError(t, "delete with keep and force both set", err)
}
