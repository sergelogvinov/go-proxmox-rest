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

// Package capabilities provides access to the Proxmox VE per-node
// capability-discovery API (endpoints under /nodes/{node}/capabilities):
// available QEMU CPU models/flags, machine types, and migration
// capabilities.
package capabilities

import (
	"context"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the capabilities package
// decoupled from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /nodes/{node}/capabilities resource tree,
// scoped to the node given to New.
type Client struct {
	client Getter
	node   string
}

// New returns a new capabilities client backed by the given root client,
// scoped to node.
func New(c Getter, node string) *Client {
	return &Client{client: c, node: node}
}

// Qemu returns an accessor for the /nodes/{node}/capabilities/qemu
// resource tree.
func (c *Client) Qemu() *qemuResource {
	return &qemuResource{client: c.client, node: c.node}
}
