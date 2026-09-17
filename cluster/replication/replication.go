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

// Package replication provides access to the Proxmox VE cluster-wide
// storage replication job API (endpoints under /cluster/replication).
package replication

import (
	"context"
	"fmt"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the replication package
// decoupled from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /cluster/replication resource tree, the
// cluster-wide storage replication job schedule.
type Client struct {
	client Getter
}

// New returns a new replication client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// List retrieves all replication jobs the caller may see via
// GET /cluster/replication.
func (c *Client) List(ctx context.Context) ([]Job, error) {
	var jobs []Job
	if err := c.client.Get(ctx, "/cluster/replication", &jobs, nil); err != nil {
		return nil, err
	}

	return jobs, nil
}

// Get retrieves a single replication job via GET /cluster/replication/{id}.
func (c *Client) Get(ctx context.Context, id string) (*Job, error) {
	var job Job
	if err := c.client.Get(ctx, "/cluster/replication/"+id, &job, nil); err != nil {
		return nil, err
	}

	return &job, nil
}

// Create creates a new replication job via POST /cluster/replication.
func (c *Client) Create(ctx context.Context, opts *JobOptions) error {
	if opts == nil {
		return fmt.Errorf("replication: job options are required")
	}
	if opts.ID == "" {
		return fmt.Errorf("replication: job id is required")
	}
	if opts.Type == "" {
		return fmt.Errorf("replication: job type is required")
	}
	if opts.Target == "" {
		return fmt.Errorf("replication: job target is required")
	}

	params, err := opts.encode()
	if err != nil {
		return err
	}

	return c.client.Create(ctx, "/cluster/replication", nil, params)
}

// Update modifies an existing replication job via
// PUT /cluster/replication/{id}.
//
// opts.ID is ignored (overwritten with id) and opts.Type is dropped:
// Proxmox requires the job id in the update body too, even though it is
// already part of the URL, but rejects "type" there entirely. opts.Target
// is required — Proxmox treats it as fixed but still requires it resent,
// unchanged, on every Update.
func (c *Client) Update(ctx context.Context, id string, opts *JobOptions) error {
	if opts == nil {
		return fmt.Errorf("replication: job options are required")
	}
	if opts.Target == "" {
		return fmt.Errorf("replication: job target is required")
	}

	opts.ID = id

	params, err := opts.encode()
	if err != nil {
		return err
	}
	delete(params, "type")

	return c.client.Update(ctx, "/cluster/replication/"+id, nil, params)
}

// Delete removes a replication job via DELETE /cluster/replication/{id}.
//
// keep and force are mutually exclusive: keep leaves the already-replicated
// data on the target in place instead of removing it; force removes the
// job's configuration entry immediately without any cleanup (use when the
// target is unreachable and the normal cleanup would block).
func (c *Client) Delete(ctx context.Context, id string, keep, force bool) error {
	if keep && force {
		return fmt.Errorf("replication: keep and force are mutually exclusive")
	}

	var params map[string]string
	if keep || force {
		params = map[string]string{}
		if keep {
			params["keep"] = "1"
		}
		if force {
			params["force"] = "1"
		}
	}

	return c.client.Delete(ctx, "/cluster/replication/"+id, nil, params)
}
