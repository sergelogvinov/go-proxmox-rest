package firewall

import (
	"context"
	"fmt"
)

// ipsetResource provides access to GET/POST /cluster/firewall/ipset and
// DELETE /cluster/firewall/ipset/{name}, the cluster-wide IP set index. A
// given set's members are reached via Entries(name).
type ipsetResource struct {
	client Getter
}

// List retrieves all IP sets via GET /cluster/firewall/ipset.
func (s *ipsetResource) List(ctx context.Context) ([]IPSet, error) {
	var sets []IPSet
	if err := s.client.Get(ctx, "/cluster/firewall/ipset", &sets, nil); err != nil {
		return nil, err
	}

	return sets, nil
}

// Create creates a new IP set via POST /cluster/firewall/ipset.
func (s *ipsetResource) Create(ctx context.Context, opts *IPSetOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: ipset options are required")
	}
	if opts.Name == "" {
		return fmt.Errorf("firewall: ipset name is required")
	}

	params, err := opts.encode()
	if err != nil {
		return err
	}

	return s.client.Create(ctx, "/cluster/firewall/ipset", nil, params)
}

// Update renames and/or re-comments an existing IP set via
// POST /cluster/firewall/ipset.
//
// Proxmox has no dedicated update endpoint for an IP set's own metadata:
// the same POST used by Create doubles as the update, distinguished by the
// "rename" parameter — when it names an existing set, Proxmox updates that
// set in place instead of creating a new one. Update sets "rename" to the
// current name (the name argument here) and targets opts.Name as the
// result; set opts.Name equal to name to update Comment without renaming,
// or to a different value to rename the set. If opts.Name is left empty,
// it defaults to name so the set keeps its current name.
func (s *ipsetResource) Update(ctx context.Context, name string, opts *IPSetOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: ipset options are required")
	}

	params, err := opts.encode()
	if err != nil {
		return err
	}

	params["rename"] = name
	if opts.Name == "" {
		params["name"] = name
	}

	return s.client.Create(ctx, "/cluster/firewall/ipset", nil, params)
}

// Delete removes an IP set via DELETE /cluster/firewall/ipset/{name}. If
// force is true, any remaining members of the set are deleted along with
// it; otherwise Proxmox refuses to delete a non-empty set.
func (s *ipsetResource) Delete(ctx context.Context, name string, force bool) error {
	var params map[string]string
	if force {
		params = map[string]string{"force": "1"}
	}

	return s.client.Delete(ctx, "/cluster/firewall/ipset/"+name, nil, params)
}

// Entries returns an accessor for the members (CIDR entries) of the IP set
// identified by name, under /cluster/firewall/ipset/{name}.
func (s *ipsetResource) Entries(name string) *ipsetEntriesResource {
	return &ipsetEntriesResource{client: s.client, name: name}
}

// ipsetEntriesResource provides access to the members of a single IP set:
// GET/POST /cluster/firewall/ipset/{name} and
// GET/PUT/DELETE /cluster/firewall/ipset/{name}/{cidr}.
type ipsetEntriesResource struct {
	client Getter
	// name is the IP set this accessor's members belong to.
	name string
}

// List retrieves all members of the IP set via
// GET /cluster/firewall/ipset/{name}.
func (e *ipsetEntriesResource) List(ctx context.Context) ([]IPSetEntry, error) {
	var entries []IPSetEntry
	if err := e.client.Get(ctx, "/cluster/firewall/ipset/"+e.name, &entries, nil); err != nil {
		return nil, err
	}

	return entries, nil
}

// Get retrieves a single member via
// GET /cluster/firewall/ipset/{name}/{cidr}.
func (e *ipsetEntriesResource) Get(ctx context.Context, cidr string) (*IPSetEntry, error) {
	var entry IPSetEntry
	if err := e.client.Get(ctx, fmt.Sprintf("/cluster/firewall/ipset/%s/%s", e.name, cidr), &entry, nil); err != nil {
		return nil, err
	}

	return &entry, nil
}

// Create adds a new member to the IP set via
// POST /cluster/firewall/ipset/{name}.
func (e *ipsetEntriesResource) Create(ctx context.Context, opts *IPSetEntryOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: ipset entry options are required")
	}
	if opts.CIDR == "" {
		return fmt.Errorf("firewall: ipset entry cidr is required")
	}

	params, err := opts.encode()
	if err != nil {
		return err
	}

	return e.client.Create(ctx, "/cluster/firewall/ipset/"+e.name, nil, params)
}

// Update modifies an existing member via
// PUT /cluster/firewall/ipset/{name}/{cidr}. The member is addressed by its
// current CIDR in the URL (the cidr argument); opts.CIDR must still be set
// (Proxmox requires it to be resent even when unchanged) and, if different
// from cidr, changes the member's address.
func (e *ipsetEntriesResource) Update(ctx context.Context, cidr string, opts *IPSetEntryOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: ipset entry options are required")
	}
	if opts.CIDR == "" {
		return fmt.Errorf("firewall: ipset entry cidr is required")
	}

	params, err := opts.encode()
	if err != nil {
		return err
	}

	return e.client.Update(ctx, fmt.Sprintf("/cluster/firewall/ipset/%s/%s", e.name, cidr), nil, params)
}

// Delete removes a member via DELETE /cluster/firewall/ipset/{name}/{cidr}.
// Proxmox requires cidr to be resent as a parameter even though it is
// already part of the URL path; digest, when non-empty, guards against a
// concurrent modification of the member.
func (e *ipsetEntriesResource) Delete(ctx context.Context, cidr string, digest string) error {
	params := map[string]string{"cidr": cidr}
	if digest != "" {
		params["digest"] = digest
	}

	return e.client.Delete(ctx, fmt.Sprintf("/cluster/firewall/ipset/%s/%s", e.name, cidr), nil, params)
}
