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
