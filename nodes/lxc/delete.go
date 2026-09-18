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
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// DeleteOptions holds the parameters for Client.Delete
// (DELETE /nodes/{node}/lxc/{vmid}). A nil *DeleteOptions is valid and
// uses Proxmox's defaults. Unlike nodes/qemu's version, LXC has no
// skiplock option on this endpoint, but adds Force.
type DeleteOptions struct {
	// DestroyUnreferencedDisks additionally destroys any disks matching
	// the container's VMID on all enabled storages that aren't
	// referenced in its config.
	DestroyUnreferencedDisks bool `url:"destroy-unreferenced-disks,omitempty"`
	// Force destroys the container even while it's running.
	Force bool `url:"force,omitempty"`
	// Purge removes the VMID from other configurations that reference
	// it, such as backup and replication jobs and HA. Related ACLs and
	// firewall entries are always removed regardless of this setting.
	Purge bool `url:"purge,omitempty"`
}

// Delete destroys a container and all files it owns via
// DELETE /nodes/{node}/lxc/{vmid}. opts may be nil to use Proxmox's
// defaults. Returns the destroy task's UPID.
//
// +proxmox:rbac:path=/vms/{vmid},method=DELETE,privs=VM.Allocate,match=all
func (c *Client) Delete(ctx context.Context, vmid int, opts *DeleteOptions) (string, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return "", err
		}
	}

	var upid string
	if err := c.client.Delete(ctx, "/nodes/"+c.node+"/lxc/"+strconv.Itoa(vmid), &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
