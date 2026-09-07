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
func (c *Client) List(ctx context.Context) ([]Pool, error) {
	var pools []Pool
	if err := c.client.Get(ctx, "/pools", &pools, nil); err != nil {
		return nil, err
	}

	return pools, nil
}

// Create creates a new pool via POST /pools.
func (c *Client) Create(ctx context.Context, name string, opts *CreateOptions) error {
	params, err := opts.encode()
	if err != nil {
		return err
	}

	params["poolid"] = name

	return c.client.Create(ctx, "/pools", nil, params)
}

// Update modifies an existing pool via PUT /pools/?poolid={poolid}.
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
// The pool must be empty (no members); Proxmox refuses to delete a pool
// that still contains guests or storages unless force is set.
func (c *Client) Delete(ctx context.Context, poolID string, force bool) error {
	params := map[string]string{"poolid": poolID}
	if force {
		params["force"] = "1"
	}

	return c.client.Delete(ctx, "/pools/", nil, params)
}
