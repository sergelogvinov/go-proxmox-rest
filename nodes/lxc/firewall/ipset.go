package firewall

import (
	"context"
	"fmt"
)

// ipsetResource provides access to GET/POST {base} and
// DELETE {base}/{name}, a guest's IP set index. A given set's members
// are reached via Entries(name). Obtain it via Client.IPSet(node, vmid).
type ipsetResource struct {
	client Getter
	base   string
}

// List retrieves all IP sets via GET {base}.
func (s *ipsetResource) List(ctx context.Context) ([]IPSet, error) {
	var sets []IPSet
	if err := s.client.Get(ctx, s.base, &sets, nil); err != nil {
		return nil, err
	}

	return sets, nil
}

// Create creates a new IP set via POST {base}.
func (s *ipsetResource) Create(ctx context.Context, opts *IPSetOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: ipset options are required")
	}
	if opts.Name == "" {
		return fmt.Errorf("firewall: ipset name is required")
	}

	params, err := opts.Encode()
	if err != nil {
		return err
	}

	return s.client.Create(ctx, s.base, nil, params)
}

// Update renames and/or re-comments an existing IP set via POST {base}.
//
// Proxmox has no dedicated update endpoint for an IP set's own
// metadata: the same POST used by Create doubles as the update,
// distinguished by the "rename" parameter — when it names an existing
// set, Proxmox updates that set in place instead of creating a new one.
// Update sets "rename" to the current name (the name argument here) and
// targets opts.Name as the result; set opts.Name equal to name to
// update Comment without renaming, or to a different value to rename
// the set. If opts.Name is left empty, it defaults to name so the set
// keeps its current name.
func (s *ipsetResource) Update(ctx context.Context, name string, opts *IPSetOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: ipset options are required")
	}

	params, err := opts.Encode()
	if err != nil {
		return err
	}

	params["rename"] = name
	if opts.Name == "" {
		params["name"] = name
	}

	return s.client.Create(ctx, s.base, nil, params)
}

// Delete removes an IP set via DELETE {base}/{name}. If force is true,
// any remaining members of the set are deleted along with it;
// otherwise Proxmox refuses to delete a non-empty set.
func (s *ipsetResource) Delete(ctx context.Context, name string, force bool) error {
	var params map[string]string
	if force {
		params = map[string]string{"force": "1"}
	}

	return s.client.Delete(ctx, s.base+"/"+name, nil, params)
}

// Entries returns an accessor for the members (CIDR entries) of the IP
// set identified by name, under {base}/{name}.
func (s *ipsetResource) Entries(name string) *ipsetEntriesResource {
	return &ipsetEntriesResource{client: s.client, base: s.base + "/" + name}
}

// ipsetEntriesResource provides access to the members of a single IP
// set: GET/POST {base} and GET/PUT/DELETE {base}/{cidr}.
type ipsetEntriesResource struct {
	client Getter
	// base is the owning IP set's own collection path, e.g.
	// ".../firewall/ipset/my-ipset".
	base string
}

// List retrieves all members of the IP set via GET {base}.
func (e *ipsetEntriesResource) List(ctx context.Context) ([]IPSetEntry, error) {
	var entries []IPSetEntry
	if err := e.client.Get(ctx, e.base, &entries, nil); err != nil {
		return nil, err
	}

	return entries, nil
}

// Get retrieves a single member via GET {base}/{cidr}.
func (e *ipsetEntriesResource) Get(ctx context.Context, cidr string) (*IPSetEntry, error) {
	var entry IPSetEntry
	if err := e.client.Get(ctx, e.base+"/"+cidr, &entry, nil); err != nil {
		return nil, err
	}

	return &entry, nil
}

// Create adds a new member to the IP set via POST {base}.
func (e *ipsetEntriesResource) Create(ctx context.Context, opts *IPSetEntryOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: ipset entry options are required")
	}
	if opts.CIDR == "" {
		return fmt.Errorf("firewall: ipset entry cidr is required")
	}

	params, err := opts.Encode()
	if err != nil {
		return err
	}

	return e.client.Create(ctx, e.base, nil, params)
}

// Update modifies an existing member via PUT {base}/{cidr}. The member
// is addressed by its current CIDR in the URL (the cidr argument);
// opts.CIDR must still be set (Proxmox requires it to be resent even
// when unchanged) and, if different from cidr, changes the member's
// address.
func (e *ipsetEntriesResource) Update(ctx context.Context, cidr string, opts *IPSetEntryOptions) error {
	if opts == nil {
		return fmt.Errorf("firewall: ipset entry options are required")
	}
	if opts.CIDR == "" {
		return fmt.Errorf("firewall: ipset entry cidr is required")
	}

	params, err := opts.Encode()
	if err != nil {
		return err
	}

	return e.client.Update(ctx, e.base+"/"+cidr, nil, params)
}

// Delete removes a member via DELETE {base}/{cidr}. Proxmox requires
// cidr to be resent as a parameter even though it is already part of
// the URL path; digest, when non-empty, guards against a concurrent
// modification of the member.
func (e *ipsetEntriesResource) Delete(ctx context.Context, cidr string, digest string) error {
	params := map[string]string{"cidr": cidr}
	if digest != "" {
		params["digest"] = digest
	}

	return e.client.Delete(ctx, e.base+"/"+cidr, nil, params)
}
