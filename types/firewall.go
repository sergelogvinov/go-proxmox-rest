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

// Package types holds wire-format types shared by sibling API subpackages
// that cannot import each other (see the dependency-direction rule in
// docs/design.md §2): a plain leaf package with no dependency on the root
// package or on any subpackage, so both sides of a shared shape can import
// it without creating a cycle.
//
// firewall.go holds the rule/alias/IP-set shapes Proxmox reuses verbatim
// across every firewall scope it exposes (cluster-wide, per-guest, ...):
// PVE::Firewall::Rules/Aliases/IPSet back all of them with the same wire
// format. cluster/firewall, nodes/qemu/firewall and nodes/lxc/firewall all
// re-export these via type aliases.
//
// GuestOptions/LogEntry/LogOptions are a second tier of sharing: Proxmox
// backs both the QEMU and LXC per-guest firewalls with the literal same
// Perl handler (PVE::API2::Firewall::VMBase, registered once for VMs and
// once for CTs with an identical option schema and log format), so
// nodes/qemu/firewall and nodes/lxc/firewall alias GuestOptions as their
// own Options — but cluster/firewall's Options is a genuinely different
// schema (cluster-wide firewalls have their own option set) and stays
// local to that package, unshared.
package types

import (
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// RuleType is the firewall rule kind: which traffic direction a rule
// applies to, or whether it references a security group.
type RuleType string

const (
	// RuleTypeIn matches inbound traffic.
	RuleTypeIn RuleType = "in"
	// RuleTypeOut matches outbound traffic.
	RuleTypeOut RuleType = "out"
	// RuleTypeForward matches forwarded traffic.
	RuleTypeForward RuleType = "forward"
	// RuleTypeGroup references a security group's rule set.
	RuleTypeGroup RuleType = "group"
)

// RuleLogLevel is the log level applied to a firewall rule's matches.
type RuleLogLevel string

const (
	RuleLogEmerg   RuleLogLevel = "emerg"
	RuleLogAlert   RuleLogLevel = "alert"
	RuleLogCrit    RuleLogLevel = "crit"
	RuleLogErr     RuleLogLevel = "err"
	RuleLogWarning RuleLogLevel = "warning"
	RuleLogNotice  RuleLogLevel = "notice"
	RuleLogInfo    RuleLogLevel = "info"
	RuleLogDebug   RuleLogLevel = "debug"
	RuleLogNolog   RuleLogLevel = "nolog"
)

// Policy is a firewall default-policy verdict.
type Policy string

const (
	// PolicyAccept accepts traffic that reaches the default policy.
	PolicyAccept Policy = "ACCEPT"
	// PolicyReject rejects traffic (with a response) that reaches the
	// default policy. Not valid for a forward policy.
	PolicyReject Policy = "REJECT"
	// PolicyDrop silently drops traffic that reaches the default policy.
	PolicyDrop Policy = "DROP"
)

// Rule describes a single firewall rule, returned by every scope's
// GET .../firewall/rules and GET .../firewall/rules/{pos} (and, for
// cluster-wide security groups, GET /cluster/firewall/groups/{group} and
// GET /cluster/firewall/groups/{group}/{pos}).
//
// Every field carries a `url` tag matching its wire name even though Rule
// is read-only and never passed to params.Encode: each firewall package's
// decode path (internal/params.Decode, used by every GET) resolves each
// field's wire name from the `url` tag, not `json`. The `json` tags are
// kept alongside for readability/documentation and are otherwise inert
// here (decode ignores them).
type Rule struct {
	// Pos is the rule's position within its rule list. It addresses the
	// rule for Get/Update/Delete.
	Pos int `json:"pos" url:"pos"`
	// Type is the rule kind: "in", "out", "forward" or "group".
	Type RuleType `json:"type,omitempty" url:"type,omitempty"`
	// Action is "ACCEPT", "DROP", "REJECT", or (for Type ==
	// RuleTypeGroup) the name of a security group to apply.
	Action string `json:"action,omitempty" url:"action,omitempty"`
	// Enable is 1 if the rule is enabled.
	Enable int `json:"enable,omitempty" url:"enable,omitempty"`
	// Comment is the rule description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
	// Source restricts the packet source address: a single IP, an IP
	// range, a CIDR, an alias ('alias-name') or an IP set ('+ipset-name').
	Source string `json:"source,omitempty" url:"source,omitempty"`
	// Dest restricts the packet destination address, same syntax as
	// Source.
	Dest string `json:"dest,omitempty" url:"dest,omitempty"`
	// Proto is the IP protocol, e.g. "tcp", "udp", or a protocol number.
	Proto string `json:"proto,omitempty" url:"proto,omitempty"`
	// SPort restricts the TCP/UDP source port (a service name, a number,
	// or a range "start:end").
	SPort string `json:"sport,omitempty" url:"sport,omitempty"`
	// DPort restricts the TCP/UDP destination port, same syntax as SPort.
	DPort string `json:"dport,omitempty" url:"dport,omitempty"`
	// IFace restricts the rule to a network interface (a guest's net key
	// name, e.g. "net0", for per-guest rules; a physical interface name
	// for cluster-wide rules).
	IFace string `json:"iface,omitempty" url:"iface,omitempty"`
	// ICMPType restricts the rule to an ICMP type; only meaningful when
	// Proto is "icmp" or "icmpv6"/"ipv6-icmp".
	ICMPType string `json:"icmp-type,omitempty" url:"icmp-type,omitempty"`
	// Macro applies a predefined standard macro (e.g. "SSH", "HTTPS")
	// instead of specifying Proto/DPort manually.
	Macro string `json:"macro,omitempty" url:"macro,omitempty"`
	// Log is the log level applied to matches of this rule.
	Log RuleLogLevel `json:"log,omitempty" url:"log,omitempty"`
	// IPVersion is the IP version the rule was resolved against (4 or 6);
	// set by Proxmox, not settable.
	IPVersion int `json:"ipversion,omitempty" url:"ipversion,omitempty"`
	// Digest is the configuration digest, usable with RuleOptions.Digest
	// to guard concurrent updates.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// RuleOptions holds the write parameters shared by rule creation
// (POST .../firewall/rules, or POST /cluster/firewall/groups/{group} for a
// security group) and rule updates (PUT .../rules/{pos}, or
// PUT /cluster/firewall/groups/{group}/{pos}).
//
// Type and Action are required by Create. Proxmox does not echo the created
// rule back (POST responds with null data) and does not report the
// position it was assigned, so after Create use List to find the new rule
// — it is appended at the end unless Pos is set to insert it elsewhere.
//
// Pointer fields are always sent when non-nil (even zero/empty), which is
// how Update expresses "clear this field"; Create simply sends whatever is
// set. Pos, MoveTo, Delete and Digest are meaningful to specific verbs only
// (see field docs).
type RuleOptions struct {
	// Type is the rule kind: "in", "out", "forward" or "group". Required
	// by Create; Proxmox also requires it to be resent on every Update.
	Type RuleType `url:"type"`
	// Action is "ACCEPT", "DROP", "REJECT", or a security group name.
	// Required by Create; Proxmox also requires it to be resent on every
	// Update.
	Action string `url:"action"`
	// Enable enables/disables the rule.
	Enable *bool `url:"enable"`
	// Comment is the rule description.
	Comment *string `url:"comment"`
	// Source restricts the packet source address.
	Source *string `url:"source"`
	// Dest restricts the packet destination address.
	Dest *string `url:"dest"`
	// Proto is the IP protocol.
	Proto *string `url:"proto"`
	// SPort restricts the TCP/UDP source port.
	SPort *string `url:"sport"`
	// DPort restricts the TCP/UDP destination port.
	DPort *string `url:"dport"`
	// IFace restricts the rule to a network interface.
	IFace *string `url:"iface"`
	// ICMPType restricts the rule to an ICMP type.
	ICMPType *string `url:"icmp-type"`
	// Macro applies a predefined standard macro.
	Macro *string `url:"macro"`
	// Log is the log level applied to matches of this rule.
	Log *RuleLogLevel `url:"log"`
	// Pos is, on Create, the position to insert the new rule at (existing
	// rules from that position on shift down); Proxmox appends to the end
	// when unset. Not meaningful on Update — use MoveTo to reposition an
	// existing rule instead.
	Pos *int `url:"pos"`
	// MoveTo is, on Update only, the new position to move the rule to.
	// Proxmox ignores every other field in the same call when this is
	// set — issue a separate Update to change other fields.
	MoveTo *int `url:"moveto"`
	// Delete lists properties to reset to their default value (Update
	// only).
	Delete []string `url:"delete"`
	// Digest prevents changes if the current configuration has changed
	// in between (value from the corresponding Get). Update/Delete only.
	Digest string `url:"digest"`
}

// Encode converts the options to form parameters.
func (o *RuleOptions) Encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("firewall: rule options are required")
	}

	return params.Encode(o)
}

