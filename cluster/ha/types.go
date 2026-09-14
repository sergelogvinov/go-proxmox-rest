package ha

import (
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Group describes a high-availability group as returned by
// GET /cluster/ha/groups and GET /cluster/ha/groups/{group}.
type Group struct {
	// Group is the HA group identifier, e.g. "my-group".
	Group string `json:"group,omitempty" url:"group,omitempty"`
	// Type is always "group".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Nodes is the property-string list of member nodes with optional
	// priority, e.g. "node1:2,node2:1" (higher number wins). Kept as an
	// opaque string, matching how similar property strings (e.g.
	// cluster.Options.Migration) are handled elsewhere.
	Nodes string `json:"nodes,omitempty" url:"nodes,omitempty"`
	// Comment is the group description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
	// Nofailback is 1 if automatic failback to a higher-priority node is
	// disabled.
	Nofailback int `json:"nofailback,omitempty" url:"nofailback,omitempty"`
	// Restricted is 1 if resources bound to this group may only run on
	// the group's nodes.
	Restricted int `json:"restricted,omitempty" url:"restricted,omitempty"`
	// Digest is the configuration digest, usable with GroupOptions.Digest
	// to guard concurrent updates.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// GroupOptions holds the write parameters shared by POST /cluster/ha/groups
// (Create) and PUT /cluster/ha/groups/{group} (Update).
//
// ID is required by Create; Update ignores it (the group id is already part
// of the URL). Pointer fields are always sent when non-nil (even when
// zero/empty), which is how Update expresses "clear this field" — Create
// simply sends whatever is set. Delete and Digest are meaningful to Update
// only.
type GroupOptions struct {
	// ID is the HA group identifier, e.g. "my-group". Required by Create;
	// not settable on Update.
	ID string `url:"group"`
	// Nodes is the property-string list of member nodes with optional
	// priority, e.g. "node1:2,node2:1". Required by Create.
	Nodes *string `url:"nodes"`
	// Comment is the group description.
	Comment *string `url:"comment"`
	// Nofailback disables automatic failback to a higher-priority node
	// once it rejoins the group.
	Nofailback *bool `url:"nofailback"`
	// Restricted restricts resources bound to this group to only run on
	// the group's nodes.
	Restricted *bool `url:"restricted"`
	// Delete lists properties to reset to their default value
	// (Update only).
	Delete []string `url:"delete"`
	// Digest prevents changes if the current configuration has changed in
	// between (value from GET /cluster/ha/groups/{group}). Update only.
	Digest string `url:"digest"`
}

// encode converts the options to form parameters.
func (o *GroupOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("ha: group options are required")
	}

	return params.Encode(o)
}

// RuleType filters GET /cluster/ha/rules and selects the schema POST/PUT
// validates the rest of an HA rule's fields against.
type RuleType string

const (
	// RuleTypeNodeAffinity pins HA resources to (or away from) specific
	// nodes.
	RuleTypeNodeAffinity RuleType = "node-affinity"
	// RuleTypeResourceAffinity keeps HA resources together on, or apart
	// from, each other.
	RuleTypeResourceAffinity RuleType = "resource-affinity"
)

// RuleAffinity controls whether a resource-affinity rule keeps its
// resources on the same node ("positive") or on separate nodes
// ("negative").
type RuleAffinity string

const (
	// RuleAffinityPositive keeps the rule's resources on the same node.
	RuleAffinityPositive RuleAffinity = "positive"
	// RuleAffinityNegative keeps the rule's resources on separate nodes.
	RuleAffinityNegative RuleAffinity = "negative"
)

// Rule describes a high-availability rule as returned by
// GET /cluster/ha/rules and GET /cluster/ha/rules/{rule}.
type Rule struct {
	// Rule is the HA rule identifier, e.g. "my-rule".
	Rule string `json:"rule,omitempty" url:"rule,omitempty"`
	// Type is the rule kind: "node-affinity" or "resource-affinity".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Resources is the list of HA resource IDs the rule applies to, e.g.
	// "vm:100", "ct:101".
	Resources []string `json:"resources,omitempty" url:"resources,omitempty"`
	// Nodes is the property-string list of member nodes with optional
	// priority (node-affinity rules only), e.g. "node1:2,node2:1". Kept as
	// an opaque string, matching Group.Nodes.
	Nodes string `json:"nodes,omitempty" url:"nodes,omitempty"`
	// Affinity is "positive" (keep together) or "negative" (keep apart)
	// (resource-affinity rules only).
	Affinity string `json:"affinity,omitempty" url:"affinity,omitempty"`
	// Strict is 1 if a node-affinity rule is strict (resources may only
	// run on the given nodes) rather than non-strict (preferred).
	Strict int `json:"strict,omitempty" url:"strict,omitempty"`
	// Disable is 1 if the rule is disabled.
	Disable int `json:"disable,omitempty" url:"disable,omitempty"`
	// Comment is the rule description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
	// Digest is the configuration digest, usable with RuleOptions.Digest
	// to guard concurrent updates.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// RuleOptions holds the write parameters shared by POST /cluster/ha/rules
// (Create) and PUT /cluster/ha/rules/{rule} (Update).
//
// ID is required by Create; Update ignores it (the rule id is already part
// of the URL). Type and Resources are required by Create; Proxmox also
// requires Type to be resent on every Update. Pointer fields are always
// sent when non-nil (even when zero/empty), which is how Update expresses
// "clear this field" — Create simply sends whatever is set. Delete and
// Digest are meaningful to Update only.
type RuleOptions struct {
	// ID is the HA rule identifier, e.g. "my-rule". Required by Create;
	// not settable on Update.
	ID string `url:"rule"`
	// Type is the rule kind: "node-affinity" or "resource-affinity".
	// Required by both Create and Update.
	Type RuleType `url:"type"`
	// Resources is the list of HA resource IDs the rule applies to, e.g.
	// "vm:100", "ct:101". Required by Create; on Update replaces the
	// current list.
	Resources []string `url:"resources"`
	// Nodes is the property-string list of member nodes with optional
	// priority (node-affinity rules only), e.g. "node1:2,node2:1".
	Nodes *string `url:"nodes"`
	// Affinity is "positive" (keep together) or "negative" (keep apart)
	// (resource-affinity rules only).
	Affinity *RuleAffinity `url:"affinity"`
	// Strict controls whether a node-affinity rule is strict (resources
	// may only run on the given nodes) or non-strict (preferred).
	Strict *bool `url:"strict"`
	// Disable disables the rule without deleting it.
	Disable *bool `url:"disable"`
	// Comment is the rule description.
	Comment *string `url:"comment"`
	// Delete lists properties to reset to their default value
	// (Update only).
	Delete []string `url:"delete"`
	// Digest prevents changes if the current configuration has changed in
	// between (value from GET /cluster/ha/rules/{rule}). Update only.
	Digest string `url:"digest"`
}

// encode converts the options to form parameters.
func (o *RuleOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("ha: rule options are required")
	}

	return params.Encode(o)
}
