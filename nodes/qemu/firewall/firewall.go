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

// Package firewall provides access to the Proxmox VE per-guest QEMU
// firewall API (endpoints under /nodes/{node}/qemu/{vmid}/firewall):
// rules, aliases, IP sets, options, the firewall log, and reference
// lookups.
//
// The resources themselves are implemented once, in internal/firewall,
// and shared with nodes/lxc/firewall and cluster/firewall: Proxmox backs
// all three scopes with the same PVE::Firewall::Rules/Aliases/IPSet
// modules. This package only supplies the guest's base path.
//
// This package's types are all re-exported (as aliases) from the shared
// types package rather than defined here — see its doc comment and
// docs/design.md §2. Rule/Alias/IPSet/IPSetEntry/Ref are the same wire
// format cluster/firewall and nodes/lxc/firewall use; Options/LogEntry/
// LogOptions are shared with nodes/lxc/firewall specifically, since
// Proxmox backs both the QEMU and LXC per-guest firewalls with the
// literal same handler code (PVE::API2::Firewall::VMBase) — but they
// differ from cluster/firewall's Options (cluster-wide firewalls have
// their own option set: enable, ebtables, policy_in/out/forward,
// log_ratelimit, with no macfilter/dhcp/ndp/radv/ipfilter/
// log_level_in/out), so that one stays local to cluster/firewall.
package firewall

import (
	"context"
	"strconv"

	fw "github.com/sergelogvinov/go-proxmox-rest/internal/firewall"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the firewall package
// decoupled from the root package (no import cycle).
type Getter = fw.Getter

// Client provides access to the /nodes/{node}/qemu/{vmid}/firewall
// resource tree, scoped to the node given to New. Every method/accessor
// takes the guest's VMID as a call argument, since this package has no
// persistent per-guest scope of its own.
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
	p := "/nodes/" + node + "/qemu/" + strconv.Itoa(vmid) + "/firewall"
	if sub != "" {
		p += "/" + sub
	}

	return p
}

// Options returns an accessor for
// GET/PUT /nodes/{node}/qemu/{vmid}/firewall/options, the guest's
// firewall configuration.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
// +proxmox:rbac:path=/vms/{vmid},method=PUT,privs=VM.Config.Network,match=all
func (c *Client) Options(vmid int) *fw.Options[Options] {
	return fw.NewOptions[Options](c.client, path(c.node, vmid, "options"))
}

// Rules returns an accessor for the guest's firewall rules under
// /nodes/{node}/qemu/{vmid}/firewall/rules.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Config.Network,match=all
// +proxmox:rbac:path=/vms/{vmid},method=PUT,privs=VM.Config.Network,match=all
// +proxmox:rbac:path=/vms/{vmid},method=DELETE,privs=VM.Config.Network,match=all
func (c *Client) Rules(vmid int) *fw.Rules {
	return fw.NewRules(c.client, path(c.node, vmid, "rules"))
}

// Aliases returns an accessor for
// /nodes/{node}/qemu/{vmid}/firewall/aliases, the guest's IP/network
// aliases.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Config.Network,match=all
// +proxmox:rbac:path=/vms/{vmid},method=PUT,privs=VM.Config.Network,match=all
// +proxmox:rbac:path=/vms/{vmid},method=DELETE,privs=VM.Config.Network,match=all
func (c *Client) Aliases(vmid int) *fw.Aliases {
	return fw.NewAliases(c.client, path(c.node, vmid, "aliases"))
}

// IPSet returns an accessor for
// /nodes/{node}/qemu/{vmid}/firewall/ipset, the guest's IP sets, and
// (via Entries) their members.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Config.Network,match=all
// +proxmox:rbac:path=/vms/{vmid},method=PUT,privs=VM.Config.Network,match=all
// +proxmox:rbac:path=/vms/{vmid},method=DELETE,privs=VM.Config.Network,match=all
func (c *Client) IPSet(vmid int) *fw.IPSets {
	return fw.NewIPSets(c.client, path(c.node, vmid, "ipset"))
}

// Refs retrieves the aliases and/or IP sets that may be referenced from
// a rule's Source/Dest fields (datacenter-, SDN-, and guest-scoped) via
// GET /nodes/{node}/qemu/{vmid}/firewall/refs. An empty refType returns
// both kinds; refType narrows the result to just aliases or just IP
// sets.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (c *Client) Refs(ctx context.Context, vmid int, refType RefType) ([]Ref, error) {
	return fw.Refs(ctx, c.client, path(c.node, vmid, "refs"), refType)
}

// Log retrieves the guest's firewall log entries (lines from
// /var/log/pve-firewall.log matching this guest's vmid) via
// GET /nodes/{node}/qemu/{vmid}/firewall/log. opts may be nil to
// request Proxmox's default window.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Console,match=all
func (c *Client) Log(ctx context.Context, vmid int, opts *LogOptions) ([]LogEntry, error) {
	return fw.Log(ctx, c.client, path(c.node, vmid, "log"), opts)
}
