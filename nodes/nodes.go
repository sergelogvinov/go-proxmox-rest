// Package nodes provides access to the Proxmox VE per-node API (endpoints
// under /nodes/{node}).
package nodes

import (
	"context"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the nodes package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the nodes API section. Every method takes the
// target node's name as a call argument (matching the fluent chain's
// filter-as-argument shape, e.g. cluster.Client.Resources().Get(ctx, type)),
// rather than a separate chain setter.
type Client struct {
	client Getter
}

// New returns a new nodes client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}
