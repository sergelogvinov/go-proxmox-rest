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

// Package backup provides access to the Proxmox VE cluster-wide vzdump
// backup job API (endpoints under /cluster/backup).
package backup

import (
	"context"
	"fmt"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the backup package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /cluster/backup resource tree, the
// cluster-wide vzdump backup job schedule.
type Client struct {
	client Getter
}

// New returns a new backup client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// List retrieves all backup jobs via GET /cluster/backup.
func (c *Client) List(ctx context.Context) ([]Job, error) {
	var jobs []Job
	if err := c.client.Get(ctx, "/cluster/backup", &jobs, nil); err != nil {
		return nil, err
	}

	return jobs, nil
}

// Get retrieves a single backup job via GET /cluster/backup/{id}.
func (c *Client) Get(ctx context.Context, id string) (*Job, error) {
	var job Job
	if err := c.client.Get(ctx, "/cluster/backup/"+id, &job, nil); err != nil {
		return nil, err
	}

	return &job, nil
}

// Create creates a new backup job via POST /cluster/backup.
//
// opts.ID and opts.Schedule are required. Proxmox itself allows the id to
// be omitted (it then autogenerates one), but since Create's response is
// always empty there would be no way to learn the generated id, so this
// client requires callers to choose one upfront, matching every other
// Create in this module layout (pools, storage, ha).
func (c *Client) Create(ctx context.Context, opts *JobOptions) error {
	if opts == nil {
		return fmt.Errorf("backup: job options are required")
	}
	if opts.ID == "" {
		return fmt.Errorf("backup: job id is required")
	}
	if opts.Schedule == nil || *opts.Schedule == "" {
		return fmt.Errorf("backup: job schedule is required")
	}

	params, err := opts.encode()
	if err != nil {
		return err
	}

	return c.client.Create(ctx, "/cluster/backup", nil, params)
}

// Update modifies an existing backup job via PUT /cluster/backup/{id}.
func (c *Client) Update(ctx context.Context, id string, opts *JobOptions) error {
	params, err := opts.encode()
	if err != nil {
		return err
	}

	return c.client.Update(ctx, "/cluster/backup/"+id, nil, params)
}

// Delete removes a backup job via DELETE /cluster/backup/{id}.
func (c *Client) Delete(ctx context.Context, id string) error {
	return c.client.Delete(ctx, "/cluster/backup/"+id, nil, nil)
}
