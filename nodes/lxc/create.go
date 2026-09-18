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
	"fmt"
	"strconv"
)

// CreateOptions holds the parameters for Client.Create
// (POST /nodes/{node}/lxc). VMID and OSTemplate are required; Config
// embeds the same fields Config/UpdateConfig already use (Hostname,
// Cores, Memory, RootFS, Net, ...) so a new container's hardware is
// described with no second schema to learn — see Config's own
// field-by-field documentation.
type CreateOptions struct {
	// Config describes the new container's configuration. The zero
	// value creates a container with Proxmox's own defaults for every
	// field.
	Config

	// VMID is the new container's ID. Required.
	VMID int
	// OSTemplate is the OS template volume (or backup archive) to
	// create the container from, e.g.
	// "local:vztmpl/debian-12-standard_12.2-1_amd64.tar.zst". Required.
	OSTemplate string
	// Pool adds the new container to the given pool.
	Pool string
	// Start starts the container immediately after creation completes.
	Start bool
}

// Create creates a new container via POST /nodes/{node}/lxc. Returns
// the creation task's UPID.
//
// Create issues a single HTTP call and does not convert the result into
// a template: pass Config.Template to mark it as one directly.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Allocate,match=all
func (c *Client) Create(ctx context.Context, opts *CreateOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("lxc: create options are required")
	}
	if opts.VMID == 0 {
		return "", fmt.Errorf("lxc: create vmid is required")
	}
	if opts.OSTemplate == "" {
		return "", fmt.Errorf("lxc: create ostemplate is required")
	}

	p, err := encodeConfig(&opts.Config)
	if err != nil {
		return "", err
	}

	p["vmid"] = strconv.Itoa(opts.VMID)
	p["ostemplate"] = opts.OSTemplate

	if opts.Pool != "" {
		p["pool"] = opts.Pool
	}
	if opts.Start {
		p["start"] = "1"
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+c.node+"/lxc", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
