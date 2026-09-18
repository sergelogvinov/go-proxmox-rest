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

// Client provides access to the /nodes/{node}/replication resource tree,
// scoped to the node given to New.
type Client struct {
	client Getter
	node   string
}

// New returns a new replication client backed by the given root client,
// scoped to node.
func New(c Getter, node string) *Client {
	return &Client{client: c, node: node}
}

// List retrieves the status of every replication job whose guest runs on
// the node via GET /nodes/{node}/replication. guest, when non-zero,
// narrows the result to that single guest's jobs.
// This user=>all endpoint has no fixed privilege; each result conditionally
// requires VM.Audit on the job-derived /vms/{vmid} path.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (c *Client) List(ctx context.Context, guest int) ([]JobStatus, error) {
	var p map[string]string
	if guest != 0 {
		p = map[string]string{"guest": strconv.Itoa(guest)}
	}

	var jobs []JobStatus
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/replication", &jobs, p); err != nil {
		return nil, err
	}

	return jobs, nil
}

// Get retrieves a single replication job's status via
// GET /nodes/{node}/replication/{id}/status.
// This user=>all endpoint has no fixed privilege; the loaded job conditionally
// requires VM.Audit on its /vms/{vmid} path.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (c *Client) Get(ctx context.Context, id string) (*JobStatus, error) {
	status := &JobStatus{}
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/replication/"+id+"/status", status, nil); err != nil {
		return nil, err
	}

	return status, nil
}

// Log retrieves a replication job's log via
// GET /nodes/{node}/replication/{id}/log. opts may be nil to request
// every line Proxmox has.
// This user=>all endpoint has no fixed privilege. Access requires either VM.Audit
// on the job-derived /vms/{vmid} or Sys.Audit on /nodes/{node}.
// The marker records the guest-scoped branch; Sys.Audit is an alternative.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (c *Client) Log(ctx context.Context, id string, opts *LogOptions) ([]LogEntry, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var entries []LogEntry
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/replication/"+id+"/log", &entries, p); err != nil {
		return nil, err
	}

	return entries, nil
}

// ScheduleNow requests that a replication job run as soon as possible via
// POST /nodes/{node}/replication/{id}/schedule_now, bypassing its normal
// schedule for one run.
// This user=>all endpoint has no fixed privilege; the ID-derived VM conditionally
// requires VM.Replicate on /vms/{vmid}.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Replicate,match=all
func (c *Client) ScheduleNow(ctx context.Context, id string) error {
	return c.client.Create(ctx, "/nodes/"+c.node+"/replication/"+id+"/schedule_now", nil, nil)
}
