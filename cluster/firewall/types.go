package firewall

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
	// default policy. Not valid for the forward policy.
	PolicyReject Policy = "REJECT"
	// PolicyDrop silently drops traffic that reaches the default policy.
	PolicyDrop Policy = "DROP"
)

// Rule describes a single firewall rule as returned by
// GET /cluster/firewall/rules, GET /cluster/firewall/rules/{pos},
// GET /cluster/firewall/groups/{group} and
// GET /cluster/firewall/groups/{group}/{pos}. The same shape is used for
// cluster-wide rules and security-group rules.
//
// Every field carries a `url` tag matching its wire name even though Rule
// is read-only and never passed to params.Encode: this package's decode
// path (internal/params.Decode, used by every GET) resolves each field's
// wire name from the `url` tag, not `json` — see decode.go's doc comment.
// The `json` tags are kept alongside for readability/documentation and are
// otherwise inert here (decode ignores them).
type Rule struct {
	// Pos is the rule's position within its rule list (cluster-wide rules
	// or a specific group's rules). It addresses the rule for
	// Get/Update/Delete.
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
	// IFace restricts the rule to a network interface (VM/CT net key
	// names, e.g. "net0", on nodes; physical interface names elsewhere).
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
// (POST /cluster/firewall/rules or POST /cluster/firewall/groups/{group})
// and rule updates (PUT .../rules/{pos} or PUT .../groups/{group}/{pos}).
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

// encode converts the options to form parameters.
func (o *RuleOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("firewall: rule options are required")
	}

	return params.Encode(o)
}

// Group describes a security group as returned by
// GET /cluster/firewall/groups. It carries only the group's own metadata;
// its rules are accessed via Client.Groups().Rules(name).
type Group struct {
	// Name is the security group identifier, e.g. "my-group".
	Name string `json:"group,omitempty" url:"group,omitempty"`
	// Comment is the group description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
	// Digest is the configuration digest, usable with GroupOptions.Digest
	// to guard concurrent updates.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// GroupOptions holds the write parameters for POST /cluster/firewall/groups.
// Proxmox has no dedicated update endpoint for a security group's own
// metadata: creation and rename/re-comment both go through this same POST,
// distinguished by the Rename field — see groupsResource.Update.
type GroupOptions struct {
	// Name is the security group's target name: the name to create
	// (Create), or the new name (Update — set equal to the current name
	// to update Comment without renaming).
	Name string `url:"group"`
	// Comment is the group description.
	Comment *string `url:"comment"`
	// Rename is, on Update only, the group's current name (set by
	// groupsResource.Update; callers do not set this directly).
	Rename string `url:"rename"`
	// Digest prevents changes if the current configuration has changed
	// in between. Update only.
	Digest string `url:"digest"`
}

// encode converts the options to form parameters.
func (o *GroupOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("firewall: group options are required")
	}

	return params.Encode(o)
}

// Alias describes an IP/network alias as returned by
// GET /cluster/firewall/aliases and GET /cluster/firewall/aliases/{name}.
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
// POST /cluster/firewall/aliases (Create) and
// PUT /cluster/firewall/aliases/{name} (Update).
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

// encode converts the options to form parameters.
func (o *AliasOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("firewall: alias options are required")
	}

	return params.Encode(o)
}

// IPSet describes an IP set as returned by GET /cluster/firewall/ipset. It
// carries only the set's own metadata; its members are accessed via
// Client.IPSet().Entries(name).
type IPSet struct {
	// Name is the IP set identifier, e.g. "my-ipset".
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Comment is the IP set description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
	// Digest is the configuration digest, usable with IPSetOptions.Digest
	// to guard concurrent updates.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// IPSetOptions holds the write parameters for POST /cluster/firewall/ipset.
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

// encode converts the options to form parameters.
func (o *IPSetOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("firewall: ipset options are required")
	}

	return params.Encode(o)
}

