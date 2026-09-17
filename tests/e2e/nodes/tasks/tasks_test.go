//go:build e2e

// Package tasks_e2e exercises the per-node task module against a live
// Proxmox VE cluster.
//
// List is read-only. Status/Log are opportunistically exercised against a
// real task's UPID when List(source=all) finds one (the node may have no
// task history at all, so this is skipped rather than fabricated). Stop is
// only exercised against a syntactically valid but nonexistent UPID —
// invoking it on a real task could kill genuinely important work, which
// this suite has no way to know isn't the case.
package tasks_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/tasks"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// nonexistentUPID is syntactically valid (parses via PVE::Tools::
// upid_decode) but names a task that has never run, so it's always safe to
// use for error-path checks.
const nonexistentUPID = "UPID:localhost:00000000:00000000:00000000:e2etest:e2e-999999999:root@pam:"

// TestTasksList verifies GET /nodes/{node}/tasks decodes without error and
// that every entry has a non-empty UPID/Type.
func TestTasksList(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping task tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	tc := client.Nodes(cfg.Node).Tasks()
	ctx := t.Context()

	list, err := tc.List(ctx, &tasks.ListOptions{Source: tasks.SourceAll})
	e2e.RequireNoError(t, "list tasks", err)
	for _, task := range list {
		if task.UPID == "" || task.Type == "" {
			t.Errorf("list tasks: entry with empty UPID/Type: %+v", task)
		}
	}
}

// TestTaskStatusAndLog opportunistically verifies GET
// /nodes/{node}/tasks/{upid}/status and .../log against a real task, if
// the node has any task history at all.
func TestTaskStatusAndLog(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping task tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	tc := client.Nodes(cfg.Node).Tasks()
	ctx := t.Context()

	list, err := tc.List(ctx, &tasks.ListOptions{Source: tasks.SourceAll, Limit: 1})
	e2e.RequireNoError(t, "list tasks", err)
	if len(list) == 0 {
		t.Skip("node has no task history; skipping status/log checks")
	}
	upid := list[0].UPID

	status, err := tc.Status(ctx, upid)
	e2e.RequireNoError(t, "task status", err)
	if status.UPID != upid {
		t.Errorf("task status: UPID = %q, want %q", status.UPID, upid)
	}

	_, err = tc.Log(ctx, upid, &tasks.LogOptions{Limit: 1})
	e2e.RequireNoError(t, "task log", err)
}

// TestTaskStopAbsent verifies that stopping a syntactically valid but
// nonexistent task returns an error.
func TestTaskStopAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping task tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	err := client.Nodes(cfg.Node).Tasks().Stop(ctx, nonexistentUPID)
	e2e.RequireError(t, "stop absent task", err)
}
