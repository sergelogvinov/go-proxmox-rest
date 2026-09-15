// Package ceph provides access to the Proxmox VE cluster-wide Ceph API
// (endpoints under /cluster/ceph).
package ceph

import (
	"context"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the ceph package decoupled from
// the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /cluster/ceph resource tree, the
// cluster-wide Ceph status and configuration.
type Client struct {
	client Getter
}

// New returns a new ceph client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// Status retrieves the cluster-wide Ceph status via
// GET /cluster/ceph/status.
func (c *Client) Status(ctx context.Context) (*Status, error) {
	var status Status
	if err := c.client.Get(ctx, "/cluster/ceph/status", &status, nil); err != nil {
		return nil, err
	}

	return &status, nil
}
