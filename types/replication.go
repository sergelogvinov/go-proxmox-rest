package types

// replication.go: the two small enums Proxmox reuses verbatim between the
// cluster-wide replication job config (cluster/replication) and the
// per-node replication status/log view (nodes/replication) — both back
// onto the same PVE::ReplicationConfig section type. Job (cluster/
// replication, a writable config entry) and JobStatus (nodes/
// replication, that same config merged with runtime state — NextSync/
// LastSync/LastTry/FailCount/Error/Duration/PID — and read-only, since
// nodes/replication has no Create/Update/Delete) are not shared: the
// extra runtime fields make them genuinely different shapes.

// Type is a replication job's storage replication mechanism. Proxmox
// currently implements only TypeLocal (node-to-node ZFS replication).
type Type string

const (
	// TypeLocal replicates ZFS-backed guest volumes to another node in
	// the same cluster.
	TypeLocal Type = "local"
)

// RemoveJob marks a replication job for removal: the job stays in the
// configuration until the background removal task has cleaned up its
// snapshots (and, for RemoveJobFull, the replicated volumes on the
// target), then removes itself.
type RemoveJob string

const (
	// RemoveJobLocal removes only the local replication snapshots.
	RemoveJobLocal RemoveJob = "local"
	// RemoveJobFull also removes the replicated volumes on the target.
	RemoveJobFull RemoveJob = "full"
)
