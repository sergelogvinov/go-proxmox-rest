// Package replication provides access to the Proxmox VE per-node
// replication status API (endpoints under /nodes/{node}/replication): the
// runtime status and logs of storage replication jobs whose guest runs on
// that node.
//
// Job configuration (create/update/delete) lives in cluster/replication;
// this package only reports status and lets a job be run early.
package replication

import (
	"context"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the replication package
// decoupled from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /nodes/{node}/replication resource tree.
// Every method takes the target node's name as a call argument, since this
// package has no persistent per-node scope of its own.
type Client struct {
	client Getter
}

// New returns a new replication client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// List retrieves the status of every replication job whose guest runs on
// the given node via GET /nodes/{node}/replication. guest, when non-zero,
// narrows the result to that single guest's jobs.
func (c *Client) List(ctx context.Context, node string, guest int) ([]JobStatus, error) {
	var p map[string]string
	if guest != 0 {
		p = map[string]string{"guest": strconv.Itoa(guest)}
	}

	var jobs []JobStatus
	if err := c.client.Get(ctx, "/nodes/"+node+"/replication", &jobs, p); err != nil {
		return nil, err
	}

	return jobs, nil
}

// Get retrieves a single replication job's status via
// GET /nodes/{node}/replication/{id}/status.
func (c *Client) Get(ctx context.Context, node, id string) (*JobStatus, error) {
	status := &JobStatus{}
	if err := c.client.Get(ctx, "/nodes/"+node+"/replication/"+id+"/status", status, nil); err != nil {
		return nil, err
	}

	return status, nil
}

// Log retrieves a replication job's log via
// GET /nodes/{node}/replication/{id}/log. opts may be nil to request
// every line Proxmox has.
func (c *Client) Log(ctx context.Context, node, id string, opts *LogOptions) ([]LogEntry, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var entries []LogEntry
	if err := c.client.Get(ctx, "/nodes/"+node+"/replication/"+id+"/log", &entries, p); err != nil {
		return nil, err
	}

	return entries, nil
}

// ScheduleNow requests that a replication job run as soon as possible via
// POST /nodes/{node}/replication/{id}/schedule_now, bypassing its normal
// schedule for one run.
func (c *Client) ScheduleNow(ctx context.Context, node, id string) error {
	return c.client.Create(ctx, "/nodes/"+node+"/replication/"+id+"/schedule_now", nil, nil)
}
