// Package cluster provides access to the Proxmox VE cluster API
// (GET/POST endpoints under /cluster).
package cluster

import (
	"context"
)

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
	NodeStatus `json:",inline" url:",inline"`

	ID      string `json:"id,omitempty" url:"id,omitempty"`
	Name    string `json:"name,omitempty" url:"name,omitempty"`
	Type    string `json:"type,omitempty" url:"type,omitempty"`
	Quorate int    `json:"quorate,omitempty" url:"quorate,omitempty"`
	Version int    `json:"version,omitempty" url:"version,omitempty"`
}
