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

// MoveDiskOptions holds the parameters for Client.MoveDisk
// (POST /nodes/{node}/qemu/{vmid}/move_disk).
type MoveDiskOptions struct {
	// Disk is the disk to move, e.g. "scsi0", "unused3". Required.
	Disk string `url:"disk"`
	// TargetVMID moves the disk to a different guest instead of just
	// to a different storage on the same guest.
	TargetVMID int `url:"target-vmid,omitempty"`
	// Storage is the target storage. Proxmox requires either Storage
	// or TargetVMID to be set.
	Storage string `url:"storage,omitempty"`
	// Format is the target disk format; only used when moving to a
	// different storage.
	Format DiskFormat `url:"format,omitempty"`
	// Delete removes the original disk after a successful copy;
	// Proxmox otherwise keeps it as an unused disk.
	Delete bool `url:"delete,omitempty"`
	// Digest prevents the move if the source guest's configuration has
	// changed since the value was read.
	Digest string `url:"digest,omitempty"`
	// BWLimit overrides the move's I/O bandwidth limit, in KiB/s. Zero
	// uses the datacenter/storage default.
	BWLimit int `url:"bwlimit,omitempty"`
	// TargetDisk is the config key the disk is moved to on the target
	// guest (e.g. "ide0"), when TargetVMID is set. Defaults to Disk.
	TargetDisk string `url:"target-disk,omitempty"`
	// TargetDigest prevents the move if the target guest's
	// configuration has changed since the value was read. Only
	// meaningful together with TargetVMID.
	TargetDigest string `url:"target-digest,omitempty"`
}

// MoveDisk moves a guest's disk to a different storage, or to a
// different guest entirely, via
// POST /nodes/{node}/qemu/{vmid}/move_disk. Returns the move task's
// UPID.
// Moving to storage conditionally requires Datastore.AllocateSpace there;
// reassignment conditionally requires VM.Config.Disk on the target VM and
// Datastore.AllocateSpace on the source volume's storage.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Config.Disk,match=all
func (c *Client) MoveDisk(ctx context.Context, vmid int, opts *MoveDiskOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("qemu: move disk options are required")
	}
	if opts.Disk == "" {
		return "", fmt.Errorf("qemu: move disk requires a disk")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+c.node+"/qemu/"+strconv.Itoa(vmid)+"/move_disk", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
