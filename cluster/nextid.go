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

package cluster

import (
	"context"
	"strconv"
)

// NextID retrieves the next free VMID via GET /cluster/nextid.
//
// With vmid == 0, Proxmox returns the next free VMID from the configured
// (or default) auto-allocation range. With vmid != 0, Proxmox instead
// asserts that vmid itself is free at the time of the check and returns it
// unchanged, or returns an error if it is already in use.
func (c *Client) NextID(ctx context.Context, vmid int) (int, error) {
	var params map[string]string
	if vmid != 0 {
		params = map[string]string{"vmid": strconv.Itoa(vmid)}
	}

	var nextID int
	if err := c.client.Get(ctx, "/cluster/nextid", &nextID, params); err != nil {
		return 0, err
	}

	return nextID, nil
}
