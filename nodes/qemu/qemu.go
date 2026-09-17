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

// Package qemu provides access to the Proxmox VE per-node QEMU guest API
// (endpoints under /nodes/{node}/qemu/{vmid}). This currently covers the
// status resource tree (current status and the
// start/stop/reset/shutdown/reboot/suspend/resume power actions),
// config (read/update the guest's configuration), clone, template,
// migrate (precondition check and the migration task itself), feature
// (capability checks), move_disk, resize, unlink, cloudinit
// (pending values and regeneration), snapshot (behind the Snapshot()
// accessor), the per-guest firewall (nodes/qemu/firewall, behind the
// Firewall() accessor), and the QEMU Guest Agent (nodes/qemu/agent,
// behind the Agent() accessor) — the latter two large enough to earn
// their own subpackages, unlike the rest of this package's flat,
// direct-on-Client shape.
package qemu

import (
	"context"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu/agent"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu/firewall"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the qemu package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /nodes/{node}/qemu/{vmid} resource tree,
// scoped to the node given to New. Every method takes the guest's VMID as
// a call argument, since this package has no persistent per-guest scope
// of its own.
type Client struct {
	client Getter
	node   string
}

// New returns a new qemu client backed by the given root client, scoped
// to node.
func New(c Getter, node string) *Client {
	return &Client{client: c, node: node}
}

// Agent returns an accessor for the
// /nodes/{node}/qemu/{vmid}/agent resource tree: the QEMU Guest Agent
// command surface (filesystem freeze/trim, network/user/OS info,
// exec, file read/write, ...).
func (c *Client) Agent() *agent.Client {
	return agent.New(c.client, c.node)
}

// Snapshot returns an accessor for the
// /nodes/{node}/qemu/{vmid}/snapshot resource tree: listing, creating,
// and deleting snapshots, reading/updating a snapshot's metadata, and
// rolling back to one.
func (c *Client) Snapshot() *snapshotResource {
	return &snapshotResource{client: c.client, node: c.node}
}

// Firewall returns an accessor for the
// /nodes/{node}/qemu/{vmid}/firewall resource tree: rules, aliases, IP
// sets, options, the firewall log, and reference lookups.
func (c *Client) Firewall() *firewall.Client {
	return firewall.New(c.client, c.node)
}
