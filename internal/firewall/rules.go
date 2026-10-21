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

	"github.com/sergelogvinov/go-proxmox-rest/types"
)

// Rules provides access to a firewall rule list: GET/POST {base} and
// GET/PUT/DELETE {base}/{pos}. base is a .../firewall/rules path, or
// /cluster/firewall/groups/{name} for a security group's rules.
type Rules struct {
	client Getter
	base   string
}

// NewRules returns the rule list rooted at base.
func NewRules(c Getter, base string) *Rules {
	return &Rules{client: c, base: base}
}

// List retrieves all rules via GET {base}.
func (r *Rules) List(ctx context.Context) ([]types.Rule, error) {
	var rules []types.Rule
	if err := r.client.Get(ctx, r.base, &rules, nil); err != nil {
		return nil, err
	}

	return rules, nil
}

// Get retrieves the rule at pos via GET {base}/{pos}.
func (r *Rules) Get(ctx context.Context, pos int) (*types.Rule, error) {
	var rule types.Rule
	if err := r.client.Get(ctx, fmt.Sprintf("%s/%d", r.base, pos), &rule, nil); err != nil {
		return nil, err
	}

	return &rule, nil
}

// Create creates a new rule via POST {base}.
//
// Proxmox does not echo the created rule back (the response is null
// data) and does not report the position it was assigned. The new rule
// is appended at the end of the list unless opts.Pos is set to insert
// it elsewhere; call List afterwards to find it.
func (r *Rules) Create(ctx context.Context, opts *types.RuleOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: rule options are required")
	}

	if opts.Type == "" {
		return fmt.Errorf("firewall: rule type is required")
	}

	if opts.Action == "" {
		return fmt.Errorf("firewall: rule action is required")
	}

	params, err := opts.Encode()
	if err != nil {
		return err
	}

	return r.client.Create(ctx, r.base, nil, params)
}

// Update modifies an existing rule via PUT {base}/{pos}.
//
// Proxmox requires Type and Action to be resent on every update, even
// fields that are not changing. To reposition a rule instead of
// changing its fields, set opts.MoveTo — Proxmox ignores every other
// field in that case.
func (r *Rules) Update(ctx context.Context, pos int, opts *types.RuleOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: rule options are required")
	}

	if opts.MoveTo == nil {
		if opts.Type == "" {
			return fmt.Errorf("firewall: rule type is required")
		}

		if opts.Action == "" {
			return fmt.Errorf("firewall: rule action is required")
		}
	}

	params, err := opts.Encode()
	if err != nil {
		return err
	}

	return r.client.Update(ctx, fmt.Sprintf("%s/%d", r.base, pos), nil, params)
}

// Delete removes the rule at pos via DELETE {base}/{pos}. digest, when
// non-empty, guards against deleting a rule that has changed since it
// was last read.
func (r *Rules) Delete(ctx context.Context, pos int, digest string) error {
	var params map[string]string
	if digest != "" {
		params = map[string]string{"digest": digest}
	}

	return r.client.Delete(ctx, fmt.Sprintf("%s/%d", r.base, pos), nil, params)
}
