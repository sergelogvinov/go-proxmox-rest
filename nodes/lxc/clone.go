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

package lxc

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// CloneOptions holds the parameters for Client.Clone
// (POST /nodes/{node}/lxc/{vmid}/clone).
type CloneOptions struct {
	// NewID is the VMID for the clone. Required.
	NewID int `url:"newid"`
	// Hostname sets a host name for the new container.
	Hostname string `url:"hostname,omitempty"`
	// Description sets a description for the new container.
	Description string `url:"description,omitempty"`
	// Pool adds the new container to the given pool.
	Pool string `url:"pool,omitempty"`
	// Snapname clones from the given snapshot instead of the
	// container's current state.
	Snapname string `url:"snapname,omitempty"`
	// Storage is the target storage for a full clone.
	Storage string `url:"storage,omitempty"`
	// Full forces a full clone (copying every disk) when true, or a
	// linked clone when false. Proxmox defaults to a full clone for a
	// normal container and a linked clone for a template.
	Full *bool `url:"full"`
	// Target is the target node. Only allowed when the source
	// container is on shared storage.
	Target string `url:"target,omitempty"`
	// BWLimit overrides the clone's I/O bandwidth limit, in KiB/s. Zero
	// uses the datacenter/storage default.
	BWLimit float64 `url:"bwlimit,omitempty"`
}

// Clone creates a copy of a container (or template) via
// POST /nodes/{node}/lxc/{vmid}/clone. Returns the clone task's UPID.
// Target allocation is an alternative: VM.Allocate on /vms/{newid}, or on
// /pool/{pool} when a pool is supplied. Storage and SDN checks are conditional.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Clone,match=all
func (c *Client) Clone(ctx context.Context, vmid int, opts *CloneOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("lxc: clone options are required")
	}
	if opts.NewID == 0 {
		return "", fmt.Errorf("lxc: clone newid is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+c.node+"/lxc/"+strconv.Itoa(vmid)+"/clone", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
