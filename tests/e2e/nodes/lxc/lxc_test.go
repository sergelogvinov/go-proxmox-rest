//go:build e2e

// Package lxc_e2e exercises the per-node LXC container module (status,
// config, clone, template) against a live Proxmox VE cluster.
//
// This suite has no CT creation or destroy method yet, so every write
// path here (the power actions in this file, UpdateConfig in
// config_test.go, Clone in clone_test.go, Template in template_test.go)
// runs against a syntactically valid but guaranteed-nonexistent VMID
// rather than a real container: starting/stopping/reconfiguring a real
// container is too disruptive to run unattended, and Clone/Template
// would leave state (a new container, or an irreversible template
// conversion) this suite has no way to clean up. Config's read path
// (config_test.go) is the exception — being read-only, it's also
// exercised opportunistically against any real container found via
// cluster.Resources, to verify decoding against production data.
//
// Proxmox's own actions split on this, matching PVE::LXC::check_running:
// Stop/Shutdown/Suspend/Reboot all die synchronously with "CT $vmid not
// running" before ever forking a task, so these return a client error.
// Start/Resume do not synchronously require the container to already
// exist (Start's own check is "already running", which is trivially
// false for a nonexistent CT; Resume has no existence check at all) —
// the HTTP call itself succeeds with a UPID, and only the async task
// (which this suite never polls) discovers there is no such container.
// That worker task is harmless: it fails almost immediately and touches
// nothing, since the container never existed.
package lxc_e2e

import (
	"testing"

	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// nonexistentVMID is used across every test in this package: it is a
// valid VMID per Proxmox's own numeric range but is never allocated by
// this suite.
const nonexistentVMID = 999999999

// TestLXCStatusCurrentAbsent verifies that GET
// /nodes/{node}/lxc/{vmid}/status/current on a nonexistent container
// returns an error.
func TestLXCStatusCurrentAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).LXC().Status(ctx, nonexistentVMID)
	e2e.RequireError(t, "current status of absent container", err)
}

// TestLXCStatusActionsAbsentSynchronousCheck verifies that the actions
// which check container existence before forking a task (Stop,
// Shutdown, Suspend, Reboot) return an error against a nonexistent
// container.
func TestLXCStatusActionsAbsentSynchronousCheck(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	lc := client.Nodes(cfg.Node).LXC()
	ctx := t.Context()

	_, err := lc.Stop(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "stop absent container", err)

	_, err = lc.Shutdown(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "shutdown absent container", err)

	_, err = lc.Suspend(ctx, nonexistentVMID)
	e2e.RequireError(t, "suspend absent container", err)

	_, err = lc.Reboot(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "reboot absent container", err)
}

// TestLXCStatusActionsAbsentAsyncTask verifies that Start/Resume against
// a nonexistent container still succeed at the HTTP level (neither has a
// synchronous existence check) and return a non-empty UPID.
func TestLXCStatusActionsAbsentAsyncTask(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	lc := client.Nodes(cfg.Node).LXC()
	ctx := t.Context()

	upid, err := lc.Start(ctx, nonexistentVMID, nil)
	e2e.RequireNoError(t, "start absent container", err)
	if upid == "" {
		t.Errorf("start absent container: got empty UPID")
	}

	upid, err = lc.Resume(ctx, nonexistentVMID)
	e2e.RequireNoError(t, "resume absent container", err)
	if upid == "" {
		t.Errorf("resume absent container: got empty UPID")
	}
}
