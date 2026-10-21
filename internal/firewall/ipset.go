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

// IPSets provides access to an IP set index: GET/POST {base} and
// DELETE {base}/{name}. A given set's members are reached via
// Entries(name).
type IPSets struct {
	client Getter
	base   string
}

// NewIPSets returns the IP set index rooted at base.
func NewIPSets(c Getter, base string) *IPSets {
	return &IPSets{client: c, base: base}
}

// List retrieves all IP sets via GET {base}.
func (s *IPSets) List(ctx context.Context) ([]types.IPSet, error) {
	var sets []types.IPSet
	if err := s.client.Get(ctx, s.base, &sets, nil); err != nil {
		return nil, err
	}

	return sets, nil
}

// Create creates a new IP set via POST {base}.
func (s *IPSets) Create(ctx context.Context, opts *types.IPSetOptions) error {
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
func (s *IPSets) Update(ctx context.Context, name string, opts *types.IPSetOptions) error {
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
func (s *IPSets) Delete(ctx context.Context, name string, force bool) error {
	var params map[string]string
	if force {
		params = map[string]string{"force": "1"}
	}

	return s.client.Delete(ctx, s.base+"/"+name, nil, params)
}

// Entries returns an accessor for the members (CIDR entries) of the IP
// set identified by name, under {base}/{name}.
func (s *IPSets) Entries(name string) *IPSetEntries {
	return &IPSetEntries{client: s.client, base: s.base + "/" + name}
}

// IPSetEntries provides access to the members of a single IP set:
// GET/POST {base} and GET/PUT/DELETE {base}/{cidr}.
type IPSetEntries struct {
	client Getter
	// base is the owning IP set's own collection path, e.g.
	// ".../firewall/ipset/my-ipset".
	base string
}

// List retrieves all members of the IP set via GET {base}.
func (e *IPSetEntries) List(ctx context.Context) ([]types.IPSetEntry, error) {
	var entries []types.IPSetEntry
	if err := e.client.Get(ctx, e.base, &entries, nil); err != nil {
		return nil, err
	}

	return entries, nil
}

// Get retrieves a single member via GET {base}/{cidr}.
func (e *IPSetEntries) Get(ctx context.Context, cidr string) (*types.IPSetEntry, error) {
	var entry types.IPSetEntry
	if err := e.client.Get(ctx, e.base+"/"+cidr, &entry, nil); err != nil {
		return nil, err
	}

	return &entry, nil
}

// Create adds a new member to the IP set via POST {base}.
func (e *IPSetEntries) Create(ctx context.Context, opts *types.IPSetEntryOptions) error {
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
func (e *IPSetEntries) Update(ctx context.Context, cidr string, opts *types.IPSetEntryOptions) error {
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
func (e *IPSetEntries) Delete(ctx context.Context, cidr string, digest string) error {
	params := map[string]string{"cidr": cidr}
	if digest != "" {
		params["digest"] = digest
	}

	return e.client.Delete(ctx, e.base+"/"+cidr, nil, params)
}
