package cluster

import (
	"context"
	"fmt"
)

// haResource provides access to the /cluster/ha resource tree, the
// cluster-wide high-availability configuration.
type haResource struct {
	client Getter
}

// HA returns an accessor for the /cluster/ha resource tree.
func (c *Client) HA() *haResource {
	return &haResource{client: c.client}
}

// haGroupsResource provides access to GET/POST /cluster/ha/groups and
// GET/PUT/DELETE /cluster/ha/groups/{group}.
type haGroupsResource struct {
	client Getter
}

// Groups returns an accessor for the /cluster/ha/groups resource.
func (h *haResource) Groups() *haGroupsResource {
	return &haGroupsResource{client: h.client}
}

// Get retrieves a single HA group via GET /cluster/ha/groups/{group}.
func (g *haGroupsResource) Get(ctx context.Context, group string) (*HAGroup, error) {
	var hg HAGroup
	if err := g.client.Get(ctx, "/cluster/ha/groups/"+group, &hg, nil); err != nil {
		return nil, err
	}

	return &hg, nil
}

// List retrieves all HA groups via GET /cluster/ha/groups.
func (g *haGroupsResource) List(ctx context.Context) ([]HAGroup, error) {
	var groups []HAGroup
	if err := g.client.Get(ctx, "/cluster/ha/groups", &groups, nil); err != nil {
		return nil, err
	}

	return groups, nil
}

// Create creates a new HA group via POST /cluster/ha/groups.
func (g *haGroupsResource) Create(ctx context.Context, opts *HAGroupOptions) (*HAGroup, error) {
	if opts == nil {
		return nil, fmt.Errorf("cluster: ha group options are required")
	}
	if opts.ID == "" {
		return nil, fmt.Errorf("cluster: ha group id is required")
	}
	if opts.Nodes == nil {
		return nil, fmt.Errorf("cluster: ha group nodes are required")
	}

	params, err := opts.encode()
	if err != nil {
		return nil, err
	}

	var hg HAGroup
	if err := g.client.Create(ctx, "/cluster/ha/groups", &hg, params); err != nil {
		return nil, err
	}

	return &hg, nil
}

// Update modifies an existing HA group via PUT /cluster/ha/groups/{group}.
func (g *haGroupsResource) Update(ctx context.Context, group string, opts *HAGroupOptions) (*HAGroup, error) {
	params, err := opts.encode()
	if err != nil {
		return nil, err
	}

	var hg HAGroup
	if err := g.client.Update(ctx, "/cluster/ha/groups/"+group, &hg, params); err != nil {
		return nil, err
	}

	return &hg, nil
}

// Delete removes an HA group via DELETE /cluster/ha/groups/{group}.
func (g *haGroupsResource) Delete(ctx context.Context, group string) error {
	return g.client.Delete(ctx, "/cluster/ha/groups/"+group, nil, nil)
}
