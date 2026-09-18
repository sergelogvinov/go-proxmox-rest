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
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// DeleteOptions holds the parameters for Client.Delete
// (DELETE /nodes/{node}/qemu/{vmid}). A nil *DeleteOptions is valid and
// uses Proxmox's defaults.
type DeleteOptions struct {
	// DestroyUnreferencedDisks additionally destroys any disks matching
	// the guest's VMID on all enabled storages that aren't referenced in
	// its config.
	DestroyUnreferencedDisks bool `url:"destroy-unreferenced-disks,omitempty"`
	// Purge removes the VMID from other configurations that reference
	// it, such as backup and replication jobs and HA.
	Purge bool `url:"purge,omitempty"`
	// Skiplock ignores locks on the guest. Root only.
	Skiplock bool `url:"skiplock,omitempty"`
}

// Delete destroys a guest and all volumes it owns via
// DELETE /nodes/{node}/qemu/{vmid}, also removing any VM-specific
// permissions and firewall rules. opts may be nil to use Proxmox's
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
	if err := c.client.Delete(ctx, "/nodes/"+c.node+"/qemu/"+strconv.Itoa(vmid), &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
