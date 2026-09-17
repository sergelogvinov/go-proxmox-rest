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

package qemu

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// DiskFormat is a target disk image format, used by both Client.Clone
// (a full clone's target format) and Client.MoveDisk (a moved disk's
// target format when it changes storage).
type DiskFormat string

const (
	DiskFormatRaw   DiskFormat = "raw"
	DiskFormatQcow2 DiskFormat = "qcow2"
	DiskFormatVmdk  DiskFormat = "vmdk"
)

// CloneOptions holds the parameters for Client.Clone
// (POST /nodes/{node}/qemu/{vmid}/clone).
type CloneOptions struct {
	// NewID is the VMID for the clone. Required.
	NewID int `url:"newid"`
	// Name sets a name for the new guest.
	Name string `url:"name,omitempty"`
	// Description sets a description for the new guest.
	Description string `url:"description,omitempty"`
	// Pool adds the new guest to the given pool.
	Pool string `url:"pool,omitempty"`
	// Snapname clones from the given snapshot instead of the guest's
	// current state.
	Snapname string `url:"snapname,omitempty"`
	// Storage is the target storage for a full clone. Not allowed for
	// a linked clone.
	Storage string `url:"storage,omitempty"`
	// Format is the target disk format for a full clone. Not allowed
	// for a linked clone.
	Format DiskFormat `url:"format,omitempty"`
	// Full forces a full clone (copying every disk) when true, or a
	// linked clone when false. Proxmox defaults to a full clone for a
	// normal guest and a linked clone for a template.
	Full *bool `url:"full"`
	// Target is the target node. Only allowed when the source guest
	// is on shared storage.
	Target string `url:"target,omitempty"`
	// BWLimit overrides the clone's I/O bandwidth limit, in KiB/s. Zero
	// uses the datacenter/storage default.
	BWLimit int `url:"bwlimit,omitempty"`
}

// Clone creates a copy of a guest (or template) via
// POST /nodes/{node}/qemu/{vmid}/clone. Returns the clone task's UPID.
func (c *Client) Clone(ctx context.Context, vmid int, opts *CloneOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("qemu: clone options are required")
	}
	if opts.NewID == 0 {
		return "", fmt.Errorf("qemu: clone newid is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+c.node+"/qemu/"+strconv.Itoa(vmid)+"/clone", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
