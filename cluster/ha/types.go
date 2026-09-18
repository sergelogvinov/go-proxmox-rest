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

// ResourceMode controls how HA-managed resources are treated while the
// cluster-wide fencing stack is disarmed. It appears both as
// StatusEntry.ResourceMode (read, type "fencing" entries) and as the
// required parameter to statusResource.Update when disarming.
type ResourceMode string

const (
	// ResourceModeFreeze leaves resources' current state untouched: new
	// commands and state changes are not applied while disarmed.
	ResourceModeFreeze ResourceMode = "freeze"
	// ResourceModeIgnore drops resources from HA tracking entirely while
	// disarmed, so they can be managed as if they were not HA managed.
	ResourceModeIgnore ResourceMode = "ignore"
)

// EntryType is the kind of a StatusEntry returned by
// GET /cluster/ha/status/current.
type EntryType string

const (
	// EntryTypeQuorum reports whether the cluster currently has quorum.
	EntryTypeQuorum EntryType = "quorum"
	// EntryTypeMaster identifies the node currently holding the CRM
	// (master) lock.
	EntryTypeMaster EntryType = "master"
	// EntryTypeLRM reports a single node's Local Resource Manager state.
	EntryTypeLRM EntryType = "lrm"
	// EntryTypeService reports a single HA-managed resource's state.
	EntryTypeService EntryType = "service"
	// EntryTypeFencing reports the cluster-wide HA fencing arm state.
	EntryTypeFencing EntryType = "fencing"
)

// StatusEntry describes a single entry returned by
// GET /cluster/ha/status/current. The list mixes five unrelated kinds of
// entry (see Type); only the fields relevant to that Type are populated.
type StatusEntry struct {
	// ID identifies the entry, e.g. "quorum", "master", "lrm:<node>" or
	// "service:<sid>".
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Node is the node associated with this entry.
	Node string `json:"node,omitempty" url:"node,omitempty"`
	// Status is the entry's status; its meaning depends on Type.
	Status string `json:"status,omitempty" url:"status,omitempty"`
	// Type is the kind of status entry.
	Type EntryType `json:"type,omitempty" url:"type,omitempty"`

	// Quorate is 1 if the cluster currently has quorum (type "quorum"
	// only).
	Quorate int `json:"quorate,omitempty" url:"quorate,omitempty"`
	// Timestamp is when the status information was recorded (types "lrm"
	// and "master" only).
	Timestamp int64 `json:"timestamp,omitempty" url:"timestamp,omitempty"`

	// CRMState is the service state as seen by the CRM (type "service"
	// only).
	CRMState string `json:"crm_state,omitempty" url:"crm_state,omitempty"`
	// State is the verbose service state, e.g. "started", "fence",
	// "recovery", "migrate" (type "service" only).
	State string `json:"state,omitempty" url:"state,omitempty"`
	// RequestState is the requested service state (type "service" only).
	RequestState string `json:"request_state,omitempty" url:"request_state,omitempty"`
	// SID is the HA resource/service ID, e.g. "vm:100" (type "service"
	// only).
	SID string `json:"sid,omitempty" url:"sid,omitempty"`
	// Failback is 1 if the resource automatically migrates back to the
	// highest-priority node once it rejoins (type "service" only,
	// defaults to enabled).
	Failback int `json:"failback,omitempty" url:"failback,omitempty"`
	// AutoRebalance is 1 if the resource may be migrated during automatic
	// rebalancing (type "service" only, defaults to enabled).
	AutoRebalance int `json:"auto-rebalance,omitempty" url:"auto-rebalance,omitempty"`
	// MaxRelocate is the resource's relocation attempt limit (type
	// "service" only).
	MaxRelocate int `json:"max_relocate,omitempty" url:"max_relocate,omitempty"`
	// MaxRestart is the resource's restart attempt limit (type "service"
	// only).
	MaxRestart int `json:"max_restart,omitempty" url:"max_restart,omitempty"`

	// ArmedState is whether HA fencing is "armed", "standby",
	// "disarming" or "disarmed" (type "fencing" only).
	ArmedState string `json:"armed-state,omitempty" url:"armed-state,omitempty"`
	// ResourceMode is how resources are handled while disarmed: "freeze"
	// or "ignore" (type "fencing" only).
	ResourceMode string `json:"resource_mode,omitempty" url:"resource_mode,omitempty"`
}

