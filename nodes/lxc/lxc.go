// Package lxc provides access to the Proxmox VE per-node LXC container
// API (endpoints under /nodes/{node}/lxc/{vmid}). This currently covers
// the status resource tree (current status and the
// start/stop/shutdown/reboot/suspend/resume power actions — LXC has no
// "reset" action, unlike QEMU), config (read/update the container's
// configuration), clone, and template — all folded directly onto Client
// rather than behind per-resource accessors, mirroring the nodes/qemu
// package's shape. The much larger snapshot/firewall/feature/... surface
// is left for a future addition.
package lxc

import (
	"context"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the lxc package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /nodes/{node}/lxc/{vmid} resource tree.
// Every method takes the target node's name and container's VMID as call
// arguments, since this package has no persistent per-container scope of
// its own.
type Client struct {
	client Getter
}

// New returns a new lxc client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}
