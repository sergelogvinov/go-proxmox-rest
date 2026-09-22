//go:build e2e

package lxc_e2e

import (
	"testing"
	"time"

	proxmox "github.com/sergelogvinov/go-proxmox-rest"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/storage"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/tasks"
	rootstorage "github.com/sergelogvinov/go-proxmox-rest/storage"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// giB is one gibibyte, in bytes, matching how Status reports MaxDisk.
const giB = 1024 * 1024 * 1024

// taskTimeout bounds how long this suite waits for the create/resize/
// delete tasks below. Container creation extracts a real OS template
// archive onto a freshly formatted loopback image, so it runs well past
// tasks.WaitOptions' short default.
const taskTimeout = 5 * time.Minute

// newRootDirStorage creates a uniquely-named "dir" storage, backed by a
// temporary path and scoped to cfg.Node, with "rootdir" content enabled
// so a container's root filesystem can be allocated on it. It registers
// its own cleanup.
func newRootDirStorage(t *testing.T, cfg *e2e.E2EConfig, client *proxmox.Client) string {
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
		Content: []string{"rootdir"},
		Path:    &path,
		Nodes:   []string{cfg.Node},
	})
	e2e.RequireNoError(t, "create storage", err)

	return name
}

// findOSTemplate looks across every storage on cfg.Node that advertises
// "vztmpl" content for an already-downloaded container template,
// returning its volume id, or "" if none is found. This suite has no
// template upload/download method of its own, so container creation can
// only be exercised opportunistically against whatever a real cluster
// already has cached — same spirit as TestQemuConfigOpportunistic in the
// qemu suite.
func findOSTemplate(t *testing.T, cfg *e2e.E2EConfig, client *proxmox.Client) string {
	t.Helper()

	ctx := t.Context()
	nc := client.Nodes(cfg.Node)

	storages, err := nc.Storage().List(ctx, &storage.ListOptions{
		Content: []string{"vztmpl"},
		Enabled: true,
	})
	e2e.RequireNoError(t, "list storages with vztmpl content", err)

	for _, st := range storages {
		volumes, err := nc.Storage().Content(st.Storage).List(ctx, &storage.ContentListOptions{Content: "vztmpl"})
		e2e.RequireNoError(t, "list vztmpl content on "+st.Storage, err)

		if len(volumes) > 0 {
			return volumes[0].VolID
		}
	}

	return ""
}

// TestLXCLifecycleCreateResizeDelete runs a full, real LXC container
// through create (in the off state) -> resize its rootfs -> delete:
//
//	create -> status(off, 1G) -> resize(+1G) -> status(2G) -> delete -> status(absent)
//
// Unlike the rest of this package's suite (see lxc_test.go's package
// doc), this test creates and destroys a real container: Create/Delete
// now exist, so this no longer has to run against a guaranteed-
// nonexistent VMID. It still needs a real OS template already present on
// the cluster (see findOSTemplate) and skips if none is found.
//
// The container is deliberately never started (CreateOptions.Start
// defaults to false) — the resize is exercised against an offline
// rootfs, which needs no running container to cooperate.
func TestLXCLifecycleCreateResizeDelete(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc lifecycle test")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	templateVolID := findOSTemplate(t, cfg, client)
	if templateVolID == "" {
		t.Skip("no OS template found on PVE_E2E_NODE; skipping lxc lifecycle test")
	}

	storageName := newRootDirStorage(t, cfg, client)
	lc := client.Nodes(cfg.Node).LXC()
	tc := client.Nodes(cfg.Node).Tasks()

	vmid, err := client.Cluster().NextID(ctx, 0)
	e2e.RequireNoError(t, "allocate next vmid", err)

	name := e2e.UniqueName(cfg.Prefix)

	// 1. create — a minimal container with a 1G rootfs on the scratch
	// storage, left in the off state.
	createUPID, err := lc.Create(ctx, &lxc.CreateOptions{
		VMID:       vmid,
		OSTemplate: templateVolID,
		Config: lxc.Config{
			Hostname: name,
			RootFS: &lxc.RootFS{
				Volume: storageName + ":1",
			},
		},
	})
	e2e.RequireNoError(t, "create container", err)

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete container", func() error {
				deleteUPID, delErr := lc.Delete(cleanupCtx, vmid, nil)
				if delErr != nil {
					return delErr
				}

				return tc.Wait(cleanupCtx, deleteUPID, &tasks.WaitOptions{Timeout: taskTimeout})
			})
		})
	}

	err = tc.Wait(ctx, createUPID, &tasks.WaitOptions{Timeout: taskTimeout})
	e2e.RequireNoError(t, "wait for container creation", err)

	// 2. status — verify the container exists, is off, and its rootfs
	// was allocated at the requested 1G.
	status, err := lc.Status(ctx, vmid)
	e2e.RequireNoError(t, "status after create", err)
	if status.Status != lxc.StateStopped {
		t.Errorf("status after create: Status = %q, want %q", status.Status, lxc.StateStopped)
	}
	if status.MaxDisk != giB {
		t.Errorf("rootfs size after create: got %d, want %d (1G)", status.MaxDisk, giB)
	}

	// 3. resize — grow the rootfs by 1G, to 2G.
	resizeUPID, err := lc.Resize(ctx, vmid, &lxc.ResizeOptions{
		Disk: "rootfs",
		Size: "+1G",
	})
	e2e.RequireNoError(t, "resize rootfs", err)

	if resizeUPID != "" {
		err = tc.Wait(ctx, resizeUPID, &tasks.WaitOptions{Timeout: taskTimeout})
		e2e.RequireNoError(t, "wait for resize", err)
	}

	// 4. status — verify the resize took effect.
	status, err = lc.Status(ctx, vmid)
	e2e.RequireNoError(t, "status after resize", err)
	if status.MaxDisk != 2*giB {
		t.Errorf("rootfs size after resize: got %d, want %d (2G)", status.MaxDisk, 2*giB)
	}

	// 5. delete.
	deleteUPID, err := lc.Delete(ctx, vmid, nil)
	e2e.RequireNoError(t, "delete container", err)

	err = tc.Wait(ctx, deleteUPID, &tasks.WaitOptions{Timeout: taskTimeout})
	e2e.RequireNoError(t, "wait for container deletion", err)

	// 6. status — verify the delete.
	_, err = lc.Status(ctx, vmid)
	e2e.RequireError(t, "status of deleted container", err)
}
