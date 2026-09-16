// Package qemu provides access to the Proxmox VE per-node QEMU guest API
// (endpoints under /nodes/{node}/qemu/{vmid}). This currently covers the
// status resource tree (current status and the
// start/stop/reset/shutdown/reboot/suspend/resume power actions),
// config (read/update the guest's configuration), clone, template,
// migrate (precondition check and the migration task itself), feature
// (capability checks), move_disk, resize, unlink, snapshot (behind the
// Snapshot() accessor), and the QEMU Guest Agent (nodes/qemu/agent,
// behind the Agent() accessor — large enough to earn its own
// subpackage, unlike the rest of this package's flat, direct-on-Client
// shape). The much larger firewall/... surface is left for a future
// addition.
package qemu

import (
	"context"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu/agent"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the qemu package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
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

// Agent returns an accessor for the
// /nodes/{node}/qemu/{vmid}/agent resource tree: the QEMU Guest Agent
// command surface (filesystem freeze/trim, network/user/OS info,
// exec, file read/write, ...).
func (c *Client) Agent() *agent.Client {
	return agent.New(c.client)
}

// Snapshot returns an accessor for the
// /nodes/{node}/qemu/{vmid}/snapshot resource tree: listing, creating,
// and deleting snapshots, reading/updating a snapshot's metadata, and
// rolling back to one.
func (c *Client) Snapshot() *snapshotResource {
	return &snapshotResource{client: c.client}
}
