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

package ha

import (
	"context"
	"fmt"
)

// groupsResource provides access to GET/POST /cluster/ha/groups and
// GET/PUT/DELETE /cluster/ha/groups/{group}.
type groupsResource struct {
	client Getter
}

// Groups returns an accessor for the /cluster/ha/groups resource.
func (c *Client) Groups() *groupsResource {
	return &groupsResource{client: c.client}
}

// Get retrieves a single HA group via GET /cluster/ha/groups/{group}.
func (g *groupsResource) Get(ctx context.Context, group string) (*Group, error) {
	var hg Group
	if err := g.client.Get(ctx, "/cluster/ha/groups/"+group, &hg, nil); err != nil {
		return nil, err
	}

	return &hg, nil
}

// List retrieves all HA groups via GET /cluster/ha/groups.
func (g *groupsResource) List(ctx context.Context) ([]Group, error) {
	var groups []Group
	if err := g.client.Get(ctx, "/cluster/ha/groups", &groups, nil); err != nil {
		return nil, err
	}

	return groups, nil
}

// Create creates a new HA group via POST /cluster/ha/groups.
func (g *groupsResource) Create(ctx context.Context, opts *GroupOptions) (*Group, error) {
	if opts == nil {
		return nil, fmt.Errorf("ha: group options are required")
	}
	if opts.ID == "" {
		return nil, fmt.Errorf("ha: group id is required")
	}
	if opts.Nodes == nil {
		return nil, fmt.Errorf("ha: group nodes are required")
	}

	params, err := opts.encode()
	if err != nil {
		return nil, err
	}

	var hg Group
	if err := g.client.Create(ctx, "/cluster/ha/groups", &hg, params); err != nil {
		return nil, err
	}

	return &hg, nil
}

// Update modifies an existing HA group via PUT /cluster/ha/groups/{group}.
func (g *groupsResource) Update(ctx context.Context, group string, opts *GroupOptions) (*Group, error) {
	params, err := opts.encode()
	if err != nil {
		return nil, err
	}

	var hg Group
	if err := g.client.Update(ctx, "/cluster/ha/groups/"+group, &hg, params); err != nil {
		return nil, err
	}

	return &hg, nil
}

// Delete removes an HA group via DELETE /cluster/ha/groups/{group}.
func (g *groupsResource) Delete(ctx context.Context, group string) error {
	return g.client.Delete(ctx, "/cluster/ha/groups/"+group, nil, nil)
}
