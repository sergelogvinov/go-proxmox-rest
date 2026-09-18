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

// aliasesResource is the accessor for a guest's
// .../firewall/aliases resource, its IP/network aliases. Obtain it via
// Client.Aliases(vmid).
type aliasesResource struct {
	client Getter
	base   string
}

// List retrieves all aliases via GET /nodes/{node}/lxc/{vmid}/firewall/aliases.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (a *aliasesResource) List(ctx context.Context) ([]Alias, error) {
	var aliases []Alias
	if err := a.client.Get(ctx, a.base, &aliases, nil); err != nil {
		return nil, err
	}

	return aliases, nil
}

// Get retrieves a single alias via GET /nodes/{node}/lxc/{vmid}/firewall/aliases/{name}.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (a *aliasesResource) Get(ctx context.Context, name string) (*Alias, error) {
	var alias Alias
	if err := a.client.Get(ctx, a.base+"/"+name, &alias, nil); err != nil {
		return nil, err
	}

	return &alias, nil
}

// Create creates a new alias via POST /nodes/{node}/lxc/{vmid}/firewall/aliases.
//
// opts.Name and opts.CIDR are required.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Config.Network,match=all
func (a *aliasesResource) Create(ctx context.Context, opts *AliasOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: alias options are required")
	}
	if opts.Name == "" {
		return fmt.Errorf("firewall: alias name is required")
	}
	if opts.CIDR == "" {
		return fmt.Errorf("firewall: alias cidr is required")
	}

	params, err := opts.Encode()
	if err != nil {
		return err
	}

	return a.client.Create(ctx, a.base, nil, params)
}

// Update modifies an existing alias via PUT /nodes/{node}/lxc/{vmid}/firewall/aliases/{name}, addressed by
// its current name. Set opts.Rename to give the alias a new name.
//
// opts.CIDR is required — Proxmox requires it to be resent on every
// update even when unchanged.
//
// +proxmox:rbac:path=/vms/{vmid},method=PUT,privs=VM.Config.Network,match=all
func (a *aliasesResource) Update(ctx context.Context, name string, opts *AliasOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: alias options are required")
	}
	if opts.CIDR == "" {
		return fmt.Errorf("firewall: alias cidr is required")
	}

	params, err := opts.Encode()
	if err != nil {
		return err
	}

	return a.client.Update(ctx, a.base+"/"+name, nil, params)
}

// Delete removes an alias via DELETE /nodes/{node}/lxc/{vmid}/firewall/aliases/{name}.
//
// digest, when non-empty, guards against concurrent modification
// (value from the corresponding Get).
//
// +proxmox:rbac:path=/vms/{vmid},method=DELETE,privs=VM.Config.Network,match=all
func (a *aliasesResource) Delete(ctx context.Context, name string, digest string) error {
	var params map[string]string
	if digest != "" {
		params = map[string]string{"digest": digest}
	}

	return a.client.Delete(ctx, a.base+"/"+name, nil, params)
}
