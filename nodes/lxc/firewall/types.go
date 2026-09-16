package firewall

import (
	"github.com/sergelogvinov/go-proxmox-rest/types"
)

// RuleType, RuleLogLevel, Policy, Rule, RuleOptions, Alias, AliasOptions,
// IPSet, IPSetOptions, IPSetEntry, IPSetEntryOptions, RefType, Ref,
// Options, LogEntry and LogOptions are all identical between the LXC and
// QEMU per-guest firewalls (and, except for Options/LogEntry/LogOptions,
// the cluster-wide firewall too): Proxmox backs every one of them with
// the same PVE::Firewall::Rules/Aliases/IPSet modules and, for the guest
// scope specifically, the literal same PVE::API2::Firewall::VMBase
// handler for both VMs and CTs. They live in the shared types package
// (see its doc comment) and are re-exported here as aliases so call
// sites read as firewall.Rule, firewall.Options, ... .
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
	Options           = types.GuestOptions
	LogEntry          = types.LogEntry
	LogOptions        = types.LogOptions
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
