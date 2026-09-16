package ceph

// Service names the ceph service scope for Client.Start/Stop/Restart.
// Proxmox accepts a bare kind (affecting every instance of that kind) or
// "kind.instance" (e.g. "mon.pve1", "osd.3") to target one instance; the
// constants below cover only the bare-kind form — build an
// instance-scoped value with a plain string conversion, e.g.
// Service("mon.pve1").
type Service string

const (
	// ServiceAll targets every Ceph service (ceph.target), Proxmox's
	// own default when Start/Stop/Restart's service is "".
	ServiceAll Service = "ceph"
	ServiceMon Service = "mon"
	ServiceMDS Service = "mds"
	ServiceOSD Service = "osd"
	ServiceMgr Service = "mgr"
)

// Release describes one known Ceph release and its installability on
// the queried node, as returned by GET /nodes/{node}/ceph/releases.
type Release struct {
	// Release is the Ceph release code name, e.g. "squid".
	Release string `json:"release,omitempty" url:"release,omitempty"`
	// Version is the Ceph release's major version, e.g. "19.2".
	Version string `json:"version,omitempty" url:"version,omitempty"`
	// Available is true if this release has packages for the node's
	// architecture and current Proxmox VE release.
	Available bool `json:"available,omitempty" url:"available,omitempty"`
	// IsDefault is true if this is the release recommended for new
	// installations.
	IsDefault bool `json:"is-default,omitempty" url:"is-default,omitempty"`
	// Unsupported is true if this release is not yet supported for
	// production use.
	Unsupported bool `json:"unsupported,omitempty" url:"unsupported,omitempty"`
}

// LogEntry is a single Ceph log line, as returned by
// GET /nodes/{node}/ceph/log.
type LogEntry struct {
	// N is the line number.
	N int64 `json:"n,omitempty" url:"n,omitempty"`
	// T is the line text.
	T string `json:"t,omitempty" url:"t,omitempty"`
}

// LogOptions filters the log lines returned by Client.Log. A nil
// *LogOptions (or the zero value) requests Proxmox's default window (the
// first ~50 lines).
type LogOptions struct {
	// Start is the first line number to return.
	Start int `url:"start,omitempty"`
	// Limit caps the number of lines returned.
	Limit int `url:"limit,omitempty"`
}

// Status describes the cluster-wide Ceph status returned by
// GET /nodes/{node}/ceph/status.
//
// This is a duplicate of cluster/ceph.Status: Proxmox's own description
// for this endpoint states the response is identical to
// GET /cluster/ceph/status, but the dependency-direction rule (child
// packages depend only on the root package, never on each other) means
// nodes/ceph cannot import cluster/ceph's type to reuse it. See that
// package's doc comment for the full rationale behind which fields are
// typed versus left as generic values.
type Status struct {
	// FSID is the Ceph cluster's unique identifier.
	FSID string `json:"fsid,omitempty" url:"fsid,omitempty"`
	// Health is the cluster's overall health.
	Health *Health `json:"health,omitempty" url:"health,omitempty"`
	// ElectionEpoch is the monitor election epoch.
	ElectionEpoch int `json:"election_epoch,omitempty" url:"election_epoch,omitempty"`
	// Quorum is the list of monitor ranks in quorum.
	Quorum []int `json:"quorum,omitempty" url:"quorum,omitempty"`
	// QuorumNames is the list of monitor names in quorum.
	QuorumNames []string `json:"quorum_names,omitempty" url:"quorum_names,omitempty"`
	// QuorumAge is how long, in seconds, the quorum has been stable.
	QuorumAge int64 `json:"quorum_age,omitempty" url:"quorum_age,omitempty"`
	// MonMap summarizes the monitor map.
	MonMap *MonMap `json:"monmap,omitempty" url:"monmap,omitempty"`
	// OSDMap summarizes the OSD map.
	OSDMap *OSDMap `json:"osdmap,omitempty" url:"osdmap,omitempty"`
	// PGMap summarizes placement group and storage utilization.
	PGMap *PGMap `json:"pgmap,omitempty" url:"pgmap,omitempty"`
	// MgrMap summarizes the manager map.
	MgrMap *MgrMap `json:"mgrmap,omitempty" url:"mgrmap,omitempty"`
	// FsMap is the raw CephFS map (per-filesystem rank/standby info).
	FsMap map[string]any `json:"fsmap,omitempty" url:"fsmap,omitempty"`
	// ServiceMap is the raw map of auxiliary services (e.g. rgw, iscsi).
	ServiceMap map[string]any `json:"servicemap,omitempty" url:"servicemap,omitempty"`
	// ProgressEvents is the raw map of in-progress long-running
	// operations (e.g. PG recovery/rebalance) and their completion.
	ProgressEvents map[string]any `json:"progress_events,omitempty" url:"progress_events,omitempty"`
}

// Health describes the cluster's overall health, its firing checks, and
// any muted checks.
type Health struct {
	// Status is the overall health: "HEALTH_OK", "HEALTH_WARN" or
	// "HEALTH_ERR".
	Status string `json:"status,omitempty" url:"status,omitempty"`
	// Checks is the map of currently firing health checks, keyed by
	// check code (e.g. "OSD_DOWN", "MON_DISK_LOW"). Each value is the
	// raw check object (severity, summary, detail, muted), plus
	// Proxmox's own "blocks-restart" annotation.
	Checks map[string]any `json:"checks,omitempty" url:"checks,omitempty"`
	// Mutes is the list of currently muted health checks.
	Mutes []any `json:"mutes,omitempty" url:"mutes,omitempty"`
}

