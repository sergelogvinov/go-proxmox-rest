//go:build e2e

package cluster_e2e

import (
	"slices"
	"strconv"
	"testing"
	"time"

	proxmox "github.com/sergelogvinov/go-proxmox-rest"
	"github.com/sergelogvinov/go-proxmox-rest/cluster/ha"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	"github.com/sergelogvinov/go-proxmox-rest/task"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// taskTimeout bounds how long the fixtures below wait for a guest's
// create/delete task; the guest has no disks, so these finish quickly.
const taskTimeout = 2 * time.Minute

// newStoppedVM creates a minimal, diskless QEMU guest on cfg.Node, left in
// the off state, and returns its HA resource ID ("vm:<vmid>"). It
// registers its own cleanup.
func newStoppedVM(t *testing.T, cfg *e2e.E2EConfig, client *proxmox.Client) string {
	t.Helper()

	ctx := t.Context()
	qc := client.Nodes(cfg.Node).Qemu()

	vmid, err := client.Cluster().NextID(ctx, 0)
	e2e.RequireNoError(t, "allocate next vmid", err)

	createTask, err := qc.Create(ctx, &qemu.CreateOptions{
		VMID:   vmid,
		Config: qemu.Config{Name: e2e.UniqueName(cfg.Prefix)},
	})
	e2e.RequireNoError(t, "create vm", err)

	t.Cleanup(func() {
		if t.Failed() && !cfg.CleanupOnFailure {
			return
		}

		cleanupCtx, cancel := e2e.CleanupContext()
		defer cancel()

		e2e.RetryCleanup(t, "delete vm", func() error {
			deleteTask, delErr := qc.Delete(cleanupCtx, vmid, nil)
			if delErr != nil {
				return delErr
			}

			return deleteTask.Wait(cleanupCtx, &task.WaitOptions{Timeout: taskTimeout})
		})
	})

	err = createTask.Wait(ctx, &task.WaitOptions{Timeout: taskTimeout})
	e2e.RequireNoError(t, "wait for vm creation", err)

	return "vm:" + strconv.Itoa(vmid)
}

// newHAResource places a fresh stopped guest under HA management, so HA
// rules can reference it — Proxmox rejects rules naming unmanaged
// resources. It returns the resource's ID and registers cleanup that
// removes the HA resource (purging it from any rule) before the guest.
func newHAResource(t *testing.T, cfg *e2e.E2EConfig, client *proxmox.Client) string {
	t.Helper()

	sid := newStoppedVM(t, cfg, client)
	hr := client.Cluster().HA().Resources()

	err := hr.Create(t.Context(), &ha.ResourceOptions{
		SID:   sid,
		State: ha.ResourceStateStopped,
	})
	e2e.RequireNoError(t, "create ha resource", err)

	// Registered after the guest's cleanup, so it runs first.
	t.Cleanup(func() {
		if t.Failed() && !cfg.CleanupOnFailure {
			return
		}

		cleanupCtx, cancel := e2e.CleanupContext()
		defer cancel()

		e2e.RetryCleanup(t, "delete ha resource", func() error {
			return hr.Delete(cleanupCtx, sid, true)
		})
	})

	return sid
}

// TestHAResourcesLifecycle runs the full CRUD lifecycle for an HA resource
// backed by a real, stopped guest:
//
//	list → get(absent) → create → get → update → get → delete → list
func TestHAResourcesLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping HA resource tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	hr := client.Cluster().HA().Resources()
	ctx := t.Context()

	sid := newStoppedVM(t, cfg, client)

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete ha resource", func() error {
				return hr.Delete(cleanupCtx, sid, true)
			})
		})
	}

	// 1. list — baseline: the guest must not be HA-managed yet.
	listed, err := hr.List(ctx, "")
	e2e.RequireNoError(t, "list ha resources", err)
	if containsHAResource(listed, sid) {
		t.Fatalf("list: ha resource %q already exists before create", sid)
	}

	// 2. get (absent) — an unmanaged guest must return an error.
	_, err = hr.Get(ctx, sid)
	e2e.RequireError(t, "get absent ha resource", err)

	// 3. create — keep the guest stopped so the CRM never starts it.
	comment := "e2e lifecycle"
	err = hr.Create(ctx, &ha.ResourceOptions{
		SID:     sid,
		State:   ha.ResourceStateStopped,
		Comment: &comment,
	})
	e2e.RequireNoError(t, "create ha resource", err)

	// 4. get — verify the create.
	res, err := hr.Get(ctx, sid)
	e2e.RequireNoError(t, "get ha resource after create", err)
	if res.SID != sid {
		t.Errorf("get: SID = %q, want %q", res.SID, sid)
	}
	if res.State != string(ha.ResourceStateStopped) {
		t.Errorf("get: State = %q, want %q", res.State, ha.ResourceStateStopped)
	}
	if res.Comment != comment {
		t.Errorf("get: Comment = %q, want %q", res.Comment, comment)
	}

	// 5. update — mutate the comment and the restart limit.
	newComment := "e2e updated"
	maxRestart := 3
	err = hr.Update(ctx, sid, &ha.ResourceOptions{
		Comment:    &newComment,
		MaxRestart: &maxRestart,
	})
	e2e.RequireNoError(t, "update ha resource", err)

	// 6. get — verify the update; State must be unchanged.
	res, err = hr.Get(ctx, sid)
	e2e.RequireNoError(t, "get ha resource after update", err)
	if res.Comment != newComment {
		t.Errorf("get after update: Comment = %q, want %q", res.Comment, newComment)
	}
	if res.MaxRestart != maxRestart {
		t.Errorf("get after update: MaxRestart = %d, want %d", res.MaxRestart, maxRestart)
	}
	if res.State != string(ha.ResourceStateStopped) {
		t.Errorf("get after update: State = %q, want unchanged %q", res.State, ha.ResourceStateStopped)
	}

	// The type filter must include the VM and exclude it under "ct".
	vms, err := hr.List(ctx, ha.ResourceTypeVM)
	e2e.RequireNoError(t, "list ha resources filtered by type=vm", err)
	if !containsHAResource(vms, sid) {
		t.Errorf("list type=vm: resource %q missing from result", sid)
	}
	cts, err := hr.List(ctx, ha.ResourceTypeCT)
	e2e.RequireNoError(t, "list ha resources filtered by type=ct", err)
	if containsHAResource(cts, sid) {
		t.Errorf("list type=ct: vm resource %q unexpectedly present", sid)
	}

	// 7. delete — remove the guest from HA management.
	err = hr.Delete(ctx, sid, true)
	e2e.RequireNoError(t, "delete ha resource", err)

	// 8. list — verify the delete.
	listed, err = hr.List(ctx, "")
	e2e.RequireNoError(t, "list ha resources after delete", err)
	if containsHAResource(listed, sid) {
		t.Errorf("list after delete: ha resource %q still present", sid)
	}
}

// TestHAResourcesValidation verifies client-side validation of
// Create/Update options, none of which should reach the API.
func TestHAResourcesValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	hr := client.Cluster().HA().Resources()
	ctx := t.Context()

	err := hr.Create(ctx, nil)
	e2e.RequireError(t, "create with nil options", err)

	err = hr.Create(ctx, &ha.ResourceOptions{State: ha.ResourceStateStopped})
	e2e.RequireError(t, "create with missing sid", err)

	err = hr.Update(ctx, "vm:100", nil)
	e2e.RequireError(t, "update with nil options", err)
}

// containsHAResource reports whether the slice contains an HA resource with
// the given ID.
func containsHAResource(list []ha.Resource, sid string) bool {
	return slices.ContainsFunc(list, func(r ha.Resource) bool { return r.SID == sid })
}
