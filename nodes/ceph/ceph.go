// Package ceph provides access to the Proxmox VE per-node Ceph service
// API (endpoints under /nodes/{node}/ceph): the basic service-lifecycle
// operations — cluster status, start/stop/restart, the list of
// installable Ceph releases, and the Ceph log — folded directly onto
// Client rather than behind an accessor. The much larger
// cfg/osd/mds/mgr/mon/fs/pool/crush/rules/init/restart-bulk/cmd-safety
// subresources are left for a future addition.
package ceph

import (
	"context"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the ceph package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /nodes/{node}/ceph resource tree. Every
// method takes the target node's name as a call argument, since this
// package has no persistent per-node scope of its own.
type Client struct {
	client Getter
}

// New returns a new ceph client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// Status retrieves the Ceph cluster status via
// GET /nodes/{node}/ceph/status — the raw `ceph status` output, merged
// with `ceph health detail`. Proxmox's own description notes this is
// identical to Client.Cluster().Ceph().Status(ctx); this per-node path
// exists purely for operator convenience (e.g. scripting against a
// specific node), so the two share the same Status shape.
func (c *Client) Status(ctx context.Context, node string) (*Status, error) {
	status := &Status{}
	if err := c.client.Get(ctx, "/nodes/"+node+"/ceph/status", status, nil); err != nil {
		return nil, err
	}

	return status, nil
}

// Start starts Ceph services on the node via
// POST /nodes/{node}/ceph/start. service selects the scope: a bare kind
// ("ceph", "mon", "mds", "osd", "mgr" — see the Service constants) to
// affect every instance of that kind, "kind.instance" (e.g. "mon.pve1")
// to target one instance, or "" for Proxmox's own default ("ceph.target",
// every service). Returns the start task's UPID.
func (c *Client) Start(ctx context.Context, node string, service Service) (string, error) {
	return c.serviceCmd(ctx, node, "start", service)
}

// Stop stops Ceph services on the node via POST /nodes/{node}/ceph/stop.
// service has the same meaning as in Start. Returns the stop task's
// UPID.
func (c *Client) Stop(ctx context.Context, node string, service Service) (string, error) {
	return c.serviceCmd(ctx, node, "stop", service)
}

// Restart restarts Ceph services on the node via
// POST /nodes/{node}/ceph/restart. service has the same meaning as in
// Start. Returns the restart task's UPID.
func (c *Client) Restart(ctx context.Context, node string, service Service) (string, error) {
	return c.serviceCmd(ctx, node, "restart", service)
}

// serviceCmd POSTs to /nodes/{node}/ceph/{action}, optionally scoped to
// service, and returns the resulting task's UPID.
func (c *Client) serviceCmd(ctx context.Context, node, action string, service Service) (string, error) {
	var p map[string]string
	if service != "" {
		p = map[string]string{"service": string(service)}
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+node+"/ceph/"+action, &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}

// Releases lists every known Ceph release, marking which ones can be
// installed on the given node, via GET /nodes/{node}/ceph/releases.
func (c *Client) Releases(ctx context.Context, node string) ([]Release, error) {
	var releases []Release
	if err := c.client.Get(ctx, "/nodes/"+node+"/ceph/releases", &releases, nil); err != nil {
		return nil, err
	}

	return releases, nil
}

// Log retrieves the node's Ceph log via GET /nodes/{node}/ceph/log.
// opts may be nil to request Proxmox's default window (the first ~50
// lines).
func (c *Client) Log(ctx context.Context, node string, opts *LogOptions) ([]LogEntry, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var entries []LogEntry
	if err := c.client.Get(ctx, "/nodes/"+node+"/ceph/log", &entries, p); err != nil {
		return nil, err
	}

	return entries, nil
}
