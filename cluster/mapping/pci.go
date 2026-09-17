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

package mapping

import (
	"context"
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// PCI describes a PCI hardware mapping as returned by
// GET /cluster/mapping/pci and GET /cluster/mapping/pci/{id}.
//
// Proxmox's single-item GET oddly omits ID (only List's response sets it),
// so pciResource.Get fills it in from the requested id after decoding.
type PCI struct {
	// ID is the logical mapping identifier, e.g. "my-gpu".
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Type is always "pci".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Description is the mapping description.
	Description string `json:"description,omitempty" url:"description,omitempty"`
	// Mdev marks the device(s) as capable of providing mediated devices.
	Mdev bool `json:"mdev,omitempty" url:"mdev,omitempty"`
	// LiveMigrationCapable marks the device(s) as able to be
	// live-migrated (experimental; needs hardware/driver support).
	LiveMigrationCapable bool `json:"live-migration-capable,omitempty" url:"live-migration-capable,omitempty"`
	// Map is the list of per-node property-string entries, e.g.
	// "node=pve1,path=0000:01:00.0,id=10de:1eb8".
	Map []string `json:"map,omitempty" url:"map,omitempty"`
	// Digest is the configuration digest, usable with
	// PCIOptions.Digest to guard concurrent updates.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
	// Checks is the diagnostics for this mapping on the requested
	// check-node. Only populated when List's checkNode is non-empty.
	Checks []Check `json:"checks,omitempty" url:"checks,omitempty"`
}

// PCIOptions holds the write parameters shared by
// POST /cluster/mapping/pci (Create) and PUT /cluster/mapping/pci/{id}
// (Update).
//
// ID is required by Create; Update ignores it (the mapping id is already
// part of the URL). Map is not a `url` struct tag Encode/Decode understand:
// unlike every other slice in this codebase, Proxmox declares it a genuine
// "array" parameter (not a comma-joined `-list` string format), and its
// entries are themselves comma-bearing property strings, so it is encoded
// separately as repeated "map" values by pciResource.Create/Update — see
// Getter's doc comment.
type PCIOptions struct {
	// ID is the mapping identifier. Required by Create; not settable on
	// Update.
	ID string `url:"id"`
	// Description is the mapping description.
	Description *string `url:"description"`
	// Mdev marks the device(s) as capable of providing mediated devices.
	Mdev *bool `url:"mdev"`
	// LiveMigrationCapable marks the device(s) as able to be
	// live-migrated (experimental; needs hardware/driver support).
	LiveMigrationCapable *bool `url:"live-migration-capable"`
	// Map is the list of per-node property-string entries. Required by
	// both Create and Update — Proxmox does not treat it as optional on
	// Update, so the full, current entry list must be resent on every
	// call.
	Map []string `url:"-"`
	// Delete lists properties to reset to their default value (Update
	// only).
	Delete []string `url:"delete"`
	// Digest prevents changes if the current configuration has changed
	// in between (value from the corresponding Get). Update only.
	Digest string `url:"digest"`
}

// encode converts the options to query parameters, keeping Map out of the
// generic scalar encoder (see PCIOptions's doc comment).
func (o *PCIOptions) encode() (map[string]string, []string, error) {
	if o == nil {
		return nil, nil, fmt.Errorf("mapping: pci options are required")
	}

	scalars, err := params.Encode(o)
	if err != nil {
		return nil, nil, err
	}

	return scalars, o.Map, nil
}

// pciResource provides access to GET/POST /cluster/mapping/pci and
// GET/PUT/DELETE /cluster/mapping/pci/{id}. Obtain it via Client.PCI().
type pciResource struct {
	client Getter
}

// List retrieves all PCI mappings via GET /cluster/mapping/pci.
//
// checkNode, when non-empty, asks Proxmox to validate every mapping's
// configuration against that node's actual hardware and populate Checks
// with any problems found.
func (r *pciResource) List(ctx context.Context, checkNode string) ([]PCI, error) {
	var params map[string]string
	if checkNode != "" {
		params = map[string]string{"check-node": checkNode}
	}

	var mappings []PCI
	if err := r.client.Get(ctx, "/cluster/mapping/pci", &mappings, params); err != nil {
		return nil, err
	}

	return mappings, nil
}

// Get retrieves a single PCI mapping via GET /cluster/mapping/pci/{id}.
func (r *pciResource) Get(ctx context.Context, id string) (*PCI, error) {
	var m PCI
	if err := r.client.Get(ctx, "/cluster/mapping/pci/"+id, &m, nil); err != nil {
		return nil, err
	}
	m.ID = id

	return &m, nil
}

// Create creates a new PCI mapping via POST /cluster/mapping/pci.
func (r *pciResource) Create(ctx context.Context, opts *PCIOptions) error {
	if opts == nil {
		return fmt.Errorf("mapping: pci options are required")
	}
	if opts.ID == "" {
		return fmt.Errorf("mapping: pci id is required")
	}

	scalars, mapEntries, err := opts.encode()
	if err != nil {
		return err
	}

	return r.client.CreateValues(ctx, "/cluster/mapping/pci", nil, toValues(scalars, mapEntries))
}

// Update modifies an existing PCI mapping via PUT /cluster/mapping/pci/{id}.
//
// opts.Map is required — Proxmox does not treat it as optional on update,
// so it must be resent (with its full, current entry list) on every call,
// the same as ha.RuleOptions.Type.
func (r *pciResource) Update(ctx context.Context, id string, opts *PCIOptions) error {
	if opts == nil {
		return fmt.Errorf("mapping: pci options are required")
	}
	if len(opts.Map) == 0 {
		return fmt.Errorf("mapping: pci map is required")
	}

	scalars, mapEntries, err := opts.encode()
	if err != nil {
		return err
	}

	return r.client.UpdateValues(ctx, "/cluster/mapping/pci/"+id, nil, toValues(scalars, mapEntries))
}

// Delete removes a PCI mapping via DELETE /cluster/mapping/pci/{id}.
func (r *pciResource) Delete(ctx context.Context, id string) error {
	return r.client.Delete(ctx, "/cluster/mapping/pci/"+id, nil, nil)
}
