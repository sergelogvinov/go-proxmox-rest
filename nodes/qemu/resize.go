package qemu

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// ResizeOptions holds the parameters for Client.Resize
// (PUT /nodes/{node}/qemu/{vmid}/resize).
type ResizeOptions struct {
	// Disk is the disk to resize, e.g. "scsi0". Required.
	Disk string `url:"disk"`
	// Size is the new size, e.g. "32G" for an absolute size or "+4G"
	// to grow by that amount. Shrinking is not supported by Proxmox.
	// Required.
	Size string `url:"size"`
	// Digest prevents the resize if the guest's configuration has
	// changed since the value was read.
	Digest string `url:"digest,omitempty"`
	// Skiplock bypasses the guest's configuration lock. Root only.
	Skiplock bool `url:"skiplock,omitempty"`
}

// Resize extends a guest disk's size via
// PUT /nodes/{node}/qemu/{vmid}/resize. Returns the resize task's UPID.
func (c *Client) Resize(ctx context.Context, node string, vmid int, opts *ResizeOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("qemu: resize options are required")
	}
	if opts.Disk == "" {
		return "", fmt.Errorf("qemu: resize requires a disk")
	}
	if opts.Size == "" {
		return "", fmt.Errorf("qemu: resize requires a size")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Update(ctx, "/nodes/"+node+"/qemu/"+strconv.Itoa(vmid)+"/resize", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
