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

// Package tasks provides access to the Proxmox VE per-node task API
// (endpoints under /nodes/{node}/tasks): the node's task history, and a
// single task's log/status/stop/wait addressed by its UPID string.
//
// The per-task operations are implemented by the task package: every
// endpoint that starts a background task already returns a *task.Task
// handle with the same methods. Use this package when all you hold is
// a UPID (e.g. one read from List).
package tasks

import (
	"context"

	"github.com/sergelogvinov/go-proxmox-rest/task"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the tasks package decoupled
// from the root package (no import cycle).
type Getter = task.Getter

// State, Source, Task (a history entry), ListOptions, LogEntry,
// LogOptions, Status, WaitOptions and FailedError are defined in the
// task package and re-exported here so existing call sites keep
// working.
type (
	State       = task.State
	Source      = task.Source
	Task        = task.Entry
	ListOptions = task.ListOptions
	LogEntry    = task.LogEntry
	LogOptions  = task.LogOptions
	Status      = task.Status
	WaitOptions = task.WaitOptions
	FailedError = task.FailedError
)

const (
	StateRunning = task.StateRunning
	StateStopped = task.StateStopped

	SourceArchive = task.SourceArchive
	SourceActive  = task.SourceActive
	SourceAll     = task.SourceAll
)

// Client provides access to the /nodes/{node}/tasks resource tree, scoped
// to the node given to New.
type Client struct {
	client Getter
	node   string
}

// New returns a new tasks client backed by the given root client, scoped
// to node.
func New(c Getter, node string) *Client {
	return &Client{client: c, node: node}
}

// Task returns the handle for the task upid. Its per-task requests are
// addressed to the node embedded in upid, not necessarily this Client's
// node — see task.Task.Node.
func (c *Client) Task(upid string) *task.Task {
	return task.New(c.client, c.node, upid)
}

// List retrieves the node's task history via GET /nodes/{node}/tasks.
// opts may be nil to request Proxmox's defaults (the most recent 50
// archived tasks, filtered to those the caller may see).
// This user=>all endpoint has no fixed privilege: own tasks are visible, while
// Sys.Audit on /nodes/{node} conditionally permits all tasks; results are filtered.
//
// +proxmox:rbac:path=/nodes/{node},method=GET,privs=Sys.Audit,match=any
func (c *Client) List(ctx context.Context, opts *ListOptions) ([]Task, error) {
	return task.List(ctx, c.client, c.node, opts)
}

// Status retrieves a single task's current status; see task.Task.Status.
//
// +proxmox:rbac:path=/nodes/{node},method=GET,privs=Sys.Audit,match=any
func (c *Client) Status(ctx context.Context, upid string) (*Status, error) {
	return c.Task(upid).Status(ctx)
}

// Log retrieves a single task's log; see task.Task.Log.
//
// +proxmox:rbac:path=/nodes/{node},method=GET,privs=Sys.Audit,match=any
func (c *Client) Log(ctx context.Context, upid string, opts *LogOptions) ([]LogEntry, error) {
	return c.Task(upid).Log(ctx, opts)
}

// Stop terminates a running task; see task.Task.Stop.
//
// +proxmox:rbac:path=/nodes/{node},method=DELETE,privs=Sys.Modify,match=any
func (c *Client) Stop(ctx context.Context, upid string) error {
	return c.Task(upid).Stop(ctx)
}

// Wait blocks until the task upid stops running; see task.Task.Wait.
func (c *Client) Wait(ctx context.Context, upid string, opts *WaitOptions) error {
	return c.Task(upid).Wait(ctx, opts)
}
