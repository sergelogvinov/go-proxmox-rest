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

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// ResizeOptions holds the parameters for Client.Resize
// (PUT /nodes/{node}/lxc/{vmid}/resize). Unlike nodes/qemu's version,
// LXC has no skiplock option on this endpoint.
type ResizeOptions struct {
	// Disk is the mount point to resize, e.g. "rootfs" or "mp0".
	// Required.
	Disk string `url:"disk"`
	// Size is the new size, e.g. "32G" for an absolute size or "+4G"
	// to grow by that amount. Shrinking is not supported by Proxmox.
	// Required.
	Size string `url:"size"`
	// Digest prevents the resize if the container's configuration has
	// changed since the value was read.
	Digest string `url:"digest,omitempty"`
}

// Resize extends a container mount point's size via
// PUT /nodes/{node}/lxc/{vmid}/resize. Returns the resize task's UPID.
func (c *Client) Resize(ctx context.Context, vmid int, opts *ResizeOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("lxc: resize options are required")
	}
	if opts.Disk == "" {
		return "", fmt.Errorf("lxc: resize requires a disk")
	}
	if opts.Size == "" {
		return "", fmt.Errorf("lxc: resize requires a size")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Update(ctx, "/nodes/"+c.node+"/lxc/"+strconv.Itoa(vmid)+"/resize", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
