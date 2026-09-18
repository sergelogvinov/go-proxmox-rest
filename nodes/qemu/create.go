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
)

// CreateOptions holds the parameters for Client.Create
// (POST /nodes/{node}/qemu). VMID is required; Config embeds the same
// fields Config/UpdateConfig already use (Boot, Agent, Cores, Memory,
// SCSIHW, SCSI, Net, ...) so a new guest's hardware is described with no
// second schema to learn — see Config's own field-by-field documentation.
type CreateOptions struct {
	// Config describes the new guest's configuration. The zero value
	// creates a guest with Proxmox's own defaults for every field.
	Config

	// VMID is the new guest's ID. Required.
	VMID int
	// Pool adds the new guest to the given pool.
	Pool string
	// Start starts the guest immediately after creation completes.
	Start bool
}

// Create creates a new guest via POST /nodes/{node}/qemu. Returns the
// creation task's UPID.
//
// Create issues a single HTTP call and does not convert the result into
// a template: pass Config.Template to mark it as one directly, or call
// Client.Template after the creation task completes to convert only a
// specific disk.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Allocate,match=all
func (c *Client) Create(ctx context.Context, opts *CreateOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("qemu: create options are required")
	}
	if opts.VMID == 0 {
		return "", fmt.Errorf("qemu: create vmid is required")
	}

	p, err := encodeConfig(&opts.Config)
	if err != nil {
		return "", err
	}

	p["vmid"] = strconv.Itoa(opts.VMID)

	if opts.Pool != "" {
		p["pool"] = opts.Pool
	}
	if opts.Start {
		p["start"] = "1"
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+c.node+"/qemu", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
