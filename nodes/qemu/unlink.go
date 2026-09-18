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

// UnlinkOptions holds the parameters for Client.Unlink
// (PUT /nodes/{node}/qemu/{vmid}/unlink).
type UnlinkOptions struct {
	// IDList lists the disk config keys to unlink, e.g.
	// []string{"scsi1", "unused0"}. Required.
	IDList []string `url:"idlist"`
	// Force allows unlinking a disk that's still referenced/in use.
	// Root only.
	Force bool `url:"force,omitempty"`
}

// Unlink removes one or more disk images from a guest's configuration
// via PUT /nodes/{node}/qemu/{vmid}/unlink. This is a thin,
// synchronous convenience wrapper around UpdateConfig's Delete
// mechanism, exposed by Proxmox as its own endpoint.
//
// +proxmox:rbac:path=/vms/{vmid},method=PUT,privs=VM.Config.Disk,match=all
func (c *Client) Unlink(ctx context.Context, vmid int, opts *UnlinkOptions) error {
	if opts == nil {
		return fmt.Errorf("qemu: unlink options are required")
	}
	if len(opts.IDList) == 0 {
		return fmt.Errorf("qemu: unlink requires at least one disk id")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return err
	}

	return c.client.Update(ctx, "/nodes/"+c.node+"/qemu/"+strconv.Itoa(vmid)+"/unlink", nil, p)
}
