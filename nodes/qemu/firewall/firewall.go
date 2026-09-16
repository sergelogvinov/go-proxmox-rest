// Package firewall provides access to the Proxmox VE per-guest QEMU
// firewall API (endpoints under /nodes/{node}/qemu/{vmid}/firewall):
// rules, aliases, IP sets, options, the firewall log, and reference
// lookups.
//
// This package's Rule/Alias/IPSet/IPSetEntry/Ref shapes (and their write
// options) are a deliberate duplicate of cluster/firewall's — Proxmox
// backs both cluster-wide and per-guest firewalls with the same
// PVE::Firewall::Rules/Aliases/IPSet modules, so the wire format is
// identical. They're duplicated rather than imported because child
// packages depend only on the root package, never on each other (see
// docs/design.md §2). Options is genuinely different: guest firewalls
// have their own option set (enable, macfilter, dhcp, ndp, radv,
// ipfilter, policy_in/out, log_level_in/out) with no ebtables/
// policy_forward/log_ratelimit.
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

// Client provides access to the /nodes/{node}/qemu/{vmid}/firewall
// resource tree. Every method/accessor takes the target node's name
// and guest's VMID as call arguments, since this package has no
// persistent per-guest scope of its own.
type Client struct {
	client Getter
}

// New returns a new firewall client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
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
func (c *Client) Options(node string, vmid int) *optionsResource {
	return &optionsResource{client: c.client, path: path(node, vmid, "options")}
}

// Rules returns an accessor for the guest's firewall rules under
// /nodes/{node}/qemu/{vmid}/firewall/rules.
func (c *Client) Rules(node string, vmid int) *ruleResource {
	return &ruleResource{client: c.client, base: path(node, vmid, "rules")}
}

// Aliases returns an accessor for
// /nodes/{node}/qemu/{vmid}/firewall/aliases, the guest's IP/network
// aliases.
func (c *Client) Aliases(node string, vmid int) *aliasesResource {
	return &aliasesResource{client: c.client, base: path(node, vmid, "aliases")}
}

// IPSet returns an accessor for
// /nodes/{node}/qemu/{vmid}/firewall/ipset, the guest's IP sets.
func (c *Client) IPSet(node string, vmid int) *ipsetResource {
	return &ipsetResource{client: c.client, base: path(node, vmid, "ipset")}
}

// Refs retrieves the aliases and/or IP sets that may be referenced from
// a rule's Source/Dest fields (datacenter-, SDN-, and guest-scoped) via
// GET /nodes/{node}/qemu/{vmid}/firewall/refs. An empty refType returns
// both kinds; refType narrows the result to just aliases or just IP
// sets.
func (c *Client) Refs(ctx context.Context, node string, vmid int, refType RefType) ([]Ref, error) {
	var p map[string]string
	if refType != "" {
		p = map[string]string{"type": string(refType)}
	}

	var refs []Ref
	if err := c.client.Get(ctx, path(node, vmid, "refs"), &refs, p); err != nil {
		return nil, err
	}

	return refs, nil
}

// Log retrieves the guest's firewall log entries (lines from
// /var/log/pve-firewall.log matching this guest's vmid) via
// GET /nodes/{node}/qemu/{vmid}/firewall/log. opts may be nil to
// request Proxmox's default window.
func (c *Client) Log(ctx context.Context, node string, vmid int, opts *LogOptions) ([]LogEntry, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var entries []LogEntry
	if err := c.client.Get(ctx, path(node, vmid, "log"), &entries, p); err != nil {
		return nil, err
	}

	return entries, nil
}
