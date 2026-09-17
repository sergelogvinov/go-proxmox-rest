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

package replication

import "github.com/sergelogvinov/go-proxmox-rest/types"

// Type and RemoveJob are identical between this package and
// cluster/replication: both back onto the same
// PVE::ReplicationConfig section type. They live in the shared types
// package (see its doc comment) and are re-exported here as aliases so
// call sites keep reading as replication.Type,
// replication.RemoveJobFull, ... . JobStatus below is not shared — see
// the types package doc comment.
type (
	Type      = types.Type
	RemoveJob = types.RemoveJob
)

const (
	TypeLocal = types.TypeLocal

	RemoveJobLocal = types.RemoveJobLocal
	RemoveJobFull  = types.RemoveJobFull
)

// JobStatus describes a replication job's configuration merged with its
// current runtime state, as returned by GET /nodes/{node}/replication and
// GET /nodes/{node}/replication/{id}/status.
type JobStatus struct {
	// ID is the job identifier, "<guest>-<jobnum>", e.g. "100-0".
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Type is the replication mechanism; currently always TypeLocal.
	Type Type `json:"type,omitempty" url:"type,omitempty"`
	// Guest is the guest (VM/CT) ID, parsed from ID.
	Guest int `json:"guest,omitempty" url:"guest,omitempty"`
	// JobNum is this job's sequence number for Guest, parsed from ID.
	JobNum int `json:"jobnum,omitempty" url:"jobnum,omitempty"`
	// VMType is the guest's type, "qemu" or "lxc".
	VMType string `json:"vmtype,omitempty" url:"vmtype,omitempty"`
	// Target is the destination node.
	Target string `json:"target,omitempty" url:"target,omitempty"`
	// Source is the node the guest is expected to run on.
	Source string `json:"source,omitempty" url:"source,omitempty"`
	// Schedule is the replication schedule, a subset of systemd
	// calendar events, e.g. "*/15" (the default: every 15 minutes).
	Schedule string `json:"schedule,omitempty" url:"schedule,omitempty"`
	// Rate is the I/O bandwidth limit in MB/s (0: unlimited).
	Rate float64 `json:"rate,omitempty" url:"rate,omitempty"`
	// Disable is true if the job is disabled.
	Disable bool `json:"disable,omitempty" url:"disable,omitempty"`
	// Comment is the job description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
	// RemoveJob is set once the job has been marked for removal.
	RemoveJob RemoveJob `json:"remove_job,omitempty" url:"remove_job,omitempty"`
	// Digest is the configuration digest.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`

	// NextSync is the unix timestamp of the job's next scheduled run,
	// or 0 if none is currently scheduled (e.g. a persistently failing
	// job waiting on an offline target).
	NextSync int64 `json:"next_sync,omitempty" url:"next_sync,omitempty"`
	// LastSync is the unix timestamp of the last successful run.
	LastSync int64 `json:"last_sync,omitempty" url:"last_sync,omitempty"`
	// LastTry is the unix timestamp of the last attempted run,
	// successful or not.
	LastTry int64 `json:"last_try,omitempty" url:"last_try,omitempty"`
	// FailCount is the number of consecutive failed runs.
	FailCount int64 `json:"fail_count,omitempty" url:"fail_count,omitempty"`
	// Error is the last run's error message, populated only if it
	// failed.
	Error string `json:"error,omitempty" url:"error,omitempty"`
	// Duration is the last run's duration, in seconds.
	Duration float64 `json:"duration,omitempty" url:"duration,omitempty"`
	// PID is the replication worker's process ID, populated only while
	// a run is currently in progress.
	PID int `json:"pid,omitempty" url:"pid,omitempty"`
}

// LogEntry is a single replication job log line, as returned by
// GET /nodes/{node}/replication/{id}/log.
type LogEntry struct {
	// N is the line number.
	N int64 `json:"n,omitempty" url:"n,omitempty"`
	// T is the line text.
	T string `json:"t,omitempty" url:"t,omitempty"`
}

// LogOptions filters the log lines returned by Client.Log. A nil
// *LogOptions (or the zero value) requests every line Proxmox has.
type LogOptions struct {
	// Start is the first line number to return.
	Start int `url:"start,omitempty"`
	// Limit caps the number of lines returned.
	Limit int `url:"limit,omitempty"`
}
