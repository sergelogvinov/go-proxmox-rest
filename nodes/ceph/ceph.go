/*
Copyright 2026 Proxmox Community.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

// Client provides access to the /nodes/{node}/ceph resource tree, scoped
// to the node given to New.
type Client struct {
	client Getter
	node   string
}

// New returns a new ceph client backed by the given root client, scoped
// to node.
func New(c Getter, node string) *Client {
	return &Client{client: c, node: node}
}

// Status retrieves the Ceph cluster status via
// GET /nodes/{node}/ceph/status — the raw `ceph status` output, merged
// with `ceph health detail`. Proxmox's own description notes this is
// identical to Client.Cluster().Ceph().Status(ctx); this per-node path
// exists purely for operator convenience (e.g. scripting against a
// specific node), so the two share the same Status shape.
func (c *Client) Status(ctx context.Context) (*Status, error) {
	status := &Status{}
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/ceph/status", status, nil); err != nil {
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
func (c *Client) Start(ctx context.Context, service Service) (string, error) {
	return c.serviceCmd(ctx, "start", service)
}

// Stop stops Ceph services on the node via POST /nodes/{node}/ceph/stop.
// service has the same meaning as in Start. Returns the stop task's
// UPID.
func (c *Client) Stop(ctx context.Context, service Service) (string, error) {
	return c.serviceCmd(ctx, "stop", service)
}

// Restart restarts Ceph services on the node via
// POST /nodes/{node}/ceph/restart. service has the same meaning as in
// Start. Returns the restart task's UPID.
func (c *Client) Restart(ctx context.Context, service Service) (string, error) {
	return c.serviceCmd(ctx, "restart", service)
}

// serviceCmd POSTs to /nodes/{node}/ceph/{action}, optionally scoped to
// service, and returns the resulting task's UPID.
func (c *Client) serviceCmd(ctx context.Context, action string, service Service) (string, error) {
	var p map[string]string
	if service != "" {
		p = map[string]string{"service": string(service)}
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+c.node+"/ceph/"+action, &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}

// Releases lists every known Ceph release, marking which ones can be
// installed on the node, via GET /nodes/{node}/ceph/releases.
func (c *Client) Releases(ctx context.Context) ([]Release, error) {
	var releases []Release
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/ceph/releases", &releases, nil); err != nil {
		return nil, err
	}

	return releases, nil
}

// Log retrieves the node's Ceph log via GET /nodes/{node}/ceph/log.
// opts may be nil to request Proxmox's default window (the first ~50
// lines).
func (c *Client) Log(ctx context.Context, opts *LogOptions) ([]LogEntry, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var entries []LogEntry
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/ceph/log", &entries, p); err != nil {
		return nil, err
	}

	return entries, nil
}
