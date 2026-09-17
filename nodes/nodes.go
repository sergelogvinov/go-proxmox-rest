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

// Package nodes provides access to the Proxmox VE per-node API (endpoints
// under /nodes/{node}).
package nodes

import (
	"context"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/capabilities"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/ceph"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/hardware"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/network"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/replication"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/storage"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/tasks"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/vzdump"
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

// Client provides access to a single node's API section. The node's name
// is fixed at construction (via the root client's Nodes(node) call,
// k8s-clientset style), so every method and child accessor below operates
// on that node without repeating its name.
type Client struct {
	client Getter
	node   string
}

// New returns a new nodes client backed by the given root client, scoped
// to node.
func New(c Getter, node string) *Client {
	return &Client{client: c, node: node}
}

// Hardware returns an accessor for the /nodes/{node}/hardware resource
// tree, a node's local PCI and USB device inventories.
func (c *Client) Hardware() *hardware.Client {
	return hardware.New(c.client, c.node)
}

// Network returns an accessor for the /nodes/{node}/network resource tree,
// a node's network interface configuration.
func (c *Client) Network() *network.Client {
	return network.New(c.client, c.node)
}

// Capabilities returns an accessor for the /nodes/{node}/capabilities
// resource tree: available QEMU CPU models/flags, machine types, and
// migration capabilities.
func (c *Client) Capabilities() *capabilities.Client {
	return capabilities.New(c.client, c.node)
}

// Replication returns an accessor for the /nodes/{node}/replication
// resource tree: the runtime status and logs of storage replication jobs
// whose guest runs on that node.
func (c *Client) Replication() *replication.Client {
	return replication.New(c.client, c.node)
}

// Tasks returns an accessor for the /nodes/{node}/tasks resource tree: the
// node's task history, and a single task's log/status/stop.
func (c *Client) Tasks() *tasks.Client {
	return tasks.New(c.client, c.node)
}

// Storage returns an accessor for the /nodes/{node}/storage resource
// tree: each storage's status on that node, its content (volumes), and
// backup retention pruning.
func (c *Client) Storage() *storage.Client {
	return storage.New(c.client, c.node)
}

// Qemu returns an accessor for the /nodes/{node}/qemu/{vmid} resource
// tree: the guest's current status (Status) and power actions (Start,
// Stop, Reset, Shutdown, Reboot, Suspend, Resume), its configuration
// (Config, UpdateConfig, UpdateConfigAsync), and Clone/Template.
func (c *Client) Qemu() *qemu.Client {
	return qemu.New(c.client, c.node)
}

// LXC returns an accessor for the /nodes/{node}/lxc/{vmid} resource
// tree: the container's current status (Status) and power actions
// (Start, Stop, Shutdown, Reboot, Suspend, Resume — LXC has no "reset"
// action), its configuration (Config, UpdateConfig), and Clone/Template.
func (c *Client) LXC() *lxc.Client {
	return lxc.New(c.client, c.node)
}

// Ceph returns an accessor for the /nodes/{node}/ceph resource tree's
// basic service-lifecycle operations: cluster status, start/stop/
// restart, installable release listing, and the Ceph log.
func (c *Client) Ceph() *ceph.Client {
	return ceph.New(c.client, c.node)
}

// VZDump returns an accessor for the /nodes/{node}/vzdump resource tree:
// creating an on-demand backup job (Create), reading the node's
// configured backup defaults (Defaults), and extracting a guest's
// configuration from an existing backup archive (ExtractConfig).
func (c *Client) VZDump() *vzdump.Client {
	return vzdump.New(c.client, c.node)
}
