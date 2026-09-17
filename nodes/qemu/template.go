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
)

// Template converts a stopped guest into a template via
// POST /nodes/{node}/qemu/{vmid}/template. disk restricts the conversion
// to a single disk (e.g. "scsi0"), converting only that disk to a base
// image instead of the whole guest; pass "" to convert the whole guest.
// Returns the conversion task's UPID.
func (c *Client) Template(ctx context.Context, vmid int, disk string) (string, error) {
	var p map[string]string
	if disk != "" {
		p = map[string]string{"disk": disk}
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+c.node+"/qemu/"+strconv.Itoa(vmid)+"/template", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
