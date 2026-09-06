// Package storage provides access to the Proxmox VE storage API
// (endpoints under /storage).
package storage

import (
	"context"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the storage package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the storage API section.
type Client struct {
	client Getter
}

// New returns a new storage client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// List retrieves all storages via GET /storage.
//
// The optional type filter restricts the result to a single plugin type
// (e.g. "dir", "zfs", "nfs"); an empty string returns all storages.
func (c *Client) List(ctx context.Context, storageType string) ([]Storage, error) {
	var params map[string]string
	if storageType != "" {
		params = map[string]string{"type": storageType}
	}

	var storages []Storage
	if err := c.client.Get(ctx, "/storage", &storages, params); err != nil {
		return nil, err
	}

	return storages, nil
}

// Get retrieves a single storage configuration via GET /storage/{storage}.
func (c *Client) Get(ctx context.Context, storageID string) (*Storage, error) {
	var storage Storage
	if err := c.client.Get(ctx, "/storage/"+storageID, &storage, nil); err != nil {
		return nil, err
	}

	return &storage, nil
}

// Create creates a new storage via POST /storage.
func (c *Client) Create(ctx context.Context, opts *CreateOptions) error {
	params, err := opts.encode()
	if err != nil {
		return err
	}

	return c.client.Create(ctx, "/storage", nil, params)
}

// Update modifies an existing storage via PUT /storage/{storage}.
func (c *Client) Update(ctx context.Context, storageID string, opts *UpdateOptions) error {
	params, err := opts.encode()
	if err != nil {
		return err
	}

	return c.client.Update(ctx, "/storage/"+storageID, nil, params)
}

// Delete removes a storage via DELETE /storage/{storage}.
//
// The storage must not be in use; Proxmox refuses to delete a storage
// that still holds references.
func (c *Client) Delete(ctx context.Context, storageID string) error {
	return c.client.Delete(ctx, "/storage/"+storageID, nil, nil)
}
