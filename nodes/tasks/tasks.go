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

// Client provides access to the /nodes/{node}/tasks resource tree. Every
// method takes the target node's name as a call argument, since this
// package has no persistent per-node scope of its own.
type Client struct {
	client Getter
}

// New returns a new tasks client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// List retrieves the given node's task history via GET /nodes/{node}/tasks.
// opts may be nil to request Proxmox's defaults (the most recent 50
// archived tasks, filtered to those the caller may see).
func (c *Client) List(ctx context.Context, node string, opts *ListOptions) ([]Task, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var list []Task
	if err := c.client.Get(ctx, "/nodes/"+node+"/tasks", &list, p); err != nil {
		return nil, err
	}

	return list, nil
}

// Status retrieves a single task's current status via
// GET /nodes/{node}/tasks/{upid}/status.
func (c *Client) Status(ctx context.Context, node, upid string) (*Status, error) {
	status := &Status{}
	if err := c.client.Get(ctx, "/nodes/"+node+"/tasks/"+upid+"/status", status, nil); err != nil {
		return nil, err
	}

	return status, nil
}

// Log retrieves a single task's log via GET /nodes/{node}/tasks/{upid}/log.
// opts may be nil to request Proxmox's default window (the first 50
// lines).
func (c *Client) Log(ctx context.Context, node, upid string, opts *LogOptions) ([]LogEntry, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var entries []LogEntry
	if err := c.client.Get(ctx, "/nodes/"+node+"/tasks/"+upid+"/log", &entries, p); err != nil {
		return nil, err
	}

	return entries, nil
}

// Stop terminates a running task via DELETE /nodes/{node}/tasks/{upid}.
func (c *Client) Stop(ctx context.Context, node, upid string) error {
	return c.client.Delete(ctx, "/nodes/"+node+"/tasks/"+upid, nil, nil)
}