// IPSetEntry describes a single member of an IP set, as returned by
// GET /cluster/firewall/ipset/{name} and
// GET /cluster/firewall/ipset/{name}/{cidr}.
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
// POST /cluster/firewall/ipset/{name} (Create) and
// PUT /cluster/firewall/ipset/{name}/{cidr} (Update).
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

// encode converts the options to form parameters.
func (o *IPSetEntryOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("firewall: ipset entry options are required")
	}

	return params.Encode(o)
}

// Options describes the cluster-wide firewall configuration exposed by
// GET/PUT /cluster/firewall/options. A single type serves both reads and
// writes, matching cluster.Options: every field is optional, Proxmox only
// includes a key in the GET response once it has been explicitly set, and
// PUT only changes fields that are non-zero (via the `url` tag) — to
// explicitly reset a field to its default, list it in Delete instead of
// trying to send a zero value.
type Options struct {
	// Enable is 1 if the firewall is enabled cluster-wide.
	Enable int `json:"enable,omitempty" url:"enable,omitempty"`
	// Ebtables enables ebtables rules cluster-wide (defaults to true on
	// Proxmox). Because the zero value (false) is indistinguishable from
	// "unset" here, explicitly disabling ebtables requires listing
	// "ebtables" in Delete first or relying on Proxmox's default.
	Ebtables bool `json:"ebtables,omitempty" url:"ebtables,omitempty"`
	// PolicyIn is the default input policy: "ACCEPT", "REJECT" or "DROP".
	PolicyIn Policy `json:"policy_in,omitempty" url:"policy_in,omitempty"`
	// PolicyOut is the default output policy: "ACCEPT", "REJECT" or
	// "DROP".
	PolicyOut Policy `json:"policy_out,omitempty" url:"policy_out,omitempty"`
	// PolicyForward is the default forward policy: "ACCEPT" or "DROP"
	// (REJECT is not valid here).
	PolicyForward Policy `json:"policy_forward,omitempty" url:"policy_forward,omitempty"`
	// LogRatelimit is the property-string of log rate-limiting settings,
	// e.g. "enable=1,rate=1/second,burst=5". Kept as an opaque string,
	// matching how similar property strings (e.g. cluster.Options.HA)
	// are handled elsewhere.
	LogRatelimit string `json:"log_ratelimit,omitempty" url:"log_ratelimit,omitempty"`
	// Delete lists properties to reset to their default value. Write-only
	// (PUT); Proxmox never returns it from GET — tagged "writeonly" so
	// Decode skips it even if a future GET response happened to echo a
	// "delete" key back.
	Delete []string `json:"-" url:"delete,omitempty,writeonly"`
	// Digest is the configuration digest, usable on a subsequent Update
	// to guard against concurrent changes. Not documented in the Proxmox
	// API schema but included for parity with other singleton config
	// objects (e.g. cluster.Options.Digest).
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// encode converts the options to form parameters.
func (o *Options) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("firewall: options are required")
	}

	return params.Encode(o)
}

// RefType filters GET /cluster/firewall/refs.
type RefType string

const (
	// RefTypeAlias lists only alias references.
	RefTypeAlias RefType = "alias"
	// RefTypeIPSet lists only IP set references.
	RefTypeIPSet RefType = "ipset"
)

// Ref describes an alias or IP set that may be referenced from a rule's
// Source/Dest fields, as returned by GET /cluster/firewall/refs.
type Ref struct {
	// Type is "alias" or "ipset".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Name is the alias or IP set name.
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Ref is the value to use in a rule's Source/Dest field to reference
	// this alias or IP set (e.g. "+ipset-name" for an IP set).
	Ref string `json:"ref,omitempty" url:"ref,omitempty"`
	// Scope is the reference's scope, e.g. "dc" for a cluster-wide
	// alias/IP set.
	Scope string `json:"scope,omitempty" url:"scope,omitempty"`
	// Comment is the alias or IP set description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
}