// MonMap summarizes the monitor map.
type MonMap struct {
	// Epoch is the monitor map epoch.
	Epoch int `json:"epoch,omitempty" url:"epoch,omitempty"`
	// MinMonReleaseName is the oldest Ceph release name still required
	// to be supported by the monitors (e.g. "reef").
	MinMonReleaseName string `json:"min_mon_release_name,omitempty" url:"min_mon_release_name,omitempty"`
	// NumMons is the number of monitors in the map.
	NumMons int `json:"num_mons,omitempty" url:"num_mons,omitempty"`
	// Mons is the raw list of individual monitor entries.
	Mons []map[string]any `json:"mons,omitempty" url:"mons,omitempty"`
}

// OSDMap summarizes the OSD map.
type OSDMap struct {
	// Epoch is the OSD map epoch.
	Epoch int `json:"epoch,omitempty" url:"epoch,omitempty"`
	// NumOSDs is the total number of OSDs.
	NumOSDs int `json:"num_osds,omitempty" url:"num_osds,omitempty"`
	// NumUpOSDs is the number of OSDs that are up.
	NumUpOSDs int `json:"num_up_osds,omitempty" url:"num_up_osds,omitempty"`
	// NumInOSDs is the number of OSDs that are in the cluster.
	NumInOSDs int `json:"num_in_osds,omitempty" url:"num_in_osds,omitempty"`
	// NumRemappedPGs is the number of placement groups remapped away
	// from their original OSDs.
	NumRemappedPGs int `json:"num_remapped_pgs,omitempty" url:"num_remapped_pgs,omitempty"`
}

// PGState is the placement-group count for a single aggregate PG state
// (e.g. "active+clean").
type PGState struct {
	// StateName is the aggregate PG state, e.g. "active+clean".
	StateName string `json:"state_name,omitempty" url:"state_name,omitempty"`
	// Count is the number of placement groups in this state.
	Count int `json:"count,omitempty" url:"count,omitempty"`
}

// PGMap summarizes placement group counts and storage utilization/IO.
type PGMap struct {
	// PGsByState breaks down placement groups by aggregate state.
	PGsByState []PGState `json:"pgs_by_state,omitempty" url:"pgs_by_state,omitempty"`
	// NumPGs is the total number of placement groups.
	NumPGs int `json:"num_pgs,omitempty" url:"num_pgs,omitempty"`
	// NumPools is the number of pools.
	NumPools int `json:"num_pools,omitempty" url:"num_pools,omitempty"`
	// NumObjects is the total number of stored objects.
	NumObjects int64 `json:"num_objects,omitempty" url:"num_objects,omitempty"`
	// DataBytes is the total logical data size in bytes.
	DataBytes int64 `json:"data_bytes,omitempty" url:"data_bytes,omitempty"`
	// BytesUsed is the total used storage in bytes.
	BytesUsed int64 `json:"bytes_used,omitempty" url:"bytes_used,omitempty"`
	// BytesAvail is the total available storage in bytes.
	BytesAvail int64 `json:"bytes_avail,omitempty" url:"bytes_avail,omitempty"`
	// BytesTotal is the total raw storage in bytes.
	BytesTotal int64 `json:"bytes_total,omitempty" url:"bytes_total,omitempty"`
	// ReadBytesSec is the current read throughput in bytes/second.
	ReadBytesSec int64 `json:"read_bytes_sec,omitempty" url:"read_bytes_sec,omitempty"`
	// WriteBytesSec is the current write throughput in bytes/second.
	WriteBytesSec int64 `json:"write_bytes_sec,omitempty" url:"write_bytes_sec,omitempty"`
	// ReadOpPerSec is the current read IOPS.
	ReadOpPerSec int64 `json:"read_op_per_sec,omitempty" url:"read_op_per_sec,omitempty"`
	// WriteOpPerSec is the current write IOPS.
	WriteOpPerSec int64 `json:"write_op_per_sec,omitempty" url:"write_op_per_sec,omitempty"`
	// DegradedObjects is the number of degraded objects.
	DegradedObjects int64 `json:"degraded_objects,omitempty" url:"degraded_objects,omitempty"`
	// DegradedTotal is the total number of object copies expected.
	DegradedTotal int64 `json:"degraded_total,omitempty" url:"degraded_total,omitempty"`
	// DegradedRatio is the fraction of object copies currently
	// degraded.
	DegradedRatio float64 `json:"degraded_ratio,omitempty" url:"degraded_ratio,omitempty"`
}

// MgrMap summarizes the manager map.
type MgrMap struct {
	// Epoch is the manager map epoch.
	Epoch int `json:"epoch,omitempty" url:"epoch,omitempty"`
	// ActiveGID is the active manager's global ID.
	ActiveGID int64 `json:"active_gid,omitempty" url:"active_gid,omitempty"`
	// ActiveName is the active manager's name.
	ActiveName string `json:"active_name,omitempty" url:"active_name,omitempty"`
	// ActiveAddr is the active manager's address.
	ActiveAddr string `json:"active_addr,omitempty" url:"active_addr,omitempty"`
	// Available is true if an active manager is present.
	Available bool `json:"available,omitempty" url:"available,omitempty"`
	// Standbys is the raw list of standby manager entries.
	Standbys []map[string]any `json:"standbys,omitempty" url:"standbys,omitempty"`
}
