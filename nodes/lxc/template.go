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
func (c *Client) Template(ctx context.Context, node string, vmid int) error {
	return c.client.Create(ctx, "/nodes/"+node+"/lxc/"+strconv.Itoa(vmid)+"/template", nil, nil)
}
