// Package qemu provides access to the Proxmox VE per-node QEMU guest API
// (endpoints under /nodes/{node}/qemu/{vmid}). This currently covers only
// the status resource tree (endpoints under .../status): the guest's
// current status, and its start/stop/reset/shutdown/reboot/suspend/resume
// power actions — folded directly onto Client rather than behind a
// separate accessor, since status is (so far) this package's only
// resource. The much larger config/clone/migrate/snapshot/... surface is
// left for a future addition.
package qemu

import (
	"context"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the qemu package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /nodes/{node}/qemu/{vmid} resource tree.
// Every method takes the target node's name and guest's VMID as call
// arguments, since this package has no persistent per-guest scope of its
// own.
type Client struct {
	client Getter
}

// New returns a new qemu client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}