// Alias describes an IP/network alias, returned by every scope's
// GET .../firewall/aliases and GET .../firewall/aliases/{name}.
type Alias struct {
	// Name is the alias identifier, e.g. "my-alias".
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// CIDR is the network/IP the alias resolves to, e.g. "10.0.0.0/24".
	CIDR string `json:"cidr,omitempty" url:"cidr,omitempty"`
	// Comment is the alias description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
	// IPVersion is the IP version of CIDR (4 or 6); set by Proxmox.
	IPVersion int `json:"ipversion,omitempty" url:"ipversion,omitempty"`
	// Digest is the configuration digest, usable with AliasOptions.Digest
	// to guard concurrent updates.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// AliasOptions holds the write parameters shared by
// POST .../firewall/aliases (Create) and PUT .../firewall/aliases/{name}
// (Update).
//
// Name and CIDR are required by Create. Proxmox requires CIDR to be resent
// on every Update even when unchanged. Update is addressed by the alias's
// current name in the URL (see aliasesResource.Update); set Rename to give
// it a new name.
type AliasOptions struct {
	// Name is the alias identifier. Required by Create; on Update it is
	// ignored (the current name is already part of the URL) — use Rename
	// to change it.
	Name string `url:"name"`
	// CIDR is the network/IP the alias resolves to. Required by both
	// Create and Update.
	CIDR string `url:"cidr"`
	// Comment is the alias description.
	Comment *string `url:"comment"`
	// Rename is, on Update only, the new name to give the alias.
	Rename *string `url:"rename"`
	// Digest prevents changes if the current configuration has changed
	// in between. Update/Delete only.
	Digest string `url:"digest"`
}

// Encode converts the options to form parameters.
func (o *AliasOptions) Encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("firewall: alias options are required")
	}

	return params.Encode(o)
}

