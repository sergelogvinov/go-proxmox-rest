package firewall

import (
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
	"github.com/sergelogvinov/go-proxmox-rest/types"
)

// RuleType, RuleLogLevel, Policy, Rule, RuleOptions, Alias, AliasOptions,
// IPSet, IPSetOptions, IPSetEntry, IPSetEntryOptions, RefType and Ref are
// identical across every firewall scope Proxmox exposes (cluster-wide,
// per-guest, ...): PVE::Firewall::Rules/Aliases/IPSet back all of them
// with the same wire format. They live in the shared types package (see
// its doc comment) and are re-exported here as aliases so existing call
// sites (firewall.Rule, firewall.RuleOptions, ...) keep working unchanged.
type (
	RuleType          = types.RuleType
	RuleLogLevel      = types.RuleLogLevel
	Policy            = types.Policy
	Rule              = types.Rule
	RuleOptions       = types.RuleOptions
	Alias             = types.Alias
	AliasOptions      = types.AliasOptions
	IPSet             = types.IPSet
	IPSetOptions      = types.IPSetOptions
	IPSetEntry        = types.IPSetEntry
	IPSetEntryOptions = types.IPSetEntryOptions
	RefType           = types.RefType
	Ref               = types.Ref
)

const (
	RuleTypeIn      = types.RuleTypeIn
	RuleTypeOut     = types.RuleTypeOut
	RuleTypeForward = types.RuleTypeForward
	RuleTypeGroup   = types.RuleTypeGroup

	RuleLogEmerg   = types.RuleLogEmerg
	RuleLogAlert   = types.RuleLogAlert
	RuleLogCrit    = types.RuleLogCrit
	RuleLogErr     = types.RuleLogErr
	RuleLogWarning = types.RuleLogWarning
	RuleLogNotice  = types.RuleLogNotice
	RuleLogInfo    = types.RuleLogInfo
	RuleLogDebug   = types.RuleLogDebug
	RuleLogNolog   = types.RuleLogNolog

	PolicyAccept = types.PolicyAccept
	PolicyReject = types.PolicyReject
	PolicyDrop   = types.PolicyDrop

	RefTypeAlias = types.RefTypeAlias
	RefTypeIPSet = types.RefTypeIPSet
)

// Group describes a security group as returned by
// GET /cluster/firewall/groups. It carries only the group's own metadata;
// its rules are accessed via Client.Groups().Rules(name). Security groups
// are a cluster-only concept (there is no per-guest equivalent), so unlike
// the aliased types above, Group has no shared definition.
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

// Options describes the cluster-wide firewall configuration exposed by
// GET/PUT /cluster/firewall/options. A single type serves both reads and
// writes, matching cluster.Options: every field is optional, Proxmox only
// includes a key in the GET response once it has been explicitly set, and
// PUT only changes fields that are non-zero (via the `url` tag) — to
// explicitly reset a field to its default, list it in Delete instead of
// trying to send a zero value.
//
// This is genuinely different from nodes/qemu/firewall's Options (guest
// firewalls have their own option set: enable, macfilter, dhcp, ndp,
// radv, ipfilter, policy_in/out, log_level_in/out, with no ebtables/
// policy_forward/log_ratelimit), so unlike the aliased types above it is
// not shared.
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
