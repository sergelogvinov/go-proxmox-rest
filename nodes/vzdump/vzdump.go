// Package vzdump provides access to the Proxmox VE per-node backup API
// (endpoints under /nodes/{node}/vzdump): creating an on-demand backup
// job, reading the node's configured backup defaults, and extracting a
// guest's configuration from an existing backup archive.
package vzdump

import (
	"context"
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the vzdump package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /nodes/{node}/vzdump resource tree,
// scoped to the node given to New.
type Client struct {
	client Getter
	node   string
}

// New returns a new vzdump client backed by the given root client, scoped
// to node.
func New(c Getter, node string) *Client {
	return &Client{client: c, node: node}
}

// Create starts an on-demand backup via POST /nodes/{node}/vzdump.
// opts.VMID or opts.All is required — Proxmox otherwise silently
// no-ops rather than erroring, so this validates it client-side.
// Returns the backup task's UPID.
func (c *Client) Create(ctx context.Context, opts *Options) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("vzdump: options are required")
	}
	if len(opts.VMID) == 0 && !opts.All {
		return "", fmt.Errorf("vzdump: either VMID or All is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+c.node+"/vzdump", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}

// Defaults retrieves the node's currently configured backup defaults
// via GET /nodes/{node}/vzdump/defaults. storage may be empty to use
// the node's default storage.
func (c *Client) Defaults(ctx context.Context, storage string) (*Options, error) {
	var p map[string]string
	if storage != "" {
		p = map[string]string{"storage": storage}
	}

	opts := &Options{}
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/vzdump/defaults", opts, p); err != nil {
		return nil, err
	}

	return opts, nil
}

// ExtractConfig extracts a guest's configuration from an existing
// backup archive via GET /nodes/{node}/vzdump/extractconfig. Returns
// the raw configuration text (a QEMU or LXC config file verbatim).
func (c *Client) ExtractConfig(ctx context.Context, volume string) (string, error) {
	var config string
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/vzdump/extractconfig", &config, map[string]string{"volume": volume}); err != nil {
		return "", err
	}

	return config, nil
}
