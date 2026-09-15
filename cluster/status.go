// Package cluster provides access to the Proxmox VE cluster API
// (GET/POST endpoints under /cluster).
package cluster

import (
	"context"
	"net/url"

	"github.com/sergelogvinov/go-proxmox-rest/cluster/backup"
	"github.com/sergelogvinov/go-proxmox-rest/cluster/ceph"
	"github.com/sergelogvinov/go-proxmox-rest/cluster/firewall"
	"github.com/sergelogvinov/go-proxmox-rest/cluster/ha"
	"github.com/sergelogvinov/go-proxmox-rest/cluster/mapping"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the cluster package decoupled
// from the root package (no import cycle).
//
// CreateValues/UpdateValues (url.Values, supporting multiple values under
// the same key) are carried here, even though only cluster/mapping needs
// them, for the same reason Create/Update/Delete are: so every child
// package can be wired in from the single c.client value below.
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
	CreateValues(ctx context.Context, path string, out any, params url.Values) error
	UpdateValues(ctx context.Context, path string, out any, params url.Values) error
}

// Client provides access to the cluster API section.
type Client struct {
	client Getter
}

// New returns a new cluster client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// HA returns an accessor for the /cluster/ha resource tree, the
// cluster-wide high-availability configuration.
func (c *Client) HA() *ha.Client {
	return ha.New(c.client)
}

// Firewall returns an accessor for the /cluster/firewall resource tree,
// the cluster-wide firewall configuration.
func (c *Client) Firewall() *firewall.Client {
	return firewall.New(c.client)
}

// Backup returns an accessor for the /cluster/backup resource tree, the
// cluster-wide vzdump backup job schedule.
func (c *Client) Backup() *backup.Client {
	return backup.New(c.client)
}

// Ceph returns an accessor for the /cluster/ceph resource tree, the
// cluster-wide Ceph status and configuration.
func (c *Client) Ceph() *ceph.Client {
	return ceph.New(c.client)
}

// Mapping returns an accessor for the /cluster/mapping resource tree, the
// cluster-wide hardware mappings (PCI, USB, directory) shared by name
// across nodes.
func (c *Client) Mapping() *mapping.Client {
	return mapping.New(c.client)
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
	NodeStatus `json:",inline" url:",inline"`

	ID      string `json:"id,omitempty" url:"id,omitempty"`
	Name    string `json:"name,omitempty" url:"name,omitempty"`
	Type    string `json:"type,omitempty" url:"type,omitempty"`
	Quorate int    `json:"quorate,omitempty" url:"quorate,omitempty"`
	Version int    `json:"version,omitempty" url:"version,omitempty"`
}
