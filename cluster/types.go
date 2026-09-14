package cluster

import (
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Status contains the cluster status returned by GET /cluster/status.
//
// The response mixes two kinds of entries in a single list: the overall
// cluster info (type "cluster") and one entry per node (type "node").
type Status struct {
	// ID is the cluster name (only set on the "cluster" entry).
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Name is the cluster name (only set on the "cluster" entry).
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Quorate indicates whether the cluster has quorum
	// (only set on the "cluster" entry).
	Quorate int `json:"quorate,omitempty" url:"quorate,omitempty"`
	// Version is the cluster configuration version
	// (only set on the "cluster" entry).
	Version int `json:"version,omitempty" url:"version,omitempty"`

	// Nodes is the list of cluster member nodes (type "node").
	Nodes []NodeStatus `json:"nodes,omitempty" url:"nodes,omitempty"`
}

// NodeStatus describes a single cluster member node.
type NodeStatus struct {
	// ID is the node ID, e.g. "node/pve1".
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Type is always "node".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Level is the node's cluster membership level (0 or 1).
	Level string `json:"level,omitempty" url:"level,omitempty"`

	// NodeID is the numeric cluster node identifier.
	NodeID int `json:"nodeid,omitempty" url:"nodeid,omitempty"`
	// Name is the node hostname, e.g. "pve1".
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// IP is the node's cluster communication IP address.
	IP string `json:"ip,omitempty" url:"ip,omitempty"`
	// Online is 1 if the node is online, 0 otherwise.
	Online int `json:"online,omitempty" url:"online,omitempty"`
	// Local is 1 if this node is the one serving the request.
	Local int `json:"local,omitempty" url:"local,omitempty"`
}

// Resource describes a single entry returned by GET /cluster/resources.
// The field set varies by Type: only the fields relevant to that type are
// populated.
type Resource struct {
	// ID is the full resource path, e.g. "node/pve1", "storage/local",
	// "qemu/100".
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Type is the resource kind: "node", "storage", "qemu", "lxc", "sdn"
	// or "pool".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Node is the node the resource belongs to or runs on.
	Node string `json:"node,omitempty" url:"node,omitempty"`
	// Storage is the storage identifier (storage entries only).
	Storage string `json:"storage,omitempty" url:"storage,omitempty"`
	// SDN is the SDN object identifier (sdn entries only).
	SDN string `json:"sdn,omitempty" url:"sdn,omitempty"`
	// Pool is the pool the guest belongs to (vm entries only).
	Pool string `json:"pool,omitempty" url:"pool,omitempty"`
	// VMID is the numeric guest ID (vm entries only).
	VMID int `json:"vmid,omitempty" url:"vmid,omitempty"`
	// Name is the guest, storage, or node name.
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Status is the resource status, e.g. "running", "available", "online".
	Status string `json:"status,omitempty" url:"status,omitempty"`
	// Template is 1 if the guest is a template (vm entries only).
	Template int `json:"template,omitempty" url:"template,omitempty"`
	// Tags is the comma-separated tag list (vm entries only).
	Tags string `json:"tags,omitempty" url:"tags,omitempty"`
	// Uptime is the uptime in seconds.
	Uptime int `json:"uptime,omitempty" url:"uptime,omitempty"`
	// Level is the node's cluster support level (node entries only).
	Level string `json:"level,omitempty" url:"level,omitempty"`
	// IP is the node's cluster communication IP address (node entries only).
	IP string `json:"ip,omitempty" url:"ip,omitempty"`
	// PluginType is the storage plugin type (storage entries only).
	PluginType string `json:"plugintype,omitempty" url:"plugintype,omitempty"`
	// Shared is 1 if the storage is shared (storage entries only).
	Shared int `json:"shared,omitempty" url:"shared,omitempty"`
	// StorageContent is the comma-separated content types of the storage
	// (storage entries only).
	StorageContent string `json:"content,omitempty" url:"content,omitempty"`
	// MaxCPU is the number of configured CPUs (vm/node entries).
	MaxCPU int `json:"maxcpu,omitempty" url:"maxcpu,omitempty"`
	// CPU is the CPU usage fraction (vm/node entries).
	CPU float64 `json:"cpu,omitempty" url:"cpu,omitempty"`
	// MaxMem is the maximum memory in bytes (vm/node entries).
	MaxMem int64 `json:"maxmem,omitempty" url:"maxmem,omitempty"`
	// Mem is the used memory in bytes (vm/node entries).
	Mem int64 `json:"mem,omitempty" url:"mem,omitempty"`
	// MaxDisk is the maximum/total disk space in bytes
	// (vm/node/storage entries).
	MaxDisk int64 `json:"maxdisk,omitempty" url:"maxdisk,omitempty"`
	// Disk is the used disk space in bytes (vm/node/storage entries).
	Disk int64 `json:"disk,omitempty" url:"disk,omitempty"`
	// DiskRead is the total bytes read from disk (vm entries only).
	DiskRead int64 `json:"diskread,omitempty" url:"diskread,omitempty"`
	// DiskWrite is the total bytes written to disk (vm entries only).
	DiskWrite int64 `json:"diskwrite,omitempty" url:"diskwrite,omitempty"`
	// NetIn is the network traffic in bytes received (vm entries only).
	NetIn int64 `json:"netin,omitempty" url:"netin,omitempty"`
	// NetOut is the network traffic in bytes sent (vm entries only).
	NetOut int64 `json:"netout,omitempty" url:"netout,omitempty"`
	// HAState is the high-availability state (vm entries only).
	HAState string `json:"hastate,omitempty" url:"hastate,omitempty"`
	// Lock is the guest lock state (vm entries only).
	Lock string `json:"lock,omitempty" url:"lock,omitempty"`
}

// Options describes the cluster-wide configuration exposed by
// GET/PUT /cluster/options. Every field is optional: Proxmox only includes
// a key in the GET response once it has been explicitly set, and PUT only
// changes the fields that are non-zero (via the `url` tag) — to explicitly
// reset a field to its default, list it in Delete instead of trying to
// send a zero value.
//
// Several fields (Bwlimit, Migration, HA, CRS, Notify, NextID, TagStyle,
// U2F, Webauthn) are Proxmox "property strings" (e.g.
// "type=secure,network=10.0.0.0/24") rather than nested JSON objects, so
// they are kept as opaque strings, matching how similar property strings
// (e.g. storage.Storage.PruneBackups) are handled elsewhere.
type Options struct {
	// Description is a cluster-wide description/comment (Datacenter
	// "Notes" field in the UI).
	Description string `json:"description,omitempty" url:"description,omitempty"`
	// EmailFrom is the sender address used for outbound notifications.
	EmailFrom string `json:"email_from,omitempty" url:"email_from,omitempty"`
	// Keyboard is the default keyboard layout for VNC/console access.
	Keyboard string `json:"keyboard,omitempty" url:"keyboard,omitempty"`
	// Language is the default UI language.
	Language string `json:"language,omitempty" url:"language,omitempty"`
	// Console is the default console viewer: "applet", "vv", "html5" or
	// "xtermjs".
	Console string `json:"console,omitempty" url:"console,omitempty"`
	// HTTPProxy is the proxy used for outbound HTTP requests, e.g. when
	// downloading appliance images.
	HTTPProxy string `json:"http_proxy,omitempty" url:"http_proxy,omitempty"`
	// MacPrefix is the OUI prefix used when generating guest MAC
	// addresses.
	MacPrefix string `json:"mac_prefix,omitempty" url:"mac_prefix,omitempty"`
	// MaxWorkers is the maximal number of worker processes per node.
	MaxWorkers int `json:"max_workers,omitempty" url:"max_workers,omitempty"`
	// Bwlimit is the property-string of default I/O bandwidth limits,
	// e.g. "clone=10240,default=0".
	Bwlimit string `json:"bwlimit,omitempty" url:"bwlimit,omitempty"`
	// Migration is the property-string controlling live migration, e.g.
	// "type=secure,network=10.0.0.0/24".
	Migration string `json:"migration,omitempty" url:"migration,omitempty"`
	// HA is the property-string configuring cluster-wide HA behavior,
	// e.g. "shutdown_policy=freeze".
	HA string `json:"ha,omitempty" url:"ha,omitempty"`
	// CRS is the property-string configuring the cluster resource
	// scheduler.
	CRS string `json:"crs,omitempty" url:"crs,omitempty"`
	// Notify is the property-string of default notification targets.
	Notify string `json:"notify,omitempty" url:"notify,omitempty"`
	// NextID is the property-string constraining VMID auto-allocation,
	// e.g. "lower=100,upper=999999999".
	NextID string `json:"next-id,omitempty" url:"next-id,omitempty"`
	// RegisteredTags is the comma-separated list of tags that are
	// managed/registered cluster-wide.
	RegisteredTags string `json:"registered-tags,omitempty" url:"registered-tags,omitempty"`
	// TagStyle is the property-string configuring tag appearance and
	// ordering in the UI.
	TagStyle string `json:"tag-style,omitempty" url:"tag-style,omitempty"`
	// U2F is the property-string of U2F authentication settings
	// (superseded by Webauthn).
	U2F string `json:"u2f,omitempty" url:"u2f,omitempty"`
	// Webauthn is the property-string of WebAuthn authentication
	// settings.
	Webauthn string `json:"webauthn,omitempty" url:"webauthn,omitempty"`

	// Delete lists properties to reset to their default value. Write-only:
	// Proxmox never returns it, so Get leaves it empty.
	Delete []string `json:"-" url:"delete"`
}

// encode converts the options to form parameters for PUT /cluster/options.
func (o *Options) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("cluster: options are required")
	}

	return params.Encode(o)
}

