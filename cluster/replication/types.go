package replication

import (
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
	"github.com/sergelogvinov/go-proxmox-rest/types"
)

// Type and RemoveJob are identical between this package and
// nodes/replication: both back onto the same PVE::ReplicationConfig
// section type. They live in the shared types package (see its doc
// comment) and are re-exported here as aliases so call sites keep
// reading as replication.Type, replication.RemoveJobFull, ... . Job
// below is not shared — see the types package doc comment.
type (
	Type      = types.Type
	RemoveJob = types.RemoveJob
)

const (
	TypeLocal = types.TypeLocal

	RemoveJobLocal = types.RemoveJobLocal
	RemoveJobFull  = types.RemoveJobFull
)

// Job describes a replication job as returned by GET /cluster/replication
// and GET /cluster/replication/{id}.
//
// Guest and JobNum are parsed from ID (the "<guest>-<jobnum>" pair) by
// Proxmox and are not settable; Digest is only populated by Get, not List.
type Job struct {
	// ID is the job identifier, "<guest>-<jobnum>", e.g. "100-0".
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Type is the replication mechanism; currently always TypeLocal.
	Type Type `json:"type,omitempty" url:"type,omitempty"`
	// Guest is the guest (VM/CT) ID, parsed from ID.
	Guest int `json:"guest,omitempty" url:"guest,omitempty"`
	// JobNum is this job's sequence number for Guest, parsed from ID.
	JobNum int `json:"jobnum,omitempty" url:"jobnum,omitempty"`
	// Target is the destination node.
	Target string `json:"target,omitempty" url:"target,omitempty"`
	// Source is the node the guest is expected to run on; used
	// internally to detect whether the guest was migrated/stolen.
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
	// Digest is the configuration digest, usable with JobOptions.Digest
	// to guard concurrent updates. Only populated by Get, not List.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// JobOptions holds the write parameters shared by POST /cluster/replication
// (Create) and PUT /cluster/replication/{id} (Update).
//
// ID, Type and Target are required by Create. Update ignores Type (Proxmox
// does not accept it there — jobsResource.Update strips it from the
// encoded request) but still requires ID and Target in the body even
// though ID is already part of the URL and Target cannot actually change
// (jobsResource.Update fills in ID from the requested id, so callers only
// need to resend Target). Pointer fields are always sent when non-nil
// (even when zero/empty), which is how Update expresses "clear this
// field" — Create simply sends whatever is set. Delete and Digest are
// meaningful to Update only.
type JobOptions struct {
	// ID is the job identifier, "<guest>-<jobnum>". Required by Create;
	// not settable on Update.
	ID string `url:"id"`
	// Type is the replication mechanism; currently always TypeLocal.
	// Required by Create; not sent on Update.
	Type Type `url:"type"`
	// Target is the destination node. Required by both Create and
	// Update (Proxmox treats it as fixed after creation, but still
	// requires it to be resent, unchanged, on every Update).
	Target string `url:"target"`
	// Source is the node the guest is expected to run on. Leave unset
	// on Create to let Proxmox detect it from the guest's current node.
	Source *string `url:"source"`
	// Schedule is the replication schedule, a subset of systemd
	// calendar events, e.g. "*/15".
	Schedule *string `url:"schedule"`
	// Rate is the I/O bandwidth limit in MB/s (0: unlimited).
	Rate *float64 `url:"rate"`
	// Disable disables the job without deleting it.
	Disable *bool `url:"disable"`
	// Comment is the job description.
	Comment *string `url:"comment"`
	// RemoveJob marks the job for removal instead of updating it in
	// place; Proxmox's replication runner then removes snapshots (and,
	// for RemoveJobFull, the target's volumes) and deletes the job.
	// jobsResource.Delete is the usual way to do this instead.
	RemoveJob *RemoveJob `url:"remove_job"`
	// Delete lists properties to reset to their default value (Update
	// only). Proxmox refuses "target" here (fixed) and any option that
	// is not otherwise optional.
	Delete []string `url:"delete"`
	// Digest prevents changes if the current configuration has changed
	// in between (value from the corresponding Get). Update only.
	Digest string `url:"digest"`
}

// encode converts the options to form parameters.
func (o *JobOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("replication: job options are required")
	}

	return params.Encode(o)
}
