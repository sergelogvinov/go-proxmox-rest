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
func (c *Client) Interfaces(ctx context.Context, node string, vmid int) ([]Interface, error) {
	var ifaces []Interface
	if err := c.client.Get(ctx, "/nodes/"+node+"/lxc/"+strconv.Itoa(vmid)+"/interfaces", &ifaces, nil); err != nil {
		return nil, err
	}

	return ifaces, nil
}
