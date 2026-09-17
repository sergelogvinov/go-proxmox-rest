//go:build e2e

// Package storage_e2e exercises the per-node storage status/content/
// prune-backups module against a live Proxmox VE cluster.
//
// Each test creates its own uniquely-named "dir" storage (backed by a
// temporary path under /tmp, scoped to PVE_E2E_NODE) via the root storage
// module, then exercises the per-node view against it. Creating storages
// needs root-level privileges (PVEAuditor is not enough); use a
// full-privilege token or user credentials.
//
// PruneBackups.Delete is only exercised for its client-side validation
// (nil/empty retention): actually pruning is destructive to real backup
// history and this suite has no safe, self-contained backup fixture to
// run it against.
package storage_e2e

import (
	"slices"
	"strings"
	"testing"

	proxmox "github.com/sergelogvinov/go-proxmox-rest"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/storage"
	rootstorage "github.com/sergelogvinov/go-proxmox-rest/storage"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// newDirStorage creates a uniquely-named "dir" storage scoped to
// cfg.Node, backed by a temporary path, and registers its cleanup.
func newDirStorage(t *testing.T, cfg *e2e.E2EConfig, client *proxmox.Client, content []string) string {
	t.Helper()

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

	_, err := sc.Create(ctx, &rootstorage.Options{
		ID:      name,
		Type:    "dir",
		Content: content,
		Path:    &path,
		Nodes:   []string{cfg.Node},
	})
	e2e.RequireNoError(t, "create storage", err)

	return name
}

// TestNodeStorageList verifies GET /nodes/{node}/storage decodes without
// error and includes a freshly created storage with the expected fields.
func TestNodeStorageList(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping node storage tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	name := newDirStorage(t, cfg, client, []string{"iso"})
	ctx := t.Context()

	list, err := client.Nodes(cfg.Node).Storage().List(ctx, nil)
	e2e.RequireNoError(t, "list node storages", err)

	idx := slices.IndexFunc(list, func(s storage.Storage) bool { return s.Storage == name })
	if idx < 0 {
		t.Fatalf("list: storage %q missing from result", name)
	}
	got := list[idx]
	if got.Type != "dir" {
		t.Errorf("list: Type = %q, want %q", got.Type, "dir")
	}
	if !got.Enabled {
		t.Errorf("list: Enabled = false, want true")
	}
	if !slices.Contains(got.Content, "iso") {
		t.Errorf("list: Content = %v, want it to contain %q", got.Content, "iso")
	}

	// The storage filter must narrow the result to just this storage.
	filtered, err := client.Nodes(cfg.Node).Storage().List(ctx, &storage.ListOptions{Storage: name})
	e2e.RequireNoError(t, "list node storages filtered by storage", err)
	if len(filtered) != 1 || filtered[0].Storage != name {
		t.Errorf("list filtered by storage=%q: got %+v", name, filtered)
	}
}

// TestNodeStorageStatus verifies GET
// /nodes/{node}/storage/{storage}/status decodes without error and
// manually fills in the storage id (which Proxmox's own response omits).
func TestNodeStorageStatus(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping node storage tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	name := newDirStorage(t, cfg, client, []string{"iso"})
	ctx := t.Context()

	st, err := client.Nodes(cfg.Node).Storage().Status(ctx, name)
	e2e.RequireNoError(t, "storage status", err)
	if st.Storage != name {
		t.Errorf("status: Storage = %q, want %q", st.Storage, name)
	}
	if st.Type != "dir" {
		t.Errorf("status: Type = %q, want %q", st.Type, "dir")
	}
	if !st.Active {
		t.Errorf("status: Active = false, want true")
	}
}

// TestNodeStorageContentLifecycle runs the full CRUD lifecycle for a
// volume allocated on a uniquely-named dir storage:
//
//	list → create → list → get → update(rejected) → delete → list
//
// Update is expected to fail here: notes/protected are only supported
// for backup-type volumes (PVE::Storage::DirPlugin dies "only backups
// can have notes"/"only backups support attribute 'protected'" for a
// disk image), and this suite has no safe way to produce a real backup
// volume to exercise the accepted path against.
func TestNodeStorageContentLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping node storage tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	name := newDirStorage(t, cfg, client, []string{"images"})
	ctx := t.Context()

	nc := client.Nodes(cfg.Node).Storage()
	const vmid = 999999999
	filename := "vm-999999999-disk-0.raw"

	// 1. list — baseline: no volumes yet.
	volumes, err := nc.Content().List(ctx, name, nil)
	e2e.RequireNoError(t, "list content (baseline)", err)
	if len(volumes) != 0 {
		t.Fatalf("list (baseline): got %d volumes, want 0: %+v", len(volumes), volumes)
	}

	// 2. create — allocate a small raw disk image.
	volid, err := nc.Content().Create(ctx, name, &storage.CreateVolumeOptions{
		Filename: filename,
		VMID:     vmid,
		Size:     "1024",
	})
	e2e.RequireNoError(t, "create volume", err)
	if !strings.HasPrefix(volid, name+":") {
		t.Fatalf("create: volid = %q, want prefix %q", volid, name+":")
	}
	volume := strings.TrimPrefix(volid, name+":")

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete volume", func() error {
				_, delErr := nc.Content().Delete(cleanupCtx, name, volume, 0)
				return delErr
			})
		})
	}

	// 3. list — verify the create.
	volumes, err = nc.Content().List(ctx, name, nil)
	e2e.RequireNoError(t, "list content after create", err)
	idx := slices.IndexFunc(volumes, func(v storage.Volume) bool { return v.VolID == volid })
	if idx < 0 {
		t.Fatalf("list after create: volume %q missing from result: %+v", volid, volumes)
	}
	if volumes[idx].VMID != vmid {
		t.Errorf("list after create: VMID = %d, want %d", volumes[idx].VMID, vmid)
	}
	if volumes[idx].Format != "raw" {
		t.Errorf("list after create: Format = %q, want %q", volumes[idx].Format, "raw")
	}

	// 4. get — verify the create from the single-volume view.
	got, err := nc.Content().Get(ctx, name, volume)
	e2e.RequireNoError(t, "get volume after create", err)
	if got.Format != "raw" {
		t.Errorf("get: Format = %q, want %q", got.Format, "raw")
	}
	if got.Size != 1024*1024 {
		t.Errorf("get: Size = %d, want %d", got.Size, 1024*1024)
	}

	// 5. update — notes/protected are only supported for backup-type
	// volumes (PVE::Storage::DirPlugin dies "only backups can have
	// notes"/"only backups support attribute 'protected'" for anything
	// else), so a disk image must reject both.
	notes := "e2e " + name
	err = nc.Content().Update(ctx, name, volume, &storage.UpdateVolumeOptions{Notes: &notes})
	e2e.RequireError(t, "update notes on a non-backup volume", err)

	protectedTrue := true
	err = nc.Content().Update(ctx, name, volume, &storage.UpdateVolumeOptions{Protected: &protectedTrue})
	e2e.RequireError(t, "update protected on a non-backup volume", err)

	// 6. delete.
	_, err = nc.Content().Delete(ctx, name, volume, 0)
	e2e.RequireNoError(t, "delete volume", err)

	// 7. list — verify the delete.
	volumes, err = nc.Content().List(ctx, name, nil)
	e2e.RequireNoError(t, "list content after delete", err)
	if slices.ContainsFunc(volumes, func(v storage.Volume) bool { return v.VolID == volid }) {
		t.Errorf("list after delete: volume %q still present", volid)
	}
}