// IPSet describes an IP set, returned by every scope's
// GET .../firewall/ipset. It carries only the set's own metadata; its
// members are accessed via the owning package's Client.IPSet(...).Entries.
type IPSet struct {
	// Name is the IP set identifier, e.g. "my-ipset".
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Comment is the IP set description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
	// Digest is the configuration digest, usable with IPSetOptions.Digest
	// to guard concurrent updates.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// IPSetOptions holds the write parameters for POST .../firewall/ipset.
// Proxmox has no dedicated update endpoint for an IP set's own metadata:
// creation and rename/re-comment both go through this same POST,
// distinguished by the Rename field — see ipsetResource.Update.
type IPSetOptions struct {
	// Name is the IP set's target name: the name to create (Create), or
	// the new name (Update — set equal to the current name to update
	// Comment without renaming).
	Name string `url:"name"`
	// Comment is the IP set description.
	Comment *string `url:"comment"`
	// Rename is, on Update only, the IP set's current name (set by
	// ipsetResource.Update; callers do not set this directly).
	Rename string `url:"rename"`
	// Digest prevents changes if the current configuration has changed
	// in between. Update only.
	Digest string `url:"digest"`
}

// Encode converts the options to form parameters.
func (o *IPSetOptions) Encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("firewall: ipset options are required")
	}

	return params.Encode(o)
}

// IPSetEntry describes a single member of an IP set, returned by every
// scope's GET .../firewall/ipset/{name} and
// GET .../firewall/ipset/{name}/{cidr}.
type IPSetEntry struct {
	// CIDR is the network/IP of this member, e.g. "10.0.0.0/24".
	CIDR string `json:"cidr,omitempty" url:"cidr,omitempty"`
	// Comment is the member description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
	// NoMatch excludes this CIDR from the set (a "negative" member),
	// letting it carve out an exception within a broader range also in
	// the set.
	NoMatch bool `json:"nomatch,omitempty" url:"nomatch,omitempty"`
	// Digest is the configuration digest, usable with
	// IPSetEntryOptions.Digest to guard concurrent updates.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// IPSetEntryOptions holds the write parameters shared by
// POST .../firewall/ipset/{name} (Create) and
// PUT .../firewall/ipset/{name}/{cidr} (Update).
//
// CIDR is required by Create and, like Alias, must be resent on every
// Update (Update is addressed by the entry's current CIDR in the URL; set
// CIDR here only if it differs, but Proxmox documents it as required
// regardless).
type IPSetEntryOptions struct {
	// CIDR is the network/IP of this member. Required by both Create and
	// Update.
	CIDR string `url:"cidr"`
	// Comment is the member description.
	Comment *string `url:"comment"`
	// NoMatch excludes this CIDR from the set.
	NoMatch *bool `url:"nomatch"`
	// Digest prevents changes if the current configuration has changed
	// in between. Update/Delete only.
	Digest string `url:"digest"`
}

// Encode converts the options to form parameters.
func (o *IPSetEntryOptions) Encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("firewall: ipset entry options are required")
	}

	return params.Encode(o)
}

