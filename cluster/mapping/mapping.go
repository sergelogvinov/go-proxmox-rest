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

// Package mapping provides access to the Proxmox VE cluster-wide hardware
// mapping API (endpoints under /cluster/mapping): PCI, USB and directory
// mappings that let a guest reference a logical device name instead of a
// node-specific path, resolved per node by the mapping's entries.
package mapping

import (
	"context"
	"net/url"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the mapping package decoupled
// from the root package (no import cycle).
//
// CreateValues/UpdateValues take url.Values rather than Create/Update's
// map[string]string: every mapping resource's "map" parameter is a genuine
// Proxmox "array" (not a `-list`-formatted comma-joined string), and its
// entries are themselves comma-bearing property strings, so the array must
// be sent as repeated same-named query parameters rather than a single
// comma-joined value. See resource.encodeMap in this package and
// proxmox.Client.doValues's doc comment for the full rationale.
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
	CreateValues(ctx context.Context, path string, out any, params url.Values) error
	UpdateValues(ctx context.Context, path string, out any, params url.Values) error
}

// Client provides access to the /cluster/mapping resource tree.
type Client struct {
	client Getter
}

// New returns a new mapping client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// PCI returns an accessor for the /cluster/mapping/pci resource, PCI
// hardware mappings.
func (c *Client) PCI() *pciResource {
	return &pciResource{client: c.client}
}

// USB returns an accessor for the /cluster/mapping/usb resource, USB
// hardware mappings.
func (c *Client) USB() *usbResource {
	return &usbResource{client: c.client}
}

// Dir returns an accessor for the /cluster/mapping/dir resource, directory
// mappings (used to share a host directory with a container).
func (c *Client) Dir() *dirResource {
	return &dirResource{client: c.client}
}
