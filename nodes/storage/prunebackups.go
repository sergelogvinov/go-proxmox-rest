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
//
// +proxmox:rbac:path=/storage/{storage},method=GET,privs=Datastore.Audit;Datastore.AllocateSpace,match=any
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
// This user=>all endpoint has no fixed privilege. With vmid, it requires both
// Datastore.AllocateSpace and VM.Backup; without vmid, Datastore.Allocate applies.
// The marker records the no-vmid branch; the vmid-specific alternative is
// documented above.
//
// +proxmox:rbac:path=/storage/{storage},method=DELETE,privs=Datastore.Allocate,match=all
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
