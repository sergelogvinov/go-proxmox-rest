package storage

import (
	"context"
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// pruneBackupsResource provides access to
// /nodes/{node}/storage/{storage}/prunebackups.
type pruneBackupsResource struct {
	client Getter
	node   string
}

// DryRun previews which backups a prune would keep or remove via
// GET /nodes/{node}/storage/{storage}/prunebackups, without deleting
// anything. opts may be nil to use the storage's configured retention.
func (r *pruneBackupsResource) DryRun(ctx context.Context, storageID string, opts *PruneOptions) ([]PruneEntry, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var entries []PruneEntry
	if err := r.client.Get(ctx, "/nodes/"+r.node+"/storage/"+storageID+"/prunebackups", &entries, p); err != nil {
		return nil, err
	}

	return entries, nil
}

// Delete prunes backups via
// DELETE /nodes/{node}/storage/{storage}/prunebackups, permanently
// removing every volume DryRun would mark PruneMarkRemove. Unlike
// DryRun, opts.PruneBackups is required here — Proxmox does not fall
// back to the storage's configured retention for the real prune.
// Returns the pruning task's UPID.
func (r *pruneBackupsResource) Delete(ctx context.Context, storageID string, opts *PruneOptions) (string, error) {
	if opts == nil || opts.PruneBackups == "" {
		return "", fmt.Errorf("storage: prune-backups retention is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := r.client.Delete(ctx, "/nodes/"+r.node+"/storage/"+storageID+"/prunebackups", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
