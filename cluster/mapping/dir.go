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

// Dir describes a directory mapping as returned by
// GET /cluster/mapping/dir and GET /cluster/mapping/dir/{id}. Directory
// mappings share a host directory (identified per node by an absolute
// path) with containers under a single logical name.
//
// Proxmox's single-item GET oddly omits ID (only List's response sets it),
// so dirResource.Get fills it in from the requested id after decoding.
type Dir struct {
	// ID is the logical mapping identifier, e.g. "my-share".
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Type is always "dir".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Description is the mapping description.
	Description string `json:"description,omitempty" url:"description,omitempty"`
	// Map is the list of per-node property-string entries, e.g.
	// "node=pve1,path=/mnt/share".
	Map []string `json:"map,omitempty" url:"map,omitempty"`
	// Digest is the configuration digest, usable with
	// DirOptions.Digest to guard concurrent updates.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
	// Checks is the diagnostics for this mapping on the requested
	// check-node. Only populated when List's checkNode is non-empty.
	Checks []Check `json:"checks,omitempty" url:"checks,omitempty"`
}

// DirOptions holds the write parameters shared by
// POST /cluster/mapping/dir (Create) and PUT /cluster/mapping/dir/{id}
// (Update).
//
// ID is required by Create; Update ignores it (the mapping id is already
// part of the URL). Map is not a `url` struct tag Encode/Decode understand:
// unlike every other slice in this codebase, Proxmox declares it a genuine
// "array" parameter (not a comma-joined `-list` string format), and its
// entries are themselves comma-bearing property strings, so it is encoded
// separately as repeated "map" values by dirResource.Create/Update — see
// Getter's doc comment.
type DirOptions struct {
	// ID is the mapping identifier. Required by Create; not settable on
	// Update.
	ID string `url:"id"`
	// Description is the mapping description.
	Description *string `url:"description"`
	// Map is the list of per-node property-string entries. Required by
	// both Create and Update — Proxmox does not treat it as optional on
	// Update, so the full, current entry list must be resent on every
	// call. Proxmox rejects more than one entry per node.
	Map []string `url:"-"`
	// Delete lists properties to reset to their default value (Update
	// only).
	Delete []string `url:"delete"`
	// Digest prevents changes if the current configuration has changed
	// in between (value from the corresponding Get). Update only.
	Digest string `url:"digest"`
}

// encode converts the options to query parameters, keeping Map out of the
// generic scalar encoder (see DirOptions's doc comment).
func (o *DirOptions) encode() (map[string]string, []string, error) {
	if o == nil {
		return nil, nil, fmt.Errorf("mapping: dir options are required")
	}

	scalars, err := params.Encode(o)
	if err != nil {
		return nil, nil, err
	}

	return scalars, o.Map, nil
}

// dirResource provides access to GET/POST /cluster/mapping/dir and
// GET/PUT/DELETE /cluster/mapping/dir/{id}. Obtain it via Client.Dir().
type dirResource struct {
	client Getter
}

// List retrieves all directory mappings via GET /cluster/mapping/dir.
//
// checkNode, when non-empty, asks Proxmox to validate every mapping's
// configuration against that node (the path must exist and be a
// directory) and populate Checks with any problems found.
func (r *dirResource) List(ctx context.Context, checkNode string) ([]Dir, error) {
	var params map[string]string
	if checkNode != "" {
		params = map[string]string{"check-node": checkNode}
	}

	var mappings []Dir
	if err := r.client.Get(ctx, "/cluster/mapping/dir", &mappings, params); err != nil {
		return nil, err
	}

	return mappings, nil
}

// Get retrieves a single directory mapping via
// GET /cluster/mapping/dir/{id}.
func (r *dirResource) Get(ctx context.Context, id string) (*Dir, error) {
	var m Dir
	if err := r.client.Get(ctx, "/cluster/mapping/dir/"+id, &m, nil); err != nil {
		return nil, err
	}
	m.ID = id

	return &m, nil
}

// Create creates a new directory mapping via POST /cluster/mapping/dir.
func (r *dirResource) Create(ctx context.Context, opts *DirOptions) error {
	if opts == nil {
		return fmt.Errorf("mapping: dir options are required")
	}
	if opts.ID == "" {
		return fmt.Errorf("mapping: dir id is required")
	}

	scalars, mapEntries, err := opts.encode()
	if err != nil {
		return err
	}

	return r.client.CreateValues(ctx, "/cluster/mapping/dir", nil, toValues(scalars, mapEntries))
}

// Update modifies an existing directory mapping via
// PUT /cluster/mapping/dir/{id}.
//
// opts.Map is required — Proxmox does not treat it as optional on update,
// so it must be resent (with its full, current entry list) on every call,
// the same as ha.RuleOptions.Type.
func (r *dirResource) Update(ctx context.Context, id string, opts *DirOptions) error {
	if opts == nil {
		return fmt.Errorf("mapping: dir options are required")
	}
	if len(opts.Map) == 0 {
		return fmt.Errorf("mapping: dir map is required")
	}

	scalars, mapEntries, err := opts.encode()
	if err != nil {
		return err
	}

	return r.client.UpdateValues(ctx, "/cluster/mapping/dir/"+id, nil, toValues(scalars, mapEntries))
}

// Delete removes a directory mapping via DELETE /cluster/mapping/dir/{id}.
func (r *dirResource) Delete(ctx context.Context, id string) error {
	return r.client.Delete(ctx, "/cluster/mapping/dir/"+id, nil, nil)
}
