//go:build e2e

package qemu_e2e

import (
	"testing"
	"time"

	proxmox "github.com/sergelogvinov/go-proxmox-rest"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/storage"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/tasks"
	rootstorage "github.com/sergelogvinov/go-proxmox-rest/storage"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// giB is one gibibyte, in bytes, matching how Storage().Content().List
// reports a volume's size.
const giB = 1024 * 1024 * 1024

// taskTimeout bounds how long this suite waits for the create/resize/
// delete tasks below, all of which touch a real (if tiny) disk image and
// so run well past tasks.WaitOptions' short default.
const taskTimeout = 2 * time.Minute

// newImagesStorage creates a uniquely-named "dir" storage, backed by a
// temporary path and scoped to cfg.Node, with "images" content enabled so
// a QEMU disk can be allocated on it. It registers its own cleanup.
func newImagesStorage(t *testing.T, cfg *e2e.E2EConfig, client *proxmox.Client) string {
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

	_, err := sc.Create(ctx, &rootstorage.Storage{
		ID:      name,
		Type:    "dir",
		Content: []string{"images"},
		Path:    &path,
		Nodes:   []string{cfg.Node},
	})
	e2e.RequireNoError(t, "create storage", err)

	return name
}

// TestQemuLifecycleCreateResizeDelete runs a full, real QEMU guest through
// create (in the off state) -> resize its disk -> delete:
//
//	create -> status(off) -> content(1G) -> resize(+1G) -> content(2G) -> delete -> status(absent)
//
// Unlike the rest of this package's suite (see qemu_test.go's package doc),
// this test creates and destroys a real guest: Create/Delete now exist, so
// this no longer has to run against a guaranteed-nonexistent VMID, and
// cleanup (guest, then its scratch storage) is fully self-contained.
// The guest is deliberately never started (CreateOptions.Start defaults to
// false) — the resize is exercised against an offline disk, which needs no
// running QEMU process to cooperate.
func TestQemuLifecycleCreateResizeDelete(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu lifecycle test")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	storageName := newImagesStorage(t, cfg, client)
	qc := client.Nodes(cfg.Node).Qemu()
	tc := client.Nodes(cfg.Node).Tasks()

	vmid, err := client.Cluster().NextID(ctx, 0)
	e2e.RequireNoError(t, "allocate next vmid", err)

	name := e2e.UniqueName(cfg.Prefix)

	// 1. create — a minimal guest with a single 1G disk on the scratch
	// storage, left in the off state.
	createUPID, err := qc.Create(ctx, &qemu.CreateOptions{
		VMID: vmid,
		Config: qemu.Config{
			Name:   name,
			SCSIHW: "virtio-scsi-single",
			SCSI: map[int]qemu.Drive{
				0: {File: storageName + ":1"},
			},
		},
	})
	e2e.RequireNoError(t, "create vm", err)

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete vm", func() error {
				deleteUPID, delErr := qc.Delete(cleanupCtx, vmid, nil)
				if delErr != nil {
					return delErr
				}

				return tc.Wait(cleanupCtx, deleteUPID, &tasks.WaitOptions{Timeout: taskTimeout})
			})
		})
	}

	err = tc.Wait(ctx, createUPID, &tasks.WaitOptions{Timeout: taskTimeout})
	e2e.RequireNoError(t, "wait for vm creation", err)

	// 2. status — verify the guest exists and is off.
	status, err := qc.Status(ctx, vmid)
	e2e.RequireNoError(t, "status after create", err)
	if status.Status != qemu.VMStatusStopped {
		t.Errorf("status after create: Status = %q, want %q", status.Status, qemu.VMStatusStopped)
	}

	// 3. content — verify the disk was allocated at the requested 1G.
	cc := client.Nodes(cfg.Node).Storage().Content(storageName)

	volumes, err := cc.List(ctx, &storage.ContentListOptions{VMID: vmid})
	e2e.RequireNoError(t, "list disk content after create", err)
	if len(volumes) != 1 {
		t.Fatalf("list disk content after create: got %d volumes, want 1: %+v", len(volumes), volumes)
	}
	if volumes[0].Size != giB {
		t.Errorf("disk size after create: got %d, want %d (1G)", volumes[0].Size, giB)
	}

	// 4. resize — grow the disk by 1G, to 2G.
	resizeUPID, err := qc.Resize(ctx, vmid, &qemu.ResizeOptions{
		Disk: "scsi0",
		Size: "+1G",
	})
	e2e.RequireNoError(t, "resize disk", err)

	if resizeUPID != "" {
		err = tc.Wait(ctx, resizeUPID, &tasks.WaitOptions{Timeout: taskTimeout})
		e2e.RequireNoError(t, "wait for resize", err)
	}

	// 5. content — verify the resize took effect.
	volumes, err = cc.List(ctx, &storage.ContentListOptions{VMID: vmid})
	e2e.RequireNoError(t, "list disk content after resize", err)
	if len(volumes) != 1 {
		t.Fatalf("list disk content after resize: got %d volumes, want 1: %+v", len(volumes), volumes)
	}
	if volumes[0].Size != 2*giB {
		t.Errorf("disk size after resize: got %d, want %d (2G)", volumes[0].Size, 2*giB)
	}

	// 6. delete.
	deleteUPID, err := qc.Delete(ctx, vmid, nil)
	e2e.RequireNoError(t, "delete vm", err)

	err = tc.Wait(ctx, deleteUPID, &tasks.WaitOptions{Timeout: taskTimeout})
	e2e.RequireNoError(t, "wait for vm deletion", err)

	// 7. status — verify the delete.
	_, err = qc.Status(ctx, vmid)
	e2e.RequireError(t, "status of deleted guest", err)
}
