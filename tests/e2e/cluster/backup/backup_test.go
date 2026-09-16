//go:build e2e

// Package backup_e2e exercises the full CRUD lifecycle of the cluster
// vzdump backup job module against a live Proxmox VE cluster:
//
//	list → get(absent) → create → get → update → get → delete → list
package backup_e2e

import (
	"slices"
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster/backup"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestBackupLifecycle runs the full CRUD lifecycle for a uniquely-named
// vzdump backup job. The job is created disabled (Enabled: false) and
// targets All guests so it never depends on — or touches — real guests,
// and the scheduler never actually runs it during the test.
func TestBackupLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	bc := client.Cluster().Backup()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)

	// Cleanup registry: delete the job even if the test fails mid-way,
	// unless the caller asked to keep resources for debugging.
	//
	// Checks existence via Get before calling Delete, rather than just
	// letting RetryCleanup's alreadyGone handle a redundant delete like
	// every other cleanup in this suite: PVE::API2::Backup::delete_job
	// re-raises its "no such job" exception through a bare `die "$@"`
	// inside a cfs_lock_file callback, which stringifies (and loses the
	// HTTP-400 typing of) the original PVE::Exception::Param — so
	// deleting an already-absent backup job surfaces here as a generic
	// HTTP 500 ("500 400 Parameter verification failed", with the
	// "no such job ..." detail truncated out of the mangled status
	// line) instead of the clean 400 alreadyGone recognizes. read_job
	// has no such bug (its raise_param_exc is not re-thrown through
	// cfs_lock_file), so Get's error is reliably matchable.
	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete backup job", func() error {
				if _, err := bc.Get(cleanupCtx, name); err != nil {
					return err
				}
				return bc.Delete(cleanupCtx, name)
			})
		})
	}

	// 1. list — baseline: our job must not exist yet.
	listed, err := bc.List(ctx)
	e2e.RequireNoError(t, "list backup jobs", err)
	if containsBackupJob(listed, name) {
		t.Fatalf("list: backup job %q already exists before create", name)
	}

	// 2. get (absent) — a non-existent job must return an error.
	_, err = bc.Get(ctx, name)
	e2e.RequireError(t, "get absent backup job", err)

	// 3. create — a uniquely-named, disabled job covering all guests.
	schedule := "sat 03:00"
	enabled := false
	all := true
	comment := "e2e lifecycle"
	err = bc.Create(ctx, &backup.JobOptions{
		ID:       name,
		Schedule: &schedule,
		Enabled:  &enabled,
		All:      &all,
		Comment:  &comment,
	})
	e2e.RequireNoError(t, "create backup job", err)

	// 4. get — verify the create.
	job, err := bc.Get(ctx, name)
	e2e.RequireNoError(t, "get backup job after create", err)
	if job.ID != name {
		t.Errorf("get: ID = %q, want %q", job.ID, name)
	}
	if job.Schedule != schedule {
		t.Errorf("get: Schedule = %q, want %q", job.Schedule, schedule)
	}
	if job.Enabled {
		t.Errorf("get: Enabled = true, want false")
	}
	if !job.All {
		t.Errorf("get: All = false, want true")
	}
	if job.Comment != comment {
		t.Errorf("get: Comment = %q, want %q", job.Comment, comment)
	}

	// 5. update — mutate the comment and schedule; the job stays disabled.
	newSchedule := "sun 04:00"
	newComment := "e2e updated"
	err = bc.Update(ctx, name, &backup.JobOptions{
		Schedule: &newSchedule,
		Comment:  &newComment,
	})
	e2e.RequireNoError(t, "update backup job", err)

	// 6. get — verify the update; All must be unchanged.
	job, err = bc.Get(ctx, name)
	e2e.RequireNoError(t, "get backup job after update", err)
	if job.Schedule != newSchedule {
		t.Errorf("get after update: Schedule = %q, want %q", job.Schedule, newSchedule)
	}
	if job.Comment != newComment {
		t.Errorf("get after update: Comment = %q, want %q", job.Comment, newComment)
	}
	if !job.All {
		t.Errorf("get after update: All = false, want unchanged true")
	}

	// 7. delete — remove the job.
	err = bc.Delete(ctx, name)
	e2e.RequireNoError(t, "delete backup job", err)

	// 8. list — verify the delete.
	listed, err = bc.List(ctx)
	e2e.RequireNoError(t, "list backup jobs after delete", err)
	if containsBackupJob(listed, name) {
		t.Errorf("list after delete: backup job %q still present", name)
	}

	// get after delete must fail again.
	_, err = bc.Get(ctx, name)
	e2e.RequireError(t, "get backup job after delete", err)
}

// TestBackupValidation verifies client-side validation of Create options,
// none of which should reach the API.
func TestBackupValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	bc := client.Cluster().Backup()
	ctx := t.Context()

	err := bc.Create(ctx, nil)
	e2e.RequireError(t, "create with nil options", err)

	schedule := "sat 03:00"
	all := true

	err = bc.Create(ctx, &backup.JobOptions{Schedule: &schedule, All: &all})
	e2e.RequireError(t, "create with missing id", err)

	err = bc.Create(ctx, &backup.JobOptions{ID: "does-not-matter", All: &all})
	e2e.RequireError(t, "create with missing schedule", err)
}

// containsBackupJob reports whether the slice contains a backup job with
// the given ID.
func containsBackupJob(list []backup.Job, id string) bool {
	return slices.ContainsFunc(list, func(j backup.Job) bool { return j.ID == id })
}
