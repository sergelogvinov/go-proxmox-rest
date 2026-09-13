// Package cluster provides access to the Proxmox VE cluster API
// (GET/POST endpoints under /cluster).
package cluster

import (
	"context"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the cluster package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the cluster API section.
type Client struct {
	client Getter
}

// New returns a new cluster client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// Status retrieves the cluster status via GET /cluster/status.
//
// On a standalone (non-clustered) node the response contains only the
// node entry, and the cluster fields of Status are left zero-valued.
func (c *Client) Status(ctx context.Context) (*Status, error) {
	var entries []statusEntry
	if err := c.client.Get(ctx, "/cluster/status", &entries, nil); err != nil {
		return nil, err
	}

	status := &Status{}
	for _, e := range entries {
		switch e.Type {
		case "cluster":
			status.ID = e.ID
			status.Name = e.Name
			status.Quorate = e.Quorate
			status.Version = e.Version
		case "node":
			status.Nodes = append(status.Nodes, e.NodeStatus)
		}
	}

	return status, nil
}

// statusEntry mirrors the flat list returned by GET /cluster/status, where
// the cluster-level entry and per-node entries share the same shape.
type statusEntry struct {
	NodeStatus `json:",inline"`

	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Type    string `json:"type,omitempty"`
	Quorate int    `json:"quorate,omitempty"`
	Version int    `json:"version,omitempty"`
}
