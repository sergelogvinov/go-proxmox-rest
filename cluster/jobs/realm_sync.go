package jobs

import (
	"context"
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Scope selects what a realm-sync job synchronizes from the LDAP/AD realm.
type Scope string

const (
	// ScopeUsers syncs only users.
	ScopeUsers Scope = "users"
	// ScopeGroups syncs only groups.
	ScopeGroups Scope = "groups"
	// ScopeBoth syncs both users and groups.
	ScopeBoth Scope = "both"
)

// RealmSyncJob describes a realm-sync job, as returned by
// realmSyncResource.List and realmSyncResource.Get.
//
// List additionally computes LastRun/NextRun and always sets ID (from the
// job's configuration key); Get returns the raw stored job entry, which
// has none of those three fields populated, since Proxmox never persists
// a job's own id inside its configuration.
type RealmSyncJob struct {
	// ID is the job identifier. Only populated by List.
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Realm is the LDAP/AD realm being synced from. Fixed at creation.
	Realm string `json:"realm,omitempty" url:"realm,omitempty"`
	// Scope selects what is synced.
	Scope Scope `json:"scope,omitempty" url:"scope,omitempty"`
	// RemoveVanished lists what is removed when a user/group vanishes
	// during a sync: a semicolon-separated subset of "entry",
	// "properties", "acl", or "none".
	RemoveVanished string `json:"remove-vanished,omitempty" url:"remove-vanished,omitempty"`
	// EnableNew is false if newly synced users are not enabled
	// immediately.
	EnableNew bool `json:"enable-new,omitempty" url:"enable-new,omitempty"`
	// Enabled is false if the job is disabled.
	Enabled bool `json:"enabled,omitempty" url:"enabled,omitempty"`
	// Schedule is the sync schedule, a subset of systemd calendar
	// events.
	Schedule string `json:"schedule,omitempty" url:"schedule,omitempty"`
	// Comment is the job description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
	// LastRun is the UNIX timestamp of the last execution. Only
	// populated by List, and only once the job has run at least once.
	LastRun int64 `json:"last-run,omitempty" url:"last-run,omitempty"`
	// NextRun is the UNIX timestamp of the next scheduled run. Only
	// populated by List.
	NextRun int64 `json:"next-run,omitempty" url:"next-run,omitempty"`
}

// RealmSyncOptions holds the write parameters shared by
// realmSyncResource.Create (POST /cluster/jobs/realm-sync/{id}) and
// realmSyncResource.Update (PUT /cluster/jobs/realm-sync/{id}).
//
// Realm and Scope are required by Create. Realm is fixed: Proxmox does not
// accept changing it on an existing job, so realmSyncResource.Update drops
// it from the encoded request even if set. Pointer fields are always sent
// when non-nil (even when zero/empty), which is how Update expresses
// "clear this field" — Create simply sends whatever is set. Delete and
// Digest are meaningful to Update only.
type RealmSyncOptions struct {
	// Realm is the LDAP/AD realm to sync from. Required by Create;
	// dropped by Update.
	Realm string `url:"realm,omitempty"`
	// Scope selects what to sync. Required by Create.
	Scope *Scope `url:"scope"`
	// Schedule is the sync schedule, a subset of systemd calendar
	// events, e.g. "*/30". Required by Create.
	Schedule *string `url:"schedule"`
	// Enabled enables/disables the job. Defaults to true.
	Enabled *bool `url:"enabled"`
	// Comment is the job description.
	Comment *string `url:"comment"`
	// RemoveVanished lists what to remove when a user/group vanishes
	// during a sync: a semicolon-separated subset of "entry",
	// "properties", "acl", or "none" (the default).
	RemoveVanished *string `url:"remove-vanished"`
	// EnableNew enables newly synced users immediately. Defaults to
	// true.
	EnableNew *bool `url:"enable-new"`
	// Delete lists properties to reset to their default value (Update
	// only): "comment", "remove-vanished", or "enable-new".
	Delete []string `url:"delete"`
	// Digest prevents changes if the current configuration has changed
	// in between (value from the corresponding Get). Update only.
	Digest string `url:"digest,omitempty"`
}

// encode converts the options to form parameters.
func (o *RealmSyncOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("jobs: realm-sync options are required")
	}

	return params.Encode(o)
}

// realmSyncResource provides access to the /cluster/jobs/realm-sync
// resource tree, behind Client.RealmSync.
type realmSyncResource struct {
	client Getter
}

// List retrieves all realm-sync jobs via GET /cluster/jobs/realm-sync.
func (r *realmSyncResource) List(ctx context.Context) ([]RealmSyncJob, error) {
	var jobs []RealmSyncJob
	if err := r.client.Get(ctx, "/cluster/jobs/realm-sync", &jobs, nil); err != nil {
		return nil, err
	}

	return jobs, nil
}

// Get retrieves a single realm-sync job via
// GET /cluster/jobs/realm-sync/{id}.
func (r *realmSyncResource) Get(ctx context.Context, id string) (*RealmSyncJob, error) {
	var job RealmSyncJob
	if err := r.client.Get(ctx, "/cluster/jobs/realm-sync/"+id, &job, nil); err != nil {
		return nil, err
	}

	return &job, nil
}

// Create creates a new realm-sync job via
// POST /cluster/jobs/realm-sync/{id}. Unlike most other Create endpoints
// in this module layout, Proxmox addresses the new job's id via the URL
// here rather than a body parameter, so id is a required call argument
// rather than a field on RealmSyncOptions.
func (r *realmSyncResource) Create(ctx context.Context, id string, opts *RealmSyncOptions) error {
	if id == "" {
		return fmt.Errorf("jobs: realm-sync job id is required")
	}
	if opts == nil {
		return fmt.Errorf("jobs: realm-sync options are required")
	}
	if opts.Realm == "" {
		return fmt.Errorf("jobs: realm-sync realm is required")
	}
	if opts.Scope == nil {
		return fmt.Errorf("jobs: realm-sync scope is required")
	}
	if opts.Schedule == nil || *opts.Schedule == "" {
		return fmt.Errorf("jobs: realm-sync schedule is required")
	}

	params, err := opts.encode()
	if err != nil {
		return err
	}

	return r.client.Create(ctx, "/cluster/jobs/realm-sync/"+id, nil, params)
}

// Update modifies an existing realm-sync job via
// PUT /cluster/jobs/realm-sync/{id}.
func (r *realmSyncResource) Update(ctx context.Context, id string, opts *RealmSyncOptions) error {
	params, err := opts.encode()
	if err != nil {
		return err
	}
	delete(params, "realm")

	return r.client.Update(ctx, "/cluster/jobs/realm-sync/"+id, nil, params)
}

// Delete removes a realm-sync job via
// DELETE /cluster/jobs/realm-sync/{id}.
func (r *realmSyncResource) Delete(ctx context.Context, id string) error {
	return r.client.Delete(ctx, "/cluster/jobs/realm-sync/"+id, nil, nil)
}
