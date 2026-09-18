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

// Package network provides access to the Proxmox VE per-node network
// configuration API (endpoints under /nodes/{node}/network).
package network

import (
	"context"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the network package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /nodes/{node}/network resource tree,
// scoped to the node given to New.
type Client struct {
	client Getter
	node   string
}

// New returns a new network client backed by the given root client,
// scoped to node.
func New(c Getter, node string) *Client {
	return &Client{client: c, node: node}
}

// List retrieves the node's network interfaces via GET /nodes/{node}/network.
// typeFilter narrows the result to interfaces of that type (or, for
// TypeAnyBridge/TypeAnyLocalBridge/TypeIncludeSDN, Proxmox's corresponding
// pseudo-filter); an empty typeFilter returns every interface (the
// loopback device excluded).
// This user=>all endpoint has no fixed privilege. Bridge, SDN vnet, and fabric
// results are conditionally filtered by SDN access as described above.
func (c *Client) List(ctx context.Context, typeFilter Type) ([]Interface, error) {
	var p map[string]string
	if typeFilter != "" {
		p = map[string]string{"type": string(typeFilter)}
	}

	var interfaces []Interface
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/network", &interfaces, p); err != nil {
		return nil, err
	}

	return interfaces, nil
}

// Get retrieves a single network interface's configuration via
// GET /nodes/{node}/network/{iface}.
//
// +proxmox:rbac:path=/nodes/{node},method=GET,privs=Sys.Audit,match=all
func (c *Client) Get(ctx context.Context, iface string) (*Interface, error) {
	ifc := &Interface{}
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/network/"+iface, ifc, nil); err != nil {
		return nil, err
	}

	// network_config's return schema does not include the interface's
	// own name (only List's index handler injects it); fill it in from
	// the requested iface, matching the mapping package's Get fix for
	// the same kind of omission.
	if ifc.Iface == "" {
		ifc.Iface = iface
	}

	return ifc, nil
}

// Create creates a new network interface configuration via
// POST /nodes/{node}/network. The interface is only applied after a
// Reload (or a manual "ifreload -a"/reboot).
//
// +proxmox:rbac:path=/nodes/{node},method=POST,privs=Sys.Modify,match=all
func (c *Client) Create(ctx context.Context, iface string, opts *InterfaceOptions) error {
	p, err := opts.encode()
	if err != nil {
		return err
	}
	// Proxmox's create_network schema has no "delete" property at all
	// (additionalProperties => 0 rejects it outright), unlike
	// update_network.
	delete(p, "delete")
	p["iface"] = iface

	return c.client.Create(ctx, "/nodes/"+c.node+"/network", nil, p)
}

// Update modifies an existing network interface configuration via
// PUT /nodes/{node}/network/{iface}. The change is only applied after a
// Reload (or a manual "ifreload -a"/reboot).
//
// +proxmox:rbac:path=/nodes/{node},method=PUT,privs=Sys.Modify,match=all
func (c *Client) Update(ctx context.Context, iface string, opts *InterfaceOptions) error {
	p, err := opts.encode()
	if err != nil {
		return err
	}
	p["iface"] = iface

	return c.client.Update(ctx, "/nodes/"+c.node+"/network/"+iface, nil, p)
}

// Delete removes a network interface configuration via
// DELETE /nodes/{node}/network/{iface}. The change is only applied after a
// Reload (or a manual "ifreload -a"/reboot).
//
// +proxmox:rbac:path=/nodes/{node},method=DELETE,privs=Sys.Modify,match=all
func (c *Client) Delete(ctx context.Context, iface string) error {
	return c.client.Delete(ctx, "/nodes/"+c.node+"/network/"+iface, nil, nil)
}

// RevertChanges discards uncommitted network configuration changes via
// DELETE /nodes/{node}/network, removing /etc/network/interfaces.new so
// the next Get/List reflects only the currently applied configuration.
//
// +proxmox:rbac:path=/nodes/{node},method=DELETE,privs=Sys.Modify,match=all
func (c *Client) RevertChanges(ctx context.Context) error {
	return c.client.Delete(ctx, "/nodes/"+c.node+"/network", nil, nil)
}

// Reload applies pending network configuration changes via
// PUT /nodes/{node}/network ("ifreload -a"), returning the UPID of the
// background task. regenerateFRR, when non-nil, overrides whether FRR
// (routing daemon) configuration is regenerated as part of the reload;
// nil lets Proxmox decide based on whether the SDN config uses FRR.
//
// Requires ifupdown2 to be installed on the node; Proxmox returns an error
// otherwise.
//
// +proxmox:rbac:path=/nodes/{node},method=PUT,privs=Sys.Modify,match=all
func (c *Client) Reload(ctx context.Context, regenerateFRR *bool) (string, error) {
	var p map[string]string
	if regenerateFRR != nil {
		p = map[string]string{"regenerate-frr": "0"}
		if *regenerateFRR {
			p["regenerate-frr"] = "1"
		}
	}

	var upid string
	if err := c.client.Update(ctx, "/nodes/"+c.node+"/network", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
