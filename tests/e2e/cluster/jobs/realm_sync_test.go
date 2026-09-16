//go:build e2e

package jobs_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster/jobs"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestRealmSyncList verifies GET /cluster/jobs/realm-sync decodes without
// error.
func TestRealmSyncList(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Cluster().Jobs().RealmSync().List(ctx)
	e2e.RequireNoError(t, "list realm-sync jobs", err)
}

// TestRealmSyncGetAbsent verifies that a nonexistent job id returns an
// error.
func TestRealmSyncGetAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)

	_, err := client.Cluster().Jobs().RealmSync().Get(ctx, name)
	e2e.RequireError(t, "get absent realm-sync job", err)
}

// TestRealmSyncCreateUnknownRealm verifies that Create rejects a realm
// that doesn't exist, before ever writing a job. A full create/update/
// delete lifecycle would need a real LDAP/AD realm configured on the test
// cluster, which this suite cannot assume exists, so this is as far as it
// safely goes without one.
func TestRealmSyncCreateUnknownRealm(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)
	scope := jobs.ScopeUsers
	schedule := "*/30"

	err := client.Cluster().Jobs().RealmSync().Create(ctx, name, &jobs.RealmSyncOptions{
		Realm:    name, // never a configured realm
		Scope:    &scope,
		Schedule: &schedule,
	})
	e2e.RequireError(t, "create realm-sync job with unknown realm", err)
}

// TestRealmSyncUpdateDeleteAbsent verifies that Update and Delete against
// a nonexistent job id return an error.
func TestRealmSyncUpdateDeleteAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	rs := client.Cluster().Jobs().RealmSync()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)
	comment := "e2e"

	err := rs.Update(ctx, name, &jobs.RealmSyncOptions{Comment: &comment})
	e2e.RequireError(t, "update absent realm-sync job", err)

	err = rs.Delete(ctx, name)
	e2e.RequireError(t, "delete absent realm-sync job", err)
}

// TestRealmSyncValidation verifies client-side validation of Create
// options, none of which should reach the API.
func TestRealmSyncValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	rs := client.Cluster().Jobs().RealmSync()
	ctx := t.Context()

	scope := jobs.ScopeUsers
	schedule := "*/30"

	err := rs.Create(ctx, "does-not-matter", nil)
	e2e.RequireError(t, "create with nil options", err)

	err = rs.Create(ctx, "", &jobs.RealmSyncOptions{Realm: "pam", Scope: &scope, Schedule: &schedule})
	e2e.RequireError(t, "create with missing id", err)

	err = rs.Create(ctx, "does-not-matter", &jobs.RealmSyncOptions{Scope: &scope, Schedule: &schedule})
	e2e.RequireError(t, "create with missing realm", err)

	err = rs.Create(ctx, "does-not-matter", &jobs.RealmSyncOptions{Realm: "pam", Schedule: &schedule})
	e2e.RequireError(t, "create with missing scope", err)

	err = rs.Create(ctx, "does-not-matter", &jobs.RealmSyncOptions{Realm: "pam", Scope: &scope})
	e2e.RequireError(t, "create with missing schedule", err)
}
