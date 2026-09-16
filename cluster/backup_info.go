package cluster

import (
	"context"
)

// BackupInfoGuest describes a guest that isn't covered by any vzdump
// backup job (neither a job in jobs.cfg nor a legacy vzdump.cron entry),
// as returned by GET /cluster/backup-info/not-backed-up.
//
// The result is filtered to guests the caller has VM.Audit permission on;
// Name is only set when the guest's configuration could be read.
type BackupInfoGuest struct {
	// VMID is the guest's ID.
	VMID int `json:"vmid,omitempty" url:"vmid,omitempty"`
	// Name is the guest's name (QEMU) or hostname (LXC), if known.
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Type is the guest kind: "qemu" or "lxc".
	Type string `json:"type,omitempty" url:"type,omitempty"`
}

// BackupInfoNotBackedUp retrieves every guest not covered by any backup
// job via GET /cluster/backup-info/not-backed-up.
func (c *Client) BackupInfoNotBackedUp(ctx context.Context) ([]BackupInfoGuest, error) {
	var guests []BackupInfoGuest
	if err := c.client.Get(ctx, "/cluster/backup-info/not-backed-up", &guests, nil); err != nil {
		return nil, err
	}

	return guests, nil
}
