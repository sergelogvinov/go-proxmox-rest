package ceph

// Status describes the cluster-wide Ceph status returned by
// GET /cluster/ceph/status.
//
// Proxmox itself declares no schema for this endpoint (its API viewer marks
// the return type as a bare "object"): it is the raw `ceph status` RADOS
// command output, merged with `ceph health detail` and, on older Ceph
// releases, a re-fetched mon/mgr map, plus a `blocks-restart` flag Proxmox
// adds to every firing health check. Only the fields that are stable across
// Ceph releases and commonly consumed are typed; the rest — health checks,
// mutes, individual monitor/manager entries, the CephFS and service maps,
// and in-progress operation events — are exposed as generic values so no
// information is lost to a shape this client does not pin down.
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

// Health describes the cluster's overall health, its firing checks, and any
// muted checks.
type Health struct {
	// Status is the overall health: "HEALTH_OK", "HEALTH_WARN" or
	// "HEALTH_ERR".
	Status string `json:"status,omitempty" url:"status,omitempty"`
	// Checks is the map of currently firing health checks, keyed by
	// check code (e.g. "OSD_DOWN", "MON_DISK_LOW"). Each value is the raw
	// check object (severity, summary, detail, muted), plus Proxmox's own
	// "blocks-restart" annotation.
	Checks map[string]any `json:"checks,omitempty" url:"checks,omitempty"`
	// Mutes is the list of currently muted health checks.
	Mutes []any `json:"mutes,omitempty" url:"mutes,omitempty"`
}

// MonMap summarizes the monitor map.
type MonMap struct {
	// Epoch is the monitor map epoch.
	Epoch int `json:"epoch,omitempty" url:"epoch,omitempty"`
	// MinMonReleaseName is the oldest Ceph release name still required to
	// be supported by the monitors (e.g. "reef").
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
	// DegradedRatio is the fraction of object copies currently degraded.
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
