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

package cluster

import (
	"context"
)

// ResourceType filters GET /cluster/resources to a single kind of entry.
type ResourceType string

const (
	// ResourceTypeVM matches guests (QEMU VMs and LXC containers).
	ResourceTypeVM ResourceType = "vm"
	// ResourceTypeStorage matches storages.
	ResourceTypeStorage ResourceType = "storage"
	// ResourceTypeNode matches cluster nodes.
	ResourceTypeNode ResourceType = "node"
	// ResourceTypeSDN matches SDN objects.
	ResourceTypeSDN ResourceType = "sdn"
)

// resourcesResource provides access to GET /cluster/resources, the
// cluster-wide inventory of nodes, storages, and guests.
type resourcesResource struct {
	client Getter
}

// Resources returns an accessor for GET /cluster/resources.
func (c *Client) Resources() *resourcesResource {
	return &resourcesResource{client: c.client}
}

// ListFilter filters the entries returned by GET /cluster/resources. Type
// is sent to the API as the "type" query parameter; every other field is
// applied client-side after the response is decoded, since Proxmox has no
// server-side support for them.
type ListFilter struct {
	// Type restricts the resource kind fetched from the API: "vm",
	// "storage", "node", or "sdn". Empty fetches every kind.
	Type ResourceType

	// GuestType restricts vm entries to a guest kind, "qemu" or "lxc".
	// Empty disables the filter. Ignored for non-vm entries.
	GuestType string

	// VMID restricts vm entries to a specific guest ID. Zero disables the
	// filter.
	VMID int

	// Node restricts entries to a specific node. Empty disables the
	// filter.
	Node string

	// SkipTemplates excludes vm entries flagged as templates
	// (Template == 1).
	SkipTemplates bool

	// Match, when set, is evaluated last for each entry that passed the
	// filters above; the entry is kept only if Match returns true. Use it
	// for conditions List can't express directly, such as a check that
	// requires another API call. An error return aborts List, and that
	// error is returned to the caller.
	Match func(*Resource) (bool, error)
}

// List executes GET /cluster/resources and applies filter to the decoded
// results.
func (r *resourcesResource) List(ctx context.Context, filter ListFilter) ([]Resource, error) {
	var params map[string]string
	if filter.Type != "" {
		params = map[string]string{"type": string(filter.Type)}
	}

	var resources []Resource
	if err := r.client.Get(ctx, "/cluster/resources", &resources, params); err != nil {
		return nil, err
	}

	out := make([]Resource, 0, len(resources))

	for i := range resources {
		rs := &resources[i]

		if filter.GuestType != "" && rs.Type != filter.GuestType {
			continue
		}

		if filter.VMID != 0 && rs.VMID != filter.VMID {
			continue
		}

		if filter.Node != "" && rs.Node != filter.Node {
			continue
		}

		if filter.SkipTemplates && rs.Template == 1 {
			continue
		}

		if filter.Match != nil {
			ok, err := filter.Match(rs)
			if err != nil {
				return nil, err
			}

			if !ok {
				continue
			}
		}

		out = append(out, *rs)
	}

	return out, nil
}
