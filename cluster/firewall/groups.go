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

package firewall

import (
	"context"
	"fmt"
)

// groupsResource provides access to the /cluster/firewall/groups resource:
// GET/POST /cluster/firewall/groups and DELETE
// /cluster/firewall/groups/{group}. A security group's own rules are
// accessed separately via Rules(name).
type groupsResource struct {
	client Getter
}

// Rules returns an accessor for the given security group's rules under
// /cluster/firewall/groups/{name}, reusing the same List/Get/Create/Update/
// Delete shape as the cluster-wide rules (Client.Rules()).
func (g *groupsResource) Rules(name string) *ruleResource {
	return &ruleResource{client: g.client, base: "/cluster/firewall/groups/" + name}
}

// List retrieves all security groups via GET /cluster/firewall/groups.
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Audit,match=all
func (g *groupsResource) List(ctx context.Context) ([]Group, error) {
	var groups []Group
	if err := g.client.Get(ctx, "/cluster/firewall/groups", &groups, nil); err != nil {
		return nil, err
	}

	return groups, nil
}

// Create creates a new, empty security group via
// POST /cluster/firewall/groups.
//
// Proxmox uses this same endpoint for both creation and rename/re-comment
// (see Update); Create leaves opts.Rename unset, which — being a plain
// string field — params.Encode naturally omits from the request, so
// Proxmox treats it as a creation.
//
// +proxmox:rbac:path=/,method=POST,privs=Sys.Modify,match=all
func (g *groupsResource) Create(ctx context.Context, opts *GroupOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: group options are required")
	}
	if opts.Name == "" {
		return fmt.Errorf("firewall: group name is required")
	}

	params, err := opts.encode()
	if err != nil {
		return err
	}

	return g.client.Create(ctx, "/cluster/firewall/groups", nil, params)
}

// Update renames and/or re-comments an existing security group. Proxmox has
// no dedicated update endpoint for a security group's own metadata: it
// reuses POST /cluster/firewall/groups, distinguishing a rename from a
// create by the presence of the "rename" parameter, which is set to the
// group's *current* name (Proxmox looks it up, then applies the "group" and
// "comment" parameters to it).
//
// name is the group's current name, addressing which group to update; it
// always overwrites whatever opts.Rename may hold. If opts.Name is left
// empty, the target name defaults to name — i.e. the group keeps its
// current name and only its comment changes. To rename the group, set
// opts.Name to the desired new name.
//
// +proxmox:rbac:path=/,method=POST,privs=Sys.Modify,match=all
func (g *groupsResource) Update(ctx context.Context, name string, opts *GroupOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: group options are required")
	}

	params, err := opts.encode()
	if err != nil {
		return err
	}

	params["rename"] = name
	if opts.Name == "" {
		params["group"] = name
	}

	return g.client.Create(ctx, "/cluster/firewall/groups", nil, params)
}

// Delete removes an (empty) security group via
// DELETE /cluster/firewall/groups/{name}.
//
// +proxmox:rbac:path=/,method=DELETE,privs=Sys.Modify,match=all
func (g *groupsResource) Delete(ctx context.Context, name string) error {
	return g.client.Delete(ctx, "/cluster/firewall/groups/"+name, nil, nil)
}
