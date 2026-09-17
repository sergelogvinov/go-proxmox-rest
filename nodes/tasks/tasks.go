// Package tasks provides access to the Proxmox VE per-node task API
// (endpoints under /nodes/{node}/tasks): the node's task history, and a
// single task's log/status/stop.
package tasks

import (
	"context"

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
// GET /nodes/{node}/tasks/{upid}/status.
func (c *Client) Status(ctx context.Context, upid string) (*Status, error) {
	status := &Status{}
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/tasks/"+upid+"/status", status, nil); err != nil {
		return nil, err
	}

	return status, nil
}

// Log retrieves a single task's log via GET /nodes/{node}/tasks/{upid}/log.
// opts may be nil to request Proxmox's default window (the first 50
// lines).
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
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/tasks/"+upid+"/log", &entries, p); err != nil {
		return nil, err
	}

	return entries, nil
}

// Stop terminates a running task via DELETE /nodes/{node}/tasks/{upid}.
func (c *Client) Stop(ctx context.Context, upid string) error {
	return c.client.Delete(ctx, "/nodes/"+c.node+"/tasks/"+upid, nil, nil)
}
