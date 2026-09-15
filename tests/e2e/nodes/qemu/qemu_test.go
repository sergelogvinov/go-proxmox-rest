//go:build e2e

// Package qemu_e2e exercises the per-node QEMU guest status module
// against a live Proxmox VE cluster.
//
// This suite has no VM creation module yet (nodes/qemu only covers the
// status resource tree so far) and, even if it did, starting/stopping a
// real guest is far too disruptive to run unattended as part of a
// general e2e suite. So every check here runs against a syntactically
// valid but guaranteed-nonexistent VMID.
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

	_, err := client.Nodes().Qemu().Status(ctx, cfg.Node, nonexistentVMID)
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
	sc := client.Nodes().Qemu()
	ctx := t.Context()

	_, err := sc.Reset(ctx, cfg.Node, nonexistentVMID, nil)
	e2e.RequireError(t, "reset absent guest", err)

	_, err = sc.Reboot(ctx, cfg.Node, nonexistentVMID, nil)
	e2e.RequireError(t, "reboot absent guest", err)

	_, err = sc.Suspend(ctx, cfg.Node, nonexistentVMID, nil)
	e2e.RequireError(t, "suspend absent guest", err)

	_, err = sc.Resume(ctx, cfg.Node, nonexistentVMID, nil)
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
	sc := client.Nodes().Qemu()
	ctx := t.Context()

	upid, err := sc.Start(ctx, cfg.Node, nonexistentVMID, nil)
	e2e.RequireNoError(t, "start absent guest", err)
	if upid == "" {
		t.Errorf("start absent guest: got empty UPID")
	}

	upid, err = sc.Stop(ctx, cfg.Node, nonexistentVMID, nil)
	e2e.RequireNoError(t, "stop absent guest", err)
	if upid == "" {
		t.Errorf("stop absent guest: got empty UPID")
	}

	upid, err = sc.Shutdown(ctx, cfg.Node, nonexistentVMID, nil)
	e2e.RequireNoError(t, "shutdown absent guest", err)
	if upid == "" {
		t.Errorf("shutdown absent guest: got empty UPID")
	}
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

	upid, err := client.Nodes().Qemu().Start(ctx, cfg.Node, nonexistentVMID, &qemu.StartOptions{
		Timeout: 5,
	})
	e2e.RequireNoError(t, "start absent guest with options", err)
	if upid == "" {
		t.Errorf("start absent guest with options: got empty UPID")
	}
}