// RefType filters a scope's Client.Refs.
type RefType string

const (
	// RefTypeAlias lists only alias references.
	RefTypeAlias RefType = "alias"
	// RefTypeIPSet lists only IP set references.
	RefTypeIPSet RefType = "ipset"
)

// Ref describes an alias or IP set that may be referenced from a rule's
// Source/Dest fields, returned by every scope's Client.Refs.
type Ref struct {
	// Type is "alias" or "ipset".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Name is the alias or IP set name.
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Ref is the value to use in a rule's Source/Dest field to reference
	// this alias or IP set (e.g. "+ipset-name" for an IP set).
	Ref string `json:"ref,omitempty" url:"ref,omitempty"`
	// Scope is the reference's scope, e.g. "dc" for a cluster-wide
	// alias/IP set, "guest" for a per-guest one.
	Scope string `json:"scope,omitempty" url:"scope,omitempty"`
	// Comment is the alias or IP set description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
}

// GuestOptions describes a per-guest firewall configuration exposed by
// GET/PUT .../firewall/options, identical for QEMU and LXC guests (see
// this file's doc comment). A single type serves both reads and writes:
// every field is optional, Proxmox only includes a key in the GET
// response once it has been explicitly set, and PUT only changes fields
// that are non-zero (via the `url` tag) — to explicitly reset a field to
// its default, list it in Delete instead of trying to send a zero value.
type GuestOptions struct {
	// Enable is 1 if the firewall is enabled for this guest.
	Enable int `json:"enable,omitempty" url:"enable,omitempty"`
	// MacFilter enables/disables the MAC address filter. Proxmox
	// defaults this to true.
	MacFilter bool `json:"macfilter,omitempty" url:"macfilter,omitempty"`
	// DHCP allows DHCP traffic.
	DHCP bool `json:"dhcp,omitempty" url:"dhcp,omitempty"`
	// NDP allows NDP (Neighbor Discovery Protocol). Proxmox defaults
	// this to true.
	NDP bool `json:"ndp,omitempty" url:"ndp,omitempty"`
	// RAdv allows the guest to send Router Advertisements.
	RAdv bool `json:"radv,omitempty" url:"radv,omitempty"`
	// IPFilter enables default IP filters, equivalent to adding an
	// empty ipfilter-net<id> IP set to every interface.
	IPFilter bool `json:"ipfilter,omitempty" url:"ipfilter,omitempty"`
	// PolicyIn is the guest's default input policy: "ACCEPT",
	// "REJECT" or "DROP".
	PolicyIn Policy `json:"policy_in,omitempty" url:"policy_in,omitempty"`
	// PolicyOut is the guest's default output policy: "ACCEPT",
	// "REJECT" or "DROP".
	PolicyOut Policy `json:"policy_out,omitempty" url:"policy_out,omitempty"`
	// LogLevelIn is the log level for incoming traffic.
	LogLevelIn RuleLogLevel `json:"log_level_in,omitempty" url:"log_level_in,omitempty"`
	// LogLevelOut is the log level for outgoing traffic.
	LogLevelOut RuleLogLevel `json:"log_level_out,omitempty" url:"log_level_out,omitempty"`
	// Delete lists properties to reset to their default value.
	// Write-only (PUT); Proxmox never returns it from GET — tagged
	// "writeonly" so Decode skips it even if a future GET response
	// happened to echo a "delete" key back.
	Delete []string `json:"-" url:"delete,omitempty,writeonly"`
	// Digest is the configuration digest, usable on a subsequent
	// Update to guard against concurrent changes.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// Encode converts the options to form parameters.
func (o *GuestOptions) Encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("firewall: options are required")
	}

	return params.Encode(o)
}

// LogEntry is a single firewall log line, as returned by a per-guest
// Client.Log.
type LogEntry struct {
	// N is the line number.
	N int64 `json:"n,omitempty" url:"n,omitempty"`
	// T is the line text.
	T string `json:"t,omitempty" url:"t,omitempty"`
}

// LogOptions filters the log lines returned by a per-guest Client.Log. A
// nil *LogOptions (or the zero value) requests Proxmox's default window.
type LogOptions struct {
	// Start is the first line number to return.
	Start int `url:"start,omitempty"`
	// Limit caps the number of lines returned.
	Limit int `url:"limit,omitempty"`
	// Since restricts the result to entries at or after this unix
	// timestamp.
	Since int64 `url:"since,omitempty"`
	// Until restricts the result to entries at or before this unix
	// timestamp.
	Until int64 `url:"until,omitempty"`
}
