//go:build e2e

// Package pools_e2e exercises the full CRUD lifecycle of the pools
// module against a live Proxmox VE cluster:
//
//	list → get(absent) → create → get → update → get → delete → list
package pools_e2e

import (
	"slices"
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/pools"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestPoolsLifecycle runs the full CRUD lifecycle for a uniquely-named pool.
func TestPoolsLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	pc := client.Pools()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)

	// Cleanup registry: delete the pool even if the test fails mid-way,
	// unless the caller asked to keep resources for debugging.
	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			_ = pc.Delete(ctx, name, true)
		})
	}

	// 1. list — baseline: our pool must not exist yet.
	listed, err := pc.List(ctx)
	e2e.RequireNoError(t, "list pools", err)
	if containsPool(listed, name) {
		t.Fatalf("list: pool %q already exists before create", name)
	}

	// 2. get (absent) — a non-existent pool must return an error.
	_, err = pc.Get(ctx, name)
	e2e.RequireError(t, "get absent pool", err)

	// 3. create — a uniquely-named pool with a comment.
	comment := "e2e lifecycle"
	err = pc.Create(ctx, name, &pools.CreateOptions{Comment: comment})
	e2e.RequireNoError(t, "create pool", err)

	// 4. get — verify the create.
	pool, err := pc.Get(ctx, name)
	e2e.RequireNoError(t, "get pool after create", err)
	if pool.PoolID != name {
		t.Errorf("get: PoolID = %q, want %q", pool.PoolID, name)
	}
	if pool.Comment != comment {
		t.Errorf("get: Comment = %q, want %q", pool.Comment, comment)
	}

	// 5. update — mutate the comment.
	newComment := "e2e updated"
	err = pc.Update(ctx, name, &pools.UpdateOptions{Comment: &newComment})
	e2e.RequireNoError(t, "update pool", err)

	// 6. get — verify the update.
	pool, err = pc.Get(ctx, name)
	e2e.RequireNoError(t, "get pool after update", err)
	if pool.Comment != newComment {
		t.Errorf("get after update: Comment = %q, want %q", pool.Comment, newComment)
	}

	// 7. delete — remove the pool (it has no members, so no force needed).
	err = pc.Delete(ctx, name, false)
	e2e.RequireNoError(t, "delete pool", err)

	// 8. list — verify the delete.
	listed, err = pc.List(ctx)
	e2e.RequireNoError(t, "list pools after delete", err)
	if containsPool(listed, name) {
		t.Errorf("list after delete: pool %q still present", name)
	}

	// get after delete must fail again.
	_, err = pc.Get(ctx, name)
	e2e.RequireError(t, "get pool after delete", err)
}

// TestPoolsUpdateClearComment verifies that a pointer field set to an empty
// string clears the comment.
func TestPoolsUpdateClearComment(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	pc := client.Pools()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			_ = pc.Delete(ctx, name, true)
		})
	}

	err := pc.Create(ctx, name, &pools.CreateOptions{Comment: "e2e to-be-cleared"})
	e2e.RequireNoError(t, "create pool", err)

	empty := ""
	err = pc.Update(ctx, name, &pools.UpdateOptions{Comment: &empty})
	e2e.RequireNoError(t, "update pool (clear comment)", err)

	pool, err := pc.Get(ctx, name)
	e2e.RequireNoError(t, "get pool after clear", err)
	if pool.Comment != "" {
		t.Errorf("get after clear: Comment = %q, want empty", pool.Comment)
	}

	err = pc.Delete(ctx, name, false)
	e2e.RequireNoError(t, "delete pool", err)
}

// TestPoolsValidation verifies client-side validation of nil options.
func TestPoolsValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	pc := client.Pools()
	ctx := t.Context()

	err := pc.Create(ctx, "", nil)
	e2e.RequireError(t, "create with nil options", err)

	err = pc.Update(ctx, "does-not-matter", nil)
	e2e.RequireError(t, "update with nil options", err)
}

// containsPool reports whether the slice contains a pool with the given ID.
func containsPool(list []pools.Pool, id string) bool {
	return slices.ContainsFunc(list, func(p pools.Pool) bool { return p.PoolID == id })
}
