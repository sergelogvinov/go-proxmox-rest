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

// CloudInitPendingEntry describes a single cloud-init configuration
// key's current and pending state, as returned by
// Client.CloudInitPending.
type CloudInitPendingEntry struct {
	// Key is the configuration option name, e.g. "ciuser", "ipconfig0".
	Key string `json:"key,omitempty" url:"key,omitempty"`
	// Value is the value used to generate the guest's current
	// cloud-init image. Absent for a newly-added key that has never
	// been applied yet. If Key is "cipassword", this is redacted to
	// "**********" rather than the real password.
	Value string `json:"value,omitempty" url:"value,omitempty"`
	// Pending is the new value queued for the next cloud-init image
	// regeneration (Client.CloudInitUpdate), if different from Value.
	// Same "cipassword" redaction as Value applies.
	Pending string `json:"pending,omitempty" url:"pending,omitempty"`
	// Delete is 1 if Key is queued for removal on the next
	// regeneration.
	Delete int `json:"delete,omitempty" url:"delete,omitempty"`
}

// CloudInitPending retrieves the guest's cloud-init configuration, with
// both its current (applied) and pending (queued but not yet
// regenerated) values, via GET /nodes/{node}/qemu/{vmid}/cloudinit.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (c *Client) CloudInitPending(ctx context.Context, vmid int) ([]CloudInitPendingEntry, error) {
	var entries []CloudInitPendingEntry
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/qemu/"+strconv.Itoa(vmid)+"/cloudinit", &entries, nil); err != nil {
		return nil, err
	}

	return entries, nil
}

// CloudInitUpdate regenerates the guest's cloud-init config drive via
// PUT /nodes/{node}/qemu/{vmid}/cloudinit, applying every pending
// cloud-init value reported by CloudInitPending.
//
// +proxmox:rbac:path=/vms/{vmid},method=PUT,privs=VM.Config.Cloudinit,match=all
func (c *Client) CloudInitUpdate(ctx context.Context, vmid int) error {
	return c.client.Update(ctx, "/nodes/"+c.node+"/qemu/"+strconv.Itoa(vmid)+"/cloudinit", nil, nil)
}
