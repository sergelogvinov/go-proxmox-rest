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

// Package pools provides access to the Proxmox VE pool API
// (endpoints under /pools).
package pools

import (
	"context"
	"fmt"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the pools package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the pools API section.
type Client struct {
	client Getter
}

// New returns a new pools client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// Get retrieves a single pool configuration via GET /pools/?poolid={poolid}.
//
// +proxmox:rbac:path=/pool/{poolid},method=GET,privs=Pool.Audit,match=all
func (c *Client) Get(ctx context.Context, poolID string) (*Pool, error) {
	var pools []Pool
	if err := c.client.Get(ctx, "/pools/", &pools, map[string]string{"poolid": poolID}); err != nil {
		return nil, err
	}

	switch len(pools) {
	case 0:
		return nil, fmt.Errorf("pool not found")
	case 1:
		return &pools[0], nil
	}

	return nil, fmt.Errorf("multiple pools found for poolid: %s", poolID)
}

// List retrieves all pools via GET /pools.
//
// With verbose=false (default) only the pool IDs are returned. With
// verbose=true the full pool configuration (members and comment) is
// included for every pool.
// Proxmox filters the result to pools on which the caller has Pool.Audit.
//
// +proxmox:rbac:path=/pool/{poolid},method=GET,privs=Pool.Audit,match=all
func (c *Client) List(ctx context.Context) ([]Pool, error) {
	var pools []Pool
	if err := c.client.Get(ctx, "/pools", &pools, nil); err != nil {
		return nil, err
	}

	return pools, nil
}

// Create creates a new pool via POST /pools.
//
// +proxmox:rbac:path=/pool/{poolid},method=POST,privs=Pool.Allocate,match=all
func (c *Client) Create(ctx context.Context, name string, opts *CreateOptions) error {
	params, err := opts.encode()
	if err != nil {
		return err
	}

	params["poolid"] = name

	return c.client.Create(ctx, "/pools", nil, params)
}

// Update modifies an existing pool via PUT /pools/?poolid={poolid}: it
// changes the comment and/or adds or removes the guests/storages named
// in opts.VMIDs/opts.Storage (see UpdateOptions's doc comment — there is
// no whole-list replace).
//
// Adding or removing a guest additionally requires Permissions.Modify or
// VM.Allocate on /vms/{vmid}. Adding or removing storage requires
// Permissions.Modify or Datastore.Allocate on /storage/{storage}. Moving a
// guest from another pool (opts.AllowMove) also requires Pool.Allocate on
// that source pool.
//
// +proxmox:rbac:path=/pool/{poolid},method=PUT,privs=Pool.Allocate,match=all
func (c *Client) Update(ctx context.Context, poolID string, opts *UpdateOptions) error {
	params, err := opts.encode()
	if err != nil {
		return err
	}
	params["poolid"] = poolID

	return c.client.Update(ctx, "/pools/", nil, params)
}

// Delete removes a pool via DELETE /pools/?poolid={poolid}.
//
// The pool must be empty (no members) and have no nested sub-pools;
// Proxmox unconditionally refuses to delete a pool that still contains
// either — there is no force option to override this (unlike Update,
// which does support removing individual members). Remove every member
// via Update first if needed.
//
// +proxmox:rbac:path=/pool/{poolid},method=DELETE,privs=Pool.Allocate,match=all
func (c *Client) Delete(ctx context.Context, poolID string) error {
	return c.client.Delete(ctx, "/pools/", nil, map[string]string{"poolid": poolID})
}
