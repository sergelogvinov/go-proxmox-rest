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

// rulesResource provides access to GET/POST /cluster/ha/rules and
// GET/PUT/DELETE /cluster/ha/rules/{rule}.
type rulesResource struct {
	client Getter
}

// Rules returns an accessor for the /cluster/ha/rules resource.
func (c *Client) Rules() *rulesResource {
	return &rulesResource{client: c.client}
}

// Get retrieves a single HA rule via GET /cluster/ha/rules/{rule}.
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Audit,match=all
func (r *rulesResource) Get(ctx context.Context, rule string) (*Rule, error) {
	var hr Rule
	if err := r.client.Get(ctx, "/cluster/ha/rules/"+rule, &hr, nil); err != nil {
		return nil, err
	}

	return &hr, nil
}

// List retrieves HA rules via GET /cluster/ha/rules, optionally narrowed by
// rule type and/or the HA resource ID they affect. An empty ruleType or
// resource means unfiltered along that dimension.
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Audit,match=all
func (r *rulesResource) List(ctx context.Context, ruleType RuleType, resource string) ([]Rule, error) {
	var params map[string]string
	if ruleType != "" || resource != "" {
		params = map[string]string{}
		if ruleType != "" {
			params["type"] = string(ruleType)
		}
		if resource != "" {
			params["resource"] = resource
		}
	}

	var rules []Rule
	if err := r.client.Get(ctx, "/cluster/ha/rules", &rules, params); err != nil {
		return nil, err
	}

	return rules, nil
}

// Create creates a new HA rule via POST /cluster/ha/rules.
//
// +proxmox:rbac:path=/,method=POST,privs=Sys.Modify,match=all
func (r *rulesResource) Create(ctx context.Context, opts *RuleOptions) (*Rule, error) {
	if opts == nil {
		return nil, fmt.Errorf("ha: rule options are required")
	}
	if opts.ID == "" {
		return nil, fmt.Errorf("ha: rule id is required")
	}
	if opts.Type == "" {
		return nil, fmt.Errorf("ha: rule type is required")
	}
	if len(opts.Resources) == 0 {
		return nil, fmt.Errorf("ha: rule resources are required")
	}

	params, err := opts.encode()
	if err != nil {
		return nil, err
	}

	var hr Rule
	if err := r.client.Create(ctx, "/cluster/ha/rules", &hr, params); err != nil {
		return nil, err
	}

	return &hr, nil
}

// Update modifies an existing HA rule via PUT /cluster/ha/rules/{rule}.
// Proxmox requires Type to be resent on every update, even though the rule
// type itself cannot change.
//
// +proxmox:rbac:path=/,method=PUT,privs=Sys.Modify,match=all
func (r *rulesResource) Update(ctx context.Context, rule string, opts *RuleOptions) (*Rule, error) {
	if opts == nil {
		return nil, fmt.Errorf("ha: rule options are required")
	}
	if opts.Type == "" {
		return nil, fmt.Errorf("ha: rule type is required")
	}

	params, err := opts.encode()
	if err != nil {
		return nil, err
	}

	var hr Rule
	if err := r.client.Update(ctx, "/cluster/ha/rules/"+rule, &hr, params); err != nil {
		return nil, err
	}

	return &hr, nil
}

// Delete removes an HA rule via DELETE /cluster/ha/rules/{rule}.
//
// +proxmox:rbac:path=/,method=DELETE,privs=Sys.Modify,match=all
func (r *rulesResource) Delete(ctx context.Context, rule string) error {
	return r.client.Delete(ctx, "/cluster/ha/rules/"+rule, nil, nil)
}
