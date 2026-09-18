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

// CopyOptions holds the parameters for Client.Content(storageID).Copy
// (POST /nodes/{node}/storage/{storage}/content/{volume}).
type CopyOptions struct {
	// Target is the destination volume name, relative to the same
	// storage unless TargetNode says otherwise. Required.
	Target string `url:"target"`
	// TargetNode moves the volume to a different node's copy of the
	// same (shared, or identically named) storage. Empty keeps the
	// source node.
	TargetNode string `url:"target_node,omitempty"`
}

// Copy copies or moves an existing volume to a new volume id and/or a
// different node via
// POST /nodes/{node}/storage/{storage}/content/{volume}. volume may be a
// bare volume name (resolved against the resource's storageID) or a
// full "storage:name" volume id, matching Get's convention. Returns the
// copy task's UPID.
//
// Proxmox itself marks this endpoint experimental ("Copy a volume. This
// is experimental code - do not use.") — it has shipped unchanged across
// many releases and this package's own e2e suite exercises it, but treat
// it accordingly.
//
// +proxmox:rbac:path=/storage/{storage},method=POST,privs=Datastore.AllocateSpace,match=all
func (r *contentResource) Copy(ctx context.Context, volume string, opts *CopyOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("storage: content copy options are required")
	}
	if opts.Target == "" {
		return "", fmt.Errorf("storage: content copy target is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := r.client.Create(ctx, "/nodes/"+r.node+"/storage/"+r.storageID+"/content/"+volume, &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
