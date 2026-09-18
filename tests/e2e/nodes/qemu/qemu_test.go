//go:build e2e

// Package qemu_e2e exercises the per-node QEMU guest module (status,
// config, clone, template, delete) against a live Proxmox VE cluster.
//
// This suite has no VM creation method yet, so every write path here
// (the power actions in this file, UpdateConfig/UpdateConfigAsync in
// config_test.go, Clone in clone_test.go, Template in template_test.go,
// Delete below) runs against a syntactically valid but
// guaranteed-nonexistent VMID rather than a real guest: starting/
// stopping/reconfiguring/destroying a real guest is too disruptive to
// run unattended, and Clone/Template would leave state (a new guest, or
// an irreversible template conversion) this suite has no way to clean
// up. Config's read path (config_test.go) is the exception — being
// read-only, it's also exercised opportunistically against any real
// guest found via cluster.Resources, to verify decoding against
// production data.
//
// Proxmox's own actions split on this: Reset/Reboot/Suspend/Resume all
// call PVE::QemuServer::check_running (or, for Resume, additionally try
// to load the guest's config) and die synchronously with "VM $vmid not
// running" before ever forking a task, so these return a client error.
// Start/Stop/Shutdown do not check the guest exists at all before
// forking their background worker task — the HTTP call itself succeeds
// with a UPID, and only the async task (which this suite never polls)
// discovers there is no such guest to start/stop/shut down. That worker
// task is harmless: it fails almost immediately and touches nothing,
// since the guest never existed.
package qemu_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// nonexistentVMID is used across every test below: it is a valid VMID
// per Proxmox's own numeric range but is never allocated by this suite.
const nonexistentVMID = 999999999

// TestQemuStatusCurrentAbsent verifies that GET
// /nodes/{node}/qemu/{vmid}/status/current on a nonexistent guest
// returns an error.
func TestQemuStatusCurrentAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).Qemu().Status(ctx, nonexistentVMID)
	e2e.RequireError(t, "current status of absent guest", err)
}

// TestQemuStatusActionsAbsentSynchronousCheck verifies that the actions
// which check guest existence before forking a task (Reset, Reboot,
// Suspend, Resume) return an error against a nonexistent guest.
func TestQemuStatusActionsAbsentSynchronousCheck(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	sc := client.Nodes(cfg.Node).Qemu()
	ctx := t.Context()

	_, err := sc.Reset(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "reset absent guest", err)

	_, err = sc.Reboot(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "reboot absent guest", err)

	_, err = sc.Suspend(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "suspend absent guest", err)

	_, err = sc.Resume(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "resume absent guest", err)
}

// TestQemuStatusActionsAbsentAsyncTask verifies that Start/Stop/Shutdown
// against a nonexistent guest still succeed at the HTTP level (Proxmox
// forks the worker task unconditionally) and return a non-empty UPID.
func TestQemuStatusActionsAbsentAsyncTask(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	sc := client.Nodes(cfg.Node).Qemu()
	ctx := t.Context()

	upid, err := sc.Start(ctx, nonexistentVMID, nil)
	e2e.RequireNoError(t, "start absent guest", err)
	if upid == "" {
		t.Errorf("start absent guest: got empty UPID")
	}

	upid, err = sc.Stop(ctx, nonexistentVMID, nil)
	e2e.RequireNoError(t, "stop absent guest", err)
	if upid == "" {
		t.Errorf("stop absent guest: got empty UPID")
	}

	upid, err = sc.Shutdown(ctx, nonexistentVMID, nil)
	e2e.RequireNoError(t, "shutdown absent guest", err)
	if upid == "" {
		t.Errorf("shutdown absent guest: got empty UPID")
	}
}

// TestQemuDeleteAbsent verifies that DELETE
// /nodes/{node}/qemu/{vmid} against a nonexistent guest returns an
// error: destroy_vm loads the guest's config synchronously and dies
// before forking its worker task if it doesn't exist.
func TestQemuDeleteAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).Qemu().Delete(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "delete absent guest", err)
}

// TestQemuStatusStartOptions verifies that Start's options are encoded
// and sent without error, returning a UPID for the (harmless, since the
// guest never existed) background task Proxmox forks unconditionally.
func TestQemuStatusStartOptions(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	upid, err := client.Nodes(cfg.Node).Qemu().Start(ctx, nonexistentVMID, &qemu.StartOptions{
		Timeout: 5,
	})
	e2e.RequireNoError(t, "start absent guest with options", err)
	if upid == "" {
		t.Errorf("start absent guest with options: got empty UPID")
	}
}
