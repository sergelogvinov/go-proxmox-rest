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

// Package firewall provides access to the Proxmox VE per-guest LXC
// firewall API (endpoints under /nodes/{node}/lxc/{vmid}/firewall):
// rules, aliases, IP sets, options, the firewall log, and reference
// lookups.
//
// This package's types are all re-exported (as aliases) from the shared
// types package rather than defined here — see its doc comment and
// docs/design.md §2. Every one of them (including Options/LogEntry/
// LogOptions) is identical to nodes/qemu/firewall's: Proxmox registers
// the literal same handler code (PVE::API2::Firewall::VMBase) for both
// the QEMU and LXC per-guest firewalls, just mounted under different
// paths, so this package's shape (and this file's Client methods) mirror
// nodes/qemu/firewall's exactly.
package firewall

import (
	"context"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the firewall package
// decoupled from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /nodes/{node}/lxc/{vmid}/firewall
// resource tree, scoped to the node given to New. Every method/accessor
// takes the container's VMID as a call argument, since this package has
// no persistent per-guest scope of its own.
type Client struct {
	client Getter
	node   string
}

// New returns a new firewall client backed by the given root client,
// scoped to node.
func New(c Getter, node string) *Client {
	return &Client{client: c, node: node}
}

// path builds the .../firewall/{sub} URL for the given node and vmid.
func path(node string, vmid int, sub string) string {
	p := "/nodes/" + node + "/lxc/" + strconv.Itoa(vmid) + "/firewall"
	if sub != "" {
		p += "/" + sub
	}

	return p
}

// Options returns an accessor for
// GET/PUT /nodes/{node}/lxc/{vmid}/firewall/options, the guest's
// firewall configuration.
func (c *Client) Options(vmid int) *optionsResource {
	return &optionsResource{client: c.client, path: path(c.node, vmid, "options")}
}

// Rules returns an accessor for the guest's firewall rules under
// /nodes/{node}/lxc/{vmid}/firewall/rules.
func (c *Client) Rules(vmid int) *ruleResource {
	return &ruleResource{client: c.client, base: path(c.node, vmid, "rules")}
}

// Aliases returns an accessor for
// /nodes/{node}/lxc/{vmid}/firewall/aliases, the guest's IP/network
// aliases.
func (c *Client) Aliases(vmid int) *aliasesResource {
	return &aliasesResource{client: c.client, base: path(c.node, vmid, "aliases")}
}

// IPSet returns an accessor for
// /nodes/{node}/lxc/{vmid}/firewall/ipset, the guest's IP sets.
func (c *Client) IPSet(vmid int) *ipsetResource {
	return &ipsetResource{client: c.client, base: path(c.node, vmid, "ipset")}
}

// Refs retrieves the aliases and/or IP sets that may be referenced from
// a rule's Source/Dest fields (datacenter-, SDN-, and guest-scoped) via
// GET /nodes/{node}/lxc/{vmid}/firewall/refs. An empty refType returns
// both kinds; refType narrows the result to just aliases or just IP
// sets.
func (c *Client) Refs(ctx context.Context, vmid int, refType RefType) ([]Ref, error) {
	var p map[string]string
	if refType != "" {
		p = map[string]string{"type": string(refType)}
	}

	var refs []Ref
	if err := c.client.Get(ctx, path(c.node, vmid, "refs"), &refs, p); err != nil {
		return nil, err
	}

	return refs, nil
}

// Log retrieves the guest's firewall log entries (lines from
// /var/log/pve-firewall.log matching this guest's vmid) via
// GET /nodes/{node}/lxc/{vmid}/firewall/log. opts may be nil to
// request Proxmox's default window.
func (c *Client) Log(ctx context.Context, vmid int, opts *LogOptions) ([]LogEntry, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var entries []LogEntry
	if err := c.client.Get(ctx, path(c.node, vmid, "log"), &entries, p); err != nil {
		return nil, err
	}

	return entries, nil
}
