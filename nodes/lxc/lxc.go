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

// Package lxc provides access to the Proxmox VE per-node LXC container
// API (endpoints under /nodes/{node}/lxc/{vmid}). This currently covers
// the status resource tree (current status and the
// start/stop/shutdown/reboot/suspend/resume power actions — LXC has no
// "reset" action, unlike QEMU), config (read/update the container's
// configuration), interfaces (the container's live network addresses),
// feature (capability checks), migrate (precondition check and the
// migration itself), resize (grow a mount point), clone, template, and
// the per-guest firewall (nodes/lxc/firewall, behind the Firewall()
// accessor) — all but the latter folded directly onto Client rather than
// behind per-resource accessors, mirroring the nodes/qemu package's
// shape. The larger snapshot/... surface is left for a future addition.
package lxc

import (
	"context"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc/firewall"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the lxc package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /nodes/{node}/lxc/{vmid} resource tree,
// scoped to the node given to New. Every method takes the container's
// VMID as a call argument, since this package has no persistent
// per-container scope of its own.
type Client struct {
	client Getter
	node   string
}

// New returns a new lxc client backed by the given root client, scoped
// to node.
func New(c Getter, node string) *Client {
	return &Client{client: c, node: node}
}

// Firewall returns an accessor for the
// /nodes/{node}/lxc/{vmid}/firewall resource tree: rules, aliases, IP
// sets, options, the firewall log, and reference lookups.
func (c *Client) Firewall() *firewall.Client {
	return firewall.New(c.client, c.node)
}
