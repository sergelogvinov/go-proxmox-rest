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

// Package firewall provides access to the Proxmox VE cluster firewall API
// (endpoints under /cluster/firewall).
//
// Rules, aliases, IP sets and options are implemented once, in
// internal/firewall, and shared with the per-guest nodes/qemu/firewall and
// nodes/lxc/firewall packages; this package supplies the /cluster/firewall
// base path, the cluster-only security groups, and its own Options type.
package firewall

import (
	"context"

	fw "github.com/sergelogvinov/go-proxmox-rest/internal/firewall"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the firewall package decoupled
// from the root package (no import cycle).
type Getter = fw.Getter

// Client provides access to the /cluster/firewall resource tree, the
// cluster-wide firewall configuration (rules, security groups, aliases, IP
// sets and options).
type Client struct {
	client Getter
}

// New returns a new firewall client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// Options returns an accessor for GET/PUT /cluster/firewall/options, the
// cluster-wide firewall configuration.
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Audit,match=all
// +proxmox:rbac:path=/,method=PUT,privs=Sys.Modify,match=all
func (c *Client) Options() *fw.Options[Options] {
	return fw.NewOptions[Options](c.client, "/cluster/firewall/options")
}

// Rules returns an accessor for the cluster-wide firewall rules under
// /cluster/firewall/rules. For a security group's rules, see
// Groups().Rules(name).
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Audit,match=all
// +proxmox:rbac:path=/,method=POST,privs=Sys.Modify,match=all
// +proxmox:rbac:path=/,method=PUT,privs=Sys.Modify,match=all
// +proxmox:rbac:path=/,method=DELETE,privs=Sys.Modify,match=all
func (c *Client) Rules() *fw.Rules {
	return fw.NewRules(c.client, "/cluster/firewall/rules")
}

// Groups returns an accessor for the /cluster/firewall/groups resource,
// cluster-wide security groups.
func (c *Client) Groups() *groupsResource {
	return &groupsResource{client: c.client}
}

// Aliases returns an accessor for the /cluster/firewall/aliases resource,
// cluster-wide IP/network aliases.
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Audit,match=all
// +proxmox:rbac:path=/,method=POST,privs=Sys.Modify,match=all
// +proxmox:rbac:path=/,method=PUT,privs=Sys.Modify,match=all
// +proxmox:rbac:path=/,method=DELETE,privs=Sys.Modify,match=all
func (c *Client) Aliases() *fw.Aliases {
	return fw.NewAliases(c.client, "/cluster/firewall/aliases")
}

// IPSet returns an accessor for the /cluster/firewall/ipset resource,
// cluster-wide IP sets, and (via Entries) their members.
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Audit,match=all
// +proxmox:rbac:path=/,method=POST,privs=Sys.Modify,match=all
// +proxmox:rbac:path=/,method=PUT,privs=Sys.Modify,match=all
// +proxmox:rbac:path=/,method=DELETE,privs=Sys.Modify,match=all
func (c *Client) IPSet() *fw.IPSets {
	return fw.NewIPSets(c.client, "/cluster/firewall/ipset")
}

// Refs retrieves the aliases and/or IP sets that may be referenced from a
// rule's Source/Dest fields via GET /cluster/firewall/refs. An empty
// refType returns both kinds; refType narrows the result to just aliases
// or just IP sets.
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Audit,match=all
func (c *Client) Refs(ctx context.Context, refType RefType) ([]Ref, error) {
	return fw.Refs(ctx, c.client, "/cluster/firewall/refs", refType)
}
