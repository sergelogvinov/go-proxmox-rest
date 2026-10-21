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

// resourcesResource provides access to GET/POST /cluster/ha/resources and
// GET/PUT/DELETE /cluster/ha/resources/{sid}.
type resourcesResource struct {
	client Getter
}

// Resources returns an accessor for the /cluster/ha/resources resource:
// the guests placed under HA management.
func (c *Client) Resources() *resourcesResource {
	return &resourcesResource{client: c.client}
}

// Get retrieves a single HA resource via GET /cluster/ha/resources/{sid}.
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Audit,match=all
func (r *resourcesResource) Get(ctx context.Context, sid string) (*Resource, error) {
	var hr Resource
	if err := r.client.Get(ctx, "/cluster/ha/resources/"+sid, &hr, nil); err != nil {
		return nil, err
	}

	return &hr, nil
}

// List retrieves HA resources via GET /cluster/ha/resources, optionally
// narrowed to one resource type. An empty resourceType means unfiltered.
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Audit,match=all
func (r *resourcesResource) List(ctx context.Context, resourceType ResourceType) ([]Resource, error) {
	var params map[string]string
	if resourceType != "" {
		params = map[string]string{"type": string(resourceType)}
	}

	var resources []Resource
	if err := r.client.Get(ctx, "/cluster/ha/resources", &resources, params); err != nil {
		return nil, err
	}

	return resources, nil
}

// Create places a guest under HA management via POST /cluster/ha/resources.
// Proxmox requires the referenced guest to exist.
//
// +proxmox:rbac:path=/,method=POST,privs=Sys.Console,match=all
func (r *resourcesResource) Create(ctx context.Context, opts *ResourceOptions) error {
	if opts == nil {
		return fmt.Errorf("ha: resource options are required")
	}
	if opts.SID == "" {
		return fmt.Errorf("ha: resource sid is required")
	}

	params, err := opts.encode()
	if err != nil {
		return err
	}

	return r.client.Create(ctx, "/cluster/ha/resources", nil, params)
}

// Update modifies an existing HA resource via
// PUT /cluster/ha/resources/{sid}.
//
// +proxmox:rbac:path=/,method=PUT,privs=Sys.Console,match=all
func (r *resourcesResource) Update(ctx context.Context, sid string, opts *ResourceOptions) error {
	if opts == nil {
		return fmt.Errorf("ha: resource options are required")
	}

	params, err := opts.encode()
	if err != nil {
		return err
	}

	return r.client.Update(ctx, "/cluster/ha/resources/"+sid, nil, params)
}

// Delete removes a guest from HA management via
// DELETE /cluster/ha/resources/{sid}. The guest itself is left untouched.
//
// purge also removes sid from every HA rule that references it, deleting
// any rule left with no resources. It is always sent explicitly because
// Proxmox defaults it to enabled.
//
// +proxmox:rbac:path=/,method=DELETE,privs=Sys.Console,match=all
func (r *resourcesResource) Delete(ctx context.Context, sid string, purge bool) error {
	p := "0"
	if purge {
		p = "1"
	}

	return r.client.Delete(ctx, "/cluster/ha/resources/"+sid, nil, map[string]string{"purge": p})
}
