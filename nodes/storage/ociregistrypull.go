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

// OCIRegistryPullOptions holds the parameters for
// Client.OCIRegistryPull
// (POST /nodes/{node}/storage/{storage}/oci-registry-pull).
type OCIRegistryPullOptions struct {
	// Reference is the OCI image to pull, e.g.
	// "docker.io/library/nginx:latest"; a tag (or digest) suffix is
	// required. Required.
	Reference string `url:"reference"`
	// Filename overrides the destination file name; Proxmox normalizes
	// it. Empty lets Proxmox derive one from Reference.
	Filename string `url:"filename,omitempty"`
}

// OCIRegistryPull pulls an OCI image from a registry directly into
// storage via POST /nodes/{node}/storage/{storage}/oci-registry-pull.
// Returns the pull task's UPID.
// Besides Datastore.AllocateTemplate on the storage, Proxmox also
// requires Sys.AccessNetwork on the node, since this makes the node
// fetch from an arbitrary registry. The marker records the
// storage-privilege branch; the node-network requirement is documented
// here since the annotation format has no way to express a second,
// ANDed path.
//
// +proxmox:rbac:path=/storage/{storage},method=POST,privs=Datastore.AllocateTemplate,match=all
func (c *Client) OCIRegistryPull(ctx context.Context, storageID string, opts *OCIRegistryPullOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("storage: oci-registry-pull options are required")
	}
	if opts.Reference == "" {
		return "", fmt.Errorf("storage: oci-registry-pull reference is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+c.node+"/storage/"+storageID+"/oci-registry-pull", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
