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

// List retrieves all aliases via GET {base}.
func (a *aliasesResource) List(ctx context.Context) ([]Alias, error) {
	var aliases []Alias
	if err := a.client.Get(ctx, a.base, &aliases, nil); err != nil {
		return nil, err
	}

	return aliases, nil
}

// Get retrieves a single alias via GET {base}/{name}.
func (a *aliasesResource) Get(ctx context.Context, name string) (*Alias, error) {
	var alias Alias
	if err := a.client.Get(ctx, a.base+"/"+name, &alias, nil); err != nil {
		return nil, err
	}

	return &alias, nil
}

// Create creates a new alias via POST {base}.
//
// opts.Name and opts.CIDR are required.
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

// Update modifies an existing alias via PUT {base}/{name}, addressed by
// its current name. Set opts.Rename to give the alias a new name.
//
// opts.CIDR is required — Proxmox requires it to be resent on every
// update even when unchanged.
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

// Delete removes an alias via DELETE {base}/{name}.
//
// digest, when non-empty, guards against concurrent modification
// (value from the corresponding Get).
func (a *aliasesResource) Delete(ctx context.Context, name string, digest string) error {
	var params map[string]string
	if digest != "" {
		params = map[string]string{"digest": digest}
	}

	return a.client.Delete(ctx, a.base+"/"+name, nil, params)
}
