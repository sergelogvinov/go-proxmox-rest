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
