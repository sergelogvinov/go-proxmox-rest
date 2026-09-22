//go:build e2e

// Package storage_e2e exercises the full CRUD lifecycle of the storage
// module against a live Proxmox VE cluster:
//
//	list → get(absent) → create → get → update → get → delete → list
//
// Storage CRUD requires a writable /var/lib/vz-style directory on the
// node, so the tests create a "dir" plugin storage backed by a
// temporary path under /tmp. Creating storages needs root-level
// privileges (PVEAuditor is not enough); use a full-privilege token
// or user credentials.
package storage_e2e

import (
	"slices"
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/storage"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestStorageLifecycle runs the full CRUD lifecycle for a uniquely-named
// dir storage.
func TestStorageLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	sc := client.Storage()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)
	path := "/tmp/" + name

	// Cleanup registry: delete the storage even if the test fails
	// mid-way, unless the caller asked to keep resources for debugging.
	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete storage", func() error {
				return sc.Delete(cleanupCtx, name)
			})
		})
	}

	// 1. list — baseline: our storage must not exist yet.
	listed, err := sc.List(ctx, "")
	e2e.RequireNoError(t, "list storages", err)
	if containsStorage(listed, name) {
		t.Fatalf("list: storage %q already exists before create", name)
	}

	// 2. get (absent) — a non-existent storage must return an error.
	_, err = sc.Get(ctx, name)
	e2e.RequireError(t, "get absent storage", err)

	// 3. create — a uniquely-named dir storage.
	stCreated, err := sc.Create(ctx, &storage.Storage{
		ID:      name,
		Type:    "dir",
		Content: []string{"iso", "vztmpl"},
		Path:    &path,
	})
	e2e.RequireNoError(t, "create storage", err)

	// Verify that the created storage has the expected ID.
	if stCreated.ID != name {
		t.Errorf("create: ID = %q, want %q", stCreated.ID, name)
	}
	if stCreated.Type != "dir" {
		t.Errorf("create: Type = %q, want %q", stCreated.Type, "dir")
	}

	// 4. get — verify the create.
	st, err := sc.Get(ctx, name)
	e2e.RequireNoError(t, "get storage after create", err)
	if st.ID != name {
		t.Errorf("get: ID = %q, want %q", st.ID, name)
	}
	if st.Type != "dir" {
		t.Errorf("get: Type = %q, want %q", st.Type, "dir")
	}
	if st.Path == nil || *st.Path != path {
		t.Errorf("get: Path = %v, want %q", st.Path, path)
	}
	requireContent(t, "get", st.Content, []string{"iso", "vztmpl"})

	// 5. update — mutate the content list.
	newContent := []string{"iso", "backup"}
	stUpdated, err := sc.Update(ctx, name, &storage.Storage{Content: newContent})
	e2e.RequireNoError(t, "update storage", err)

	if stUpdated.ID != name {
		t.Errorf("update: ID = %q, want %q", stUpdated.ID, name)
	}

	// 6. get — verify the update.
	st, err = sc.Get(ctx, name)
	e2e.RequireNoError(t, "get storage after update", err)
	requireContent(t, "get after update", st.Content, []string{"backup", "iso"})

	// 7. delete — remove the storage (it holds no content).
	err = sc.Delete(ctx, name)
	e2e.RequireNoError(t, "delete storage", err)

	// 8. list — verify the delete.
	listed, err = sc.List(ctx, "")
	e2e.RequireNoError(t, "list storages after delete", err)
	if containsStorage(listed, name) {
		t.Errorf("list after delete: storage %q still present", name)
	}

	// get after delete must fail again.
	_, err = sc.Get(ctx, name)
	e2e.RequireError(t, "get storage after delete", err)
}

// TestStorageListTypeFilter verifies that the type filter narrows the
// result set to the requested plugin type.
func TestStorageListTypeFilter(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	sc := client.Storage()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)
	path := "/tmp/" + name

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete storage", func() error {
				return sc.Delete(cleanupCtx, name)
			})
		})
	}

	_, err := sc.Create(ctx, &storage.Storage{
		ID:      name,
		Type:    "dir",
		Content: []string{"iso"},
		Path:    &path,
	})
	e2e.RequireNoError(t, "create storage", err)

	// The filter must include our dir storage.
	dirs, err := sc.List(ctx, "dir")
	e2e.RequireNoError(t, "list storages filtered by type=dir", err)
	if !containsStorage(dirs, name) {
		t.Errorf("list type=dir: storage %q missing from result", name)
	}
	for _, st := range dirs {
		if st.Type != "dir" {
			t.Errorf("list type=dir: storage %q has Type %q, want %q", st.ID, st.Type, "dir")
		}
	}

	// A filter for an unrelated type must not include our storage.
	zfs, err := sc.List(ctx, "zfs")
	e2e.RequireNoError(t, "list storages filtered by type=zfs", err)
	if containsStorage(zfs, name) {
		t.Errorf("list type=zfs: dir storage %q unexpectedly present", name)
	}

	err = sc.Delete(ctx, name)
	e2e.RequireNoError(t, "delete storage", err)
}

// // TestStorageUpdateContent verifies that updating the content list
// // replaces the previous value.
func TestStorageUpdateContent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	sc := client.Storage()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)
	path := "/tmp/" + name

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete storage", func() error {
				return sc.Delete(cleanupCtx, name)
			})
		})
	}

	_, err := sc.Create(ctx, &storage.Storage{
		ID:      name,
		Type:    "dir",
		Content: []string{"iso"},
		Path:    &path,
	})
	e2e.RequireNoError(t, "create storage", err)

	newContent := []string{"vztmpl", "backup"}
	_, err = sc.Update(ctx, name, &storage.Storage{Content: newContent})
	e2e.RequireNoError(t, "update storage (content)", err)

	st, err := sc.Get(ctx, name)
	e2e.RequireNoError(t, "get storage after update", err)
	requireContent(t, "get after update", st.Content, []string{"backup", "vztmpl"})

	err = sc.Delete(ctx, name)
	e2e.RequireNoError(t, "delete storage", err)
}

// TestStorageValidation verifies client-side validation of nil options
// and missing required fields.
func TestStorageValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	sc := client.Storage()
	ctx := t.Context()

	_, err := sc.Create(ctx, nil)
	e2e.RequireError(t, "create with nil options", err)

	_, err = sc.Create(ctx, &storage.Storage{Type: "dir"})
	e2e.RequireError(t, "create without id", err)

	_, err = sc.Create(ctx, &storage.Storage{ID: "no-type"})
	e2e.RequireError(t, "create without type", err)

	_, err = sc.Update(ctx, "does-not-matter", nil)
	e2e.RequireError(t, "update with nil options", err)
}

// requireContent fails the test when got does not contain exactly the
// same elements as want, ignoring order.
func requireContent(t *testing.T, what string, got, want []string) {
	t.Helper()

	sorted := slices.Clone(got)
	slices.Sort(sorted)
	slices.Sort(want)

	if !slices.Equal(sorted, want) {
		t.Errorf("%s: Content = %v, want %v", what, got, want)
	}
}

// containsStorage reports whether the slice contains a storage with the
// given ID.
func containsStorage(list []storage.Storage, id string) bool {
	return slices.ContainsFunc(list, func(s storage.Storage) bool { return s.ID == id })
}
