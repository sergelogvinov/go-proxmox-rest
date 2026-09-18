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

// Interfaces retrieves the IP addresses of a container's network
// interfaces via GET /nodes/{node}/lxc/{vmid}/interfaces. Unlike
// nodes/qemu/agent's NetworkGetInterfaces, this does not require a guest
// agent — Proxmox reads the addresses directly from the container's
// network namespace.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (c *Client) Interfaces(ctx context.Context, vmid int) ([]Interface, error) {
	var ifaces []Interface
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/lxc/"+strconv.Itoa(vmid)+"/interfaces", &ifaces, nil); err != nil {
		return nil, err
	}

	return ifaces, nil
}