// HAGroup describes a high-availability group as returned by
// GET /cluster/ha/groups and GET /cluster/ha/groups/{group}.
type HAGroup struct {
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
	// Digest is the configuration digest, usable with
	// HAGroupOptions.Digest to guard concurrent updates.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// HAGroupOptions holds the write parameters shared by POST /cluster/ha/groups
// (Create) and PUT /cluster/ha/groups/{group} (Update).
//
// ID is required by Create; Update ignores it (the group id is already part
// of the URL). Pointer fields are always sent when non-nil (even when
// zero/empty), which is how Update expresses "clear this field" — Create
// simply sends whatever is set. Delete and Digest are meaningful to Update
// only.
type HAGroupOptions struct {
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
func (o *HAGroupOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("cluster: ha group options are required")
	}

	return params.Encode(o)
}

// HARuleType filters GET /cluster/ha/rules and selects the schema POST/PUT
// validates the rest of an HA rule's fields against.
type HARuleType string

const (
	// HARuleTypeNodeAffinity pins HA resources to (or away from) specific
	// nodes.
	HARuleTypeNodeAffinity HARuleType = "node-affinity"
	// HARuleTypeResourceAffinity keeps HA resources together on, or apart
	// from, each other.
	HARuleTypeResourceAffinity HARuleType = "resource-affinity"
)

// HARuleAffinity controls whether a resource-affinity rule keeps its
// resources on the same node ("positive") or on separate nodes
// ("negative").
type HARuleAffinity string

const (
	// HARuleAffinityPositive keeps the rule's resources on the same node.
	HARuleAffinityPositive HARuleAffinity = "positive"
	// HARuleAffinityNegative keeps the rule's resources on separate nodes.
	HARuleAffinityNegative HARuleAffinity = "negative"
)

// HARule describes a high-availability rule as returned by
// GET /cluster/ha/rules and GET /cluster/ha/rules/{rule}.
type HARule struct {
	// Rule is the HA rule identifier, e.g. "my-rule".
	Rule string `json:"rule,omitempty" url:"rule,omitempty"`
	// Type is the rule kind: "node-affinity" or "resource-affinity".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Resources is the list of HA resource IDs the rule applies to, e.g.
	// "vm:100", "ct:101".
	Resources []string `json:"resources,omitempty" url:"resources,omitempty"`
	// Nodes is the property-string list of member nodes with optional
	// priority (node-affinity rules only), e.g. "node1:2,node2:1". Kept as
	// an opaque string, matching HAGroup.Nodes.
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
	// Digest is the configuration digest, usable with HARuleOptions.Digest
	// to guard concurrent updates.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// HARuleOptions holds the write parameters shared by POST /cluster/ha/rules
// (Create) and PUT /cluster/ha/rules/{rule} (Update).
//
// ID is required by Create; Update ignores it (the rule id is already part
// of the URL). Type and Resources are required by Create; Proxmox also
// requires Type to be resent on every Update. Pointer fields are always
// sent when non-nil (even when zero/empty), which is how Update expresses
// "clear this field" — Create simply sends whatever is set. Delete and
// Digest are meaningful to Update only.
type HARuleOptions struct {
	// ID is the HA rule identifier, e.g. "my-rule". Required by Create;
	// not settable on Update.
	ID string `url:"rule"`
	// Type is the rule kind: "node-affinity" or "resource-affinity".
	// Required by both Create and Update.
	Type HARuleType `url:"type"`
	// Resources is the list of HA resource IDs the rule applies to, e.g.
	// "vm:100", "ct:101". Required by Create; on Update replaces the
	// current list.
	Resources []string `url:"resources"`
	// Nodes is the property-string list of member nodes with optional
	// priority (node-affinity rules only), e.g. "node1:2,node2:1".
	Nodes *string `url:"nodes"`
	// Affinity is "positive" (keep together) or "negative" (keep apart)
	// (resource-affinity rules only).
	Affinity *HARuleAffinity `url:"affinity"`
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
func (o *HARuleOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("cluster: ha rule options are required")
	}

	return params.Encode(o)
}
