// Package hardware provides access to the Proxmox VE per-node hardware
// detection API (endpoints under /nodes/{node}/hardware), local PCI and USB
// device inventories.
package hardware

import (
	"context"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the hardware package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /nodes/{node}/hardware resource tree,
// scoped to the node given to New.
type Client struct {
	client Getter
	node   string
}

// New returns a new hardware client backed by the given root client,
// scoped to node.
func New(c Getter, node string) *Client {
	return &Client{client: c, node: node}
}

// PCI returns an accessor for the /nodes/{node}/hardware/pci resource tree,
// the node's local PCI device inventory.
func (c *Client) PCI() *pciResource {
	return &pciResource{client: c.client, node: c.node}
}

// USB returns an accessor for the /nodes/{node}/hardware/usb resource tree,
// the node's local USB device inventory.
func (c *Client) USB() *usbResource {
	return &usbResource{client: c.client, node: c.node}
}
