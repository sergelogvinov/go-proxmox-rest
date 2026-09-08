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

// Get executes GET /cluster/resources, optionally filtered by resType. An
// empty ResourceType returns every resource type.
func (r *resourcesResource) Get(ctx context.Context, resType ResourceType) ([]Resource, error) {
	var params map[string]string
	if resType != "" {
		params = map[string]string{"type": string(resType)}
	}

	var resources []Resource
	if err := r.client.Get(ctx, "/cluster/resources", &resources, params); err != nil {
		return nil, err
	}

	return resources, nil
}
