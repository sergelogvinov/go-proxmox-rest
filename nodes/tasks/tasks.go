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
// (endpoints under /nodes/{node}/tasks): the node's task history, a
// single task's log/status/stop, and Wait, a client-side blocking helper
// that polls Status until the task finishes (Proxmox has no
// server-side "block until done" endpoint of its own).
package tasks

import (
	"context"
	"strings"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the tasks package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

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

// List retrieves the node's task history via GET /nodes/{node}/tasks.
// opts may be nil to request Proxmox's defaults (the most recent 50
// archived tasks, filtered to those the caller may see).
// This user=>all endpoint has no fixed privilege: own tasks are visible, while
// Sys.Audit on /nodes/{node} conditionally permits all tasks; results are filtered.
//
// +proxmox:rbac:path=/nodes/{node},method=GET,privs=Sys.Audit,match=any
func (c *Client) List(ctx context.Context, opts *ListOptions) ([]Task, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var list []Task
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/tasks", &list, p); err != nil {
		return nil, err
	}

	return list, nil
}

// Status retrieves a single task's current status via
// GET /nodes/{node}/tasks/{upid}/status. {node} is the node embedded in
// upid itself (see taskNode), which does not always match the node this
// Client is scoped to — e.g. a storage upload's imgcopy task can end up
// running on whichever node actually hosts that storage, not the node
// the upload request's own URL named. Addressing the wrong node fails
// with "Parameter verification failed" (upid doesn't match {node}) or
// "no such task" (right node, but it never ran this upid).
// This user=>all endpoint has no fixed privilege for the UPID owner; a non-owner
// conditionally requires Sys.Audit on /nodes/{node}.
//
// +proxmox:rbac:path=/nodes/{node},method=GET,privs=Sys.Audit,match=any
func (c *Client) Status(ctx context.Context, upid string) (*Status, error) {
	status := &Status{}
	if err := c.client.Get(ctx, "/nodes/"+taskNode(c.node, upid)+"/tasks/"+upid+"/status", status, nil); err != nil {
		return nil, err
	}

	return status, nil
}

// Log retrieves a single task's log via GET /nodes/{node}/tasks/{upid}/log.
// {node} is upid's own node — see Status's doc comment. opts may be nil
// to request Proxmox's default window (the first 50 lines).
// This user=>all endpoint has no fixed privilege for the UPID owner; a non-owner
// conditionally requires Sys.Audit on /nodes/{node}.
//
// +proxmox:rbac:path=/nodes/{node},method=GET,privs=Sys.Audit,match=any
func (c *Client) Log(ctx context.Context, upid string, opts *LogOptions) ([]LogEntry, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var entries []LogEntry
	if err := c.client.Get(ctx, "/nodes/"+taskNode(c.node, upid)+"/tasks/"+upid+"/log", &entries, p); err != nil {
		return nil, err
	}

	return entries, nil
}

// Stop terminates a running task via DELETE /nodes/{node}/tasks/{upid}.
// {node} is upid's own node — see Status's doc comment.
// This user=>all endpoint has no fixed privilege for the UPID owner; a non-owner
// conditionally requires Sys.Modify on /nodes/{node}.
//
// +proxmox:rbac:path=/nodes/{node},method=DELETE,privs=Sys.Modify,match=any
func (c *Client) Stop(ctx context.Context, upid string) error {
	return c.client.Delete(ctx, "/nodes/"+taskNode(c.node, upid)+"/tasks/"+upid, nil, nil)
}

// taskNode returns the node embedded in upid — a UPID's format is
// "UPID:{node}:{pid}:{pstart}:{starttime}:{type}:{id}:{user}:" — falling
// back to defaultNode if upid isn't in the expected format. Per-task
// endpoints (status/log/stop) must address this node, which is not
// necessarily the node a Client happens to be scoped to (see Status's
// doc comment): Proxmox spawns a task's worker process on whichever node
// actually performs the work, and that's the only node that can answer
// for it.
func taskNode(defaultNode, upid string) string {
	parts := strings.SplitN(upid, ":", 3)
	if len(parts) < 2 || parts[0] != "UPID" || parts[1] == "" {
		return defaultNode
	}

	return parts[1]
}
