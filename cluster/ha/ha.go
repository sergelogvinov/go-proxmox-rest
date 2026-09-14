// Package ha provides access to the Proxmox VE cluster high-availability API
// (endpoints under /cluster/ha).
package ha

import (
	"context"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the ha package decoupled from
// the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /cluster/ha resource tree, the cluster-wide
// high-availability configuration.
type Client struct {
	client Getter
}

// New returns a new ha client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}
