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
)

// Template converts a stopped container into a template via
// POST /nodes/{node}/lxc/{vmid}/template.
//
// Unlike nodes/qemu's Template, this endpoint's own schema declares
// `returns => { type => 'null' }`: Proxmox forks the conversion as a
// background task internally but never returns its UPID to the caller,
// so there is nothing for this method to hand back beyond success/error.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Allocate,match=all
func (c *Client) Template(ctx context.Context, vmid int) error {
	return c.client.Create(ctx, "/nodes/"+c.node+"/lxc/"+strconv.Itoa(vmid)+"/template", nil, nil)
}