// ManagerStatus is the response of GET /cluster/ha/status/manager_status:
// the elected CRM master's persisted status, merged with live quorum and
// per-node LRM info. Proxmox does not publish a formal schema for this
// endpoint (its API declares the response as a bare "object"), so this
// mirrors the on-disk /etc/pve/ha/manager_status structure as maintained by
// PVE::HA::Manager; unknown fields are ignored, so the type can grow to
// cover more of it later without breaking callers.
type ManagerStatus struct {
	// Manager is the CRM master's own persisted status, nil if no master
	// is currently elected.
	Manager *ManagerState `json:"manager_status,omitempty" url:"manager_status,omitempty"`
	// Quorum is the live quorum state as seen by the responding node.
	Quorum *ManagerQuorum `json:"quorum,omitempty" url:"quorum,omitempty"`
	// LRM is the live per-node Local Resource Manager state, keyed by
	// node name.
	LRM map[string]LRMState `json:"lrm_status,omitempty" url:"lrm_status,omitempty"`
}

// ManagerState is the CRM master's own persisted status, the
// "manager_status" key of ManagerStatus.
type ManagerState struct {
	// MasterNode is the node currently holding the CRM lock.
	MasterNode string `json:"master_node,omitempty" url:"master_node,omitempty"`
	// Timestamp is when the master last wrote this status.
	Timestamp int64 `json:"timestamp,omitempty" url:"timestamp,omitempty"`
	// NodeStatus is the per-node HA availability as seen by the master,
	// keyed by node name (e.g. "online", "unknown", "fence").
	NodeStatus map[string]string `json:"node_status,omitempty" url:"node_status,omitempty"`
	// ServiceStatus is the per-service HA state, keyed by HA resource ID
	// (e.g. "vm:100").
	ServiceStatus map[string]ServiceState `json:"service_status,omitempty" url:"service_status,omitempty"`
}

// ServiceState is a single HA-managed resource's state within
// ManagerState.ServiceStatus.
type ServiceState struct {
	// State is the service's current state, e.g. "started", "stopped",
	// "fence", "recovery", "migrate".
	State string `json:"state,omitempty" url:"state,omitempty"`
	// Node is the node the service currently runs on (or is assigned
	// to).
	Node string `json:"node,omitempty" url:"node,omitempty"`
	// UID is the CRM-internal unique identifier for this service's
	// current state transition.
	UID string `json:"uid,omitempty" url:"uid,omitempty"`
}

// ManagerQuorum is the live quorum state within ManagerStatus.
type ManagerQuorum struct {
	// Node is the node the quorum state was observed on.
	Node string `json:"node,omitempty" url:"node,omitempty"`
	// Quorate is 1 if the cluster currently has quorum.
	Quorate int `json:"quorate,omitempty" url:"quorate,omitempty"`
}

// LRMState is a single node's Local Resource Manager state within
// ManagerStatus.LRM.
type LRMState struct {
	// Mode is the LRM's operating mode: "wait_for_agent_lock" (idle,
	// waiting for the exclusive lock), "active" (holds the lock and has
	// services configured), or "lost_agent_lock" (lost the lock,
	// typically after losing quorum).
	Mode string `json:"mode,omitempty" url:"mode,omitempty"`
	// State mirrors Mode for older Proxmox versions that report it under
	// this key instead.
	State string `json:"state,omitempty" url:"state,omitempty"`
	// Timestamp is when the LRM last wrote this status.
	Timestamp int64 `json:"timestamp,omitempty" url:"timestamp,omitempty"`
}