// TestNodeStoragePruneBackupsDryRun verifies GET
// /nodes/{node}/storage/{storage}/prunebackups decodes without error on
// a storage with no backups, and that Delete validates its required
// retention parameter client-side without making a request.
//
// PVE::Storage::prune_backups requires a retention rule from somewhere —
// either the storage's own "prune-backups" config or an explicit
// PruneOptions.PruneBackups — dying "no prune-backups options configured
// for storage '...'" otherwise. This freshly created storage has no
// config for it, so DryRun below passes one explicitly rather than nil.
func TestNodeStoragePruneBackupsDryRun(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping node storage tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	name := newDirStorage(t, cfg, client, []string{"backup"})
	ctx := t.Context()

	entries, err := client.Nodes(cfg.Node).Storage().PruneBackups().DryRun(
		ctx, name, &storage.PruneOptions{PruneBackups: "keep-all=1"},
	)
	e2e.RequireNoError(t, "prune backups dry run", err)
	if len(entries) != 0 {
		t.Errorf("dry run: got %d entries on an empty storage, want 0: %+v", len(entries), entries)
	}

	// nil (no explicit retention, and the storage has none configured)
	// must fail server-side with the "no prune-backups options
	// configured" error above.
	_, err = client.Nodes(cfg.Node).Storage().PruneBackups().DryRun(ctx, name, nil)
	e2e.RequireError(t, "prune backups dry run without any retention configured", err)

	_, err = client.Nodes(cfg.Node).Storage().PruneBackups().Delete(ctx, name, nil)
	e2e.RequireError(t, "prune backups with nil options", err)

	_, err = client.Nodes(cfg.Node).Storage().PruneBackups().Delete(ctx, name, &storage.PruneOptions{})
	e2e.RequireError(t, "prune backups without retention", err)
}
