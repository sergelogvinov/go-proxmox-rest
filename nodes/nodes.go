// Package nodes provides access to the Proxmox VE per-node API (endpoints
// under /nodes/{node}).
package nodes

import (
	"context"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/capabilities"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/hardware"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/network"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/replication"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/tasks"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the nodes package decoupled
// from the root package (no import cycle).
//
// Create/Delete are carried here, even though only nodes/network needs
// them so far, for the same reason cluster.Getter carries CreateValues/
// UpdateValues: so every child package can be wired in from the single
// c.client value below.
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
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

// Hardware returns an accessor for the /nodes/{node}/hardware resource
// tree, a node's local PCI and USB device inventories.
func (c *Client) Hardware() *hardware.Client {
	return hardware.New(c.client)
}

// Network returns an accessor for the /nodes/{node}/network resource tree,
// a node's network interface configuration.
func (c *Client) Network() *network.Client {
	return network.New(c.client)
}

// Capabilities returns an accessor for the /nodes/{node}/capabilities
// resource tree: available QEMU CPU models/flags, machine types, and
// migration capabilities.
func (c *Client) Capabilities() *capabilities.Client {
	return capabilities.New(c.client)
}

// Replication returns an accessor for the /nodes/{node}/replication
// resource tree: the runtime status and logs of storage replication jobs
// whose guest runs on that node.
func (c *Client) Replication() *replication.Client {
	return replication.New(c.client)
}

// Tasks returns an accessor for the /nodes/{node}/tasks resource tree: the
// node's task history, and a single task's log/status/stop.
func (c *Client) Tasks() *tasks.Client {
	return tasks.New(c.client)
}
