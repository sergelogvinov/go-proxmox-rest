package cluster

// Status contains the cluster status returned by GET /cluster/status.
//
// The response mixes two kinds of entries in a single list: the overall
// cluster info (type "cluster") and one entry per node (type "node").
type Status struct {
	// ID is the cluster name (only set on the "cluster" entry).
	ID string `json:"id,omitempty"`
	// Name is the cluster name (only set on the "cluster" entry).
	Name string `json:"name,omitempty"`
	// Quorate indicates whether the cluster has quorum
	// (only set on the "cluster" entry).
	Quorate int `json:"quorate,omitempty"`
	// Version is the cluster configuration version
	// (only set on the "cluster" entry).
	Version int `json:"version,omitempty"`

	// Nodes is the list of cluster member nodes (type "node").
	Nodes []NodeStatus `json:"nodes,omitempty"`
}

// NodeStatus describes a single cluster member node.
type NodeStatus struct {
	// ID is the node ID, e.g. "node/pve1".
	ID string `json:"id,omitempty"`
	// Type is always "node".
	Type string `json:"type,omitempty"`
	// Level is the node's cluster membership level (0 or 1).
	Level string `json:"level,omitempty"`

	// NodeID is the numeric cluster node identifier.
	NodeID int `json:"nodeid,omitempty"`
	// Name is the node hostname, e.g. "pve1".
	Name string `json:"name,omitempty"`
	// IP is the node's cluster communication IP address.
	IP string `json:"ip,omitempty"`
	// Online is 1 if the node is online, 0 otherwise.
	Online int `json:"online,omitempty"`
	// Local is 1 if this node is the one serving the request.
	Local int `json:"local,omitempty"`
}

// Resource describes a single entry returned by GET /cluster/resources.
// The field set varies by Type: only the fields relevant to that type are
// populated.
type Resource struct {
	// ID is the full resource path, e.g. "node/pve1", "storage/local",
	// "qemu/100".
	ID string `json:"id,omitempty"`
	// Type is the resource kind: "node", "storage", "qemu", "lxc", "sdn"
	// or "pool".
	Type string `json:"type,omitempty"`
	// Node is the node the resource belongs to or runs on.
	Node string `json:"node,omitempty"`
	// Storage is the storage identifier (storage entries only).
	Storage string `json:"storage,omitempty"`
	// SDN is the SDN object identifier (sdn entries only).
	SDN string `json:"sdn,omitempty"`
	// Pool is the pool the guest belongs to (vm entries only).
	Pool string `json:"pool,omitempty"`
	// VMID is the numeric guest ID (vm entries only).
	VMID int `json:"vmid,omitempty"`
	// Name is the guest, storage, or node name.
	Name string `json:"name,omitempty"`
	// Status is the resource status, e.g. "running", "available", "online".
	Status string `json:"status,omitempty"`
	// Template is 1 if the guest is a template (vm entries only).
	Template int `json:"template,omitempty"`
	// Tags is the comma-separated tag list (vm entries only).
	Tags string `json:"tags,omitempty"`
	// Uptime is the uptime in seconds.
	Uptime int `json:"uptime,omitempty"`
	// Level is the node's cluster support level (node entries only).
	Level string `json:"level,omitempty"`
	// IP is the node's cluster communication IP address (node entries only).
	IP string `json:"ip,omitempty"`
	// PluginType is the storage plugin type (storage entries only).
	PluginType string `json:"plugintype,omitempty"`
	// Shared is 1 if the storage is shared (storage entries only).
	Shared int `json:"shared,omitempty"`
	// StorageContent is the comma-separated content types of the storage
	// (storage entries only).
	StorageContent string `json:"content,omitempty"`
	// MaxCPU is the number of configured CPUs (vm/node entries).
	MaxCPU int `json:"maxcpu,omitempty"`
	// CPU is the CPU usage fraction (vm/node entries).
	CPU float64 `json:"cpu,omitempty"`
	// MaxMem is the maximum memory in bytes (vm/node entries).
	MaxMem int64 `json:"maxmem,omitempty"`
	// Mem is the used memory in bytes (vm/node entries).
	Mem int64 `json:"mem,omitempty"`
	// MaxDisk is the maximum/total disk space in bytes
	// (vm/node/storage entries).
	MaxDisk int64 `json:"maxdisk,omitempty"`
	// Disk is the used disk space in bytes (vm/node/storage entries).
	Disk int64 `json:"disk,omitempty"`
	// DiskRead is the total bytes read from disk (vm entries only).
	DiskRead int64 `json:"diskread,omitempty"`
	// DiskWrite is the total bytes written to disk (vm entries only).
	DiskWrite int64 `json:"diskwrite,omitempty"`
	// NetIn is the network traffic in bytes received (vm entries only).
	NetIn int64 `json:"netin,omitempty"`
	// NetOut is the network traffic in bytes sent (vm entries only).
	NetOut int64 `json:"netout,omitempty"`
	// HAState is the high-availability state (vm entries only).
	HAState string `json:"hastate,omitempty"`
	// Lock is the guest lock state (vm entries only).
	Lock string `json:"lock,omitempty"`
}
