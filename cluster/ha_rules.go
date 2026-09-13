package cluster

import (
	"context"
	"fmt"
)

// haRulesResource provides access to GET/POST /cluster/ha/rules and
// GET/PUT/DELETE /cluster/ha/rules/{rule}.
type haRulesResource struct {
	client Getter
}

// Rules returns an accessor for the /cluster/ha/rules resource.
func (h *haResource) Rules() *haRulesResource {
	return &haRulesResource{client: h.client}
}

// Get retrieves a single HA rule via GET /cluster/ha/rules/{rule}.
func (r *haRulesResource) Get(ctx context.Context, rule string) (*HARule, error) {
	var hr HARule
	if err := r.client.Get(ctx, "/cluster/ha/rules/"+rule, &hr, nil); err != nil {
		return nil, err
	}

	return &hr, nil
}

// List retrieves HA rules via GET /cluster/ha/rules, optionally narrowed by
// rule type and/or the HA resource ID they affect. An empty ruleType or
// resource means unfiltered along that dimension.
func (r *haRulesResource) List(ctx context.Context, ruleType HARuleType, resource string) ([]HARule, error) {
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

	var rules []HARule
	if err := r.client.Get(ctx, "/cluster/ha/rules", &rules, params); err != nil {
		return nil, err
	}

	return rules, nil
}

// Create creates a new HA rule via POST /cluster/ha/rules.
func (r *haRulesResource) Create(ctx context.Context, opts *HARuleOptions) (*HARule, error) {
	if opts == nil {
		return nil, fmt.Errorf("cluster: ha rule options are required")
	}
	if opts.ID == "" {
		return nil, fmt.Errorf("cluster: ha rule id is required")
	}
	if opts.Type == "" {
		return nil, fmt.Errorf("cluster: ha rule type is required")
	}
	if len(opts.Resources) == 0 {
		return nil, fmt.Errorf("cluster: ha rule resources are required")
	}

	params, err := opts.encode()
	if err != nil {
		return nil, err
	}

	var hr HARule
	if err := r.client.Create(ctx, "/cluster/ha/rules", &hr, params); err != nil {
		return nil, err
	}

	return &hr, nil
}

// Update modifies an existing HA rule via PUT /cluster/ha/rules/{rule}.
// Proxmox requires Type to be resent on every update, even though the rule
// type itself cannot change.
func (r *haRulesResource) Update(ctx context.Context, rule string, opts *HARuleOptions) (*HARule, error) {
	if opts == nil {
		return nil, fmt.Errorf("cluster: ha rule options are required")
	}
	if opts.Type == "" {
		return nil, fmt.Errorf("cluster: ha rule type is required")
	}

	params, err := opts.encode()
	if err != nil {
		return nil, err
	}

	var hr HARule
	if err := r.client.Update(ctx, "/cluster/ha/rules/"+rule, &hr, params); err != nil {
		return nil, err
	}

	return &hr, nil
}

// Delete removes an HA rule via DELETE /cluster/ha/rules/{rule}.
func (r *haRulesResource) Delete(ctx context.Context, rule string) error {
	return r.client.Delete(ctx, "/cluster/ha/rules/"+rule, nil, nil)
}
