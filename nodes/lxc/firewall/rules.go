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

// ruleResource provides access to a guest's firewall rule list:
// GET/POST {base} and GET/PUT/DELETE {base}/{pos}. base is
// .../firewall/rules for the guest identified by the guest Client.Rules
// was called with.
type ruleResource struct {
	client Getter
	base   string
}

// List retrieves rules via GET /nodes/{node}/lxc/{vmid}/firewall/rules.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (r *ruleResource) List(ctx context.Context) ([]Rule, error) {
	var rules []Rule
	if err := r.client.Get(ctx, r.base, &rules, nil); err != nil {
		return nil, err
	}

	return rules, nil
}

// Get retrieves a rule via GET /nodes/{node}/lxc/{vmid}/firewall/rules/{pos}.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (r *ruleResource) Get(ctx context.Context, pos int) (*Rule, error) {
	var rule Rule
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
// POST /nodes/{node}/lxc/{vmid}/firewall/rules.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Config.Network,match=all
func (r *ruleResource) Create(ctx context.Context, opts *RuleOptions) error {
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
// PUT /nodes/{node}/lxc/{vmid}/firewall/rules/{pos}.
//
// +proxmox:rbac:path=/vms/{vmid},method=PUT,privs=VM.Config.Network,match=all
func (r *ruleResource) Update(ctx context.Context, pos int, opts *RuleOptions) error {
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

// Delete removes a rule via DELETE {base}/{pos}. digest, when
// non-empty, guards against deleting a rule that has changed since it
// was last read.
// DELETE /nodes/{node}/lxc/{vmid}/firewall/rules/{pos}.
//
// +proxmox:rbac:path=/vms/{vmid},method=DELETE,privs=VM.Config.Network,match=all
func (r *ruleResource) Delete(ctx context.Context, pos int, digest string) error {
	var params map[string]string
	if digest != "" {
		params = map[string]string{"digest": digest}
	}

	return r.client.Delete(ctx, fmt.Sprintf("%s/%d", r.base, pos), nil, params)
}
