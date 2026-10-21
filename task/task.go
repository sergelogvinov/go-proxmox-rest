/*
Copyright 2026 Proxmox Community.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package task is the handle for a Proxmox VE background task (a
// worker identified by a UPID) and the client-side logic for following
// one: its status, log, stop, and Wait, a blocking helper that polls
// until the task finishes (Proxmox has no server-side "block until
// done" endpoint of its own).
//
// Every endpoint that starts a background task (a guest start, a clone,
// a migration, a storage upload, ...) returns a *Task bound to the
// client that started it:
//
//	t, err := client.Nodes("pve").Qemu().Start(ctx, 100, nil)
//	if err != nil { ... }
//	err = t.Wait(ctx, &task.WaitOptions{Timeout: 5 * time.Minute})
//
// Like the types package, task is a leaf: it depends on no other package
// in this module except internal ones, so any API section can return a
// *Task without importing a sibling (see docs/design.md §2).
// nodes/tasks exposes the same operations for a UPID string, plus the
// node's task history.
package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Getter is the subset of the root client a Task uses. It is satisfied
// by *proxmox.Client, and by the Getter interface of every package that
// returns a *Task.
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Task is a handle for one background task. It is cheap to create and
// holds no state beyond its UPID: every method queries Proxmox afresh.
type Task struct {
	client Getter
	node   string
	upid   string
}

// New returns a handle for the task upid, queried through c. node is the
// node the triggering request was addressed to; it is used only when
// upid doesn't name a node itself (see Node). An empty upid yields a
// handle for a request that completed synchronously: Wait returns nil
// at once, and Status, Log and Stop return an error.
func New(c Getter, node, upid string) *Task {
	return &Task{client: c, node: node, upid: upid}
}

// UPID returns the task's unique process ID, or "" if the request that
// returned this handle completed synchronously.
func (t *Task) UPID() string {
	return t.upid
}

// Node returns the node the task runs on: the node embedded in its UPID
// — a UPID's format is "UPID:{node}:{pid}:{pstart}:{starttime}:{type}:
// {id}:{user}:" — falling back to the node given to New if the UPID
// isn't in the expected format.
//
// Per-task endpoints (status/log/stop) must address this node, which is
// not necessarily the node the triggering request was addressed to:
// Proxmox spawns a task's worker process on whichever node actually
// performs the work (e.g. a storage upload's imgcopy task can run on
// whichever node hosts that storage), and that's the only node that can
// answer for it. Addressing the wrong node fails with "Parameter
// verification failed" (upid doesn't match {node}) or "no such task"
// (right node, but it never ran this upid).
func (t *Task) Node() string {
	return taskNode(t.node, t.upid)
}

// Status retrieves the task's current status via
// GET /nodes/{node}/tasks/{upid}/status, {node} being Node().
// This user=>all endpoint has no fixed privilege for the UPID owner; a non-owner
// conditionally requires Sys.Audit on /nodes/{node}.
//
// +proxmox:rbac:path=/nodes/{node},method=GET,privs=Sys.Audit,match=any
func (t *Task) Status(ctx context.Context) (*Status, error) {
	if err := t.check(); err != nil {
		return nil, err
	}

	status := &Status{}
	if err := t.client.Get(ctx, t.path("/status"), status, nil); err != nil {
		return nil, err
	}

	return status, nil
}

// Log retrieves the task's log via GET /nodes/{node}/tasks/{upid}/log,
// {node} being Node(). opts may be nil to request Proxmox's default
// window (the first 50 lines).
// This user=>all endpoint has no fixed privilege for the UPID owner; a non-owner
// conditionally requires Sys.Audit on /nodes/{node}.
//
// +proxmox:rbac:path=/nodes/{node},method=GET,privs=Sys.Audit,match=any
func (t *Task) Log(ctx context.Context, opts *LogOptions) ([]LogEntry, error) {
	if err := t.check(); err != nil {
		return nil, err
	}

	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var entries []LogEntry
	if err := t.client.Get(ctx, t.path("/log"), &entries, p); err != nil {
		return nil, err
	}

	return entries, nil
}

// Stop terminates the running task via DELETE /nodes/{node}/tasks/{upid},
// {node} being Node().
// This user=>all endpoint has no fixed privilege for the UPID owner; a non-owner
// conditionally requires Sys.Modify on /nodes/{node}.
//
// +proxmox:rbac:path=/nodes/{node},method=DELETE,privs=Sys.Modify,match=any
func (t *Task) Stop(ctx context.Context) error {
	if err := t.check(); err != nil {
		return err
	}

	return t.client.Delete(ctx, t.path(""), nil, nil)
}

// List retrieves node's task history via GET /nodes/{node}/tasks. opts
// may be nil to request Proxmox's defaults (the most recent 50 archived
// tasks, filtered to those the caller may see).
func List(ctx context.Context, c Getter, node string, opts *ListOptions) ([]Entry, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var list []Entry
	if err := c.Get(ctx, "/nodes/"+node+"/tasks", &list, p); err != nil {
		return nil, err
	}

	return list, nil
}

// check rejects per-task requests on a handle with no UPID.
func (t *Task) check() error {
	if t.upid == "" {
		return fmt.Errorf("task: no task to query: the request completed synchronously")
	}

	return nil
}

// path builds the /nodes/{node}/tasks/{upid}{sub} URL.
func (t *Task) path(sub string) string {
	return "/nodes/" + t.Node() + "/tasks/" + t.upid + sub
}

// taskNode returns the node embedded in upid, falling back to
// defaultNode if upid isn't in the expected format. See Task.Node.
func taskNode(defaultNode, upid string) string {
	parts := strings.SplitN(upid, ":", 3)
	if len(parts) < 2 || parts[0] != "UPID" || parts[1] == "" {
		return defaultNode
	}

	return parts[1]
}
