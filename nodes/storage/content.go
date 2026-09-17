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

package storage

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// contentResource provides access to
// /nodes/{node}/storage/{storage}/content.
type contentResource struct {
	client Getter
	node   string
}

// List retrieves a storage's volumes via
// GET /nodes/{node}/storage/{storage}/content. opts may be nil to
// request every volume unfiltered.
func (r *contentResource) List(ctx context.Context, storageID string, opts *ContentListOptions) ([]Volume, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var volumes []Volume
	if err := r.client.Get(ctx, "/nodes/"+r.node+"/storage/"+storageID+"/content", &volumes, p); err != nil {
		return nil, err
	}

	return volumes, nil
}

// Get retrieves a single volume's attributes via
// GET /nodes/{node}/storage/{storage}/content/{volume}. volume may be a
// bare volume name (resolved against storageID) or a full "storage:name"
// volume id.
func (r *contentResource) Get(ctx context.Context, storageID, volume string) (*Volume, error) {
	v := &Volume{}
	if err := r.client.Get(ctx, "/nodes/"+r.node+"/storage/"+storageID+"/content/"+volume, v, nil); err != nil {
		return nil, err
	}

	return v, nil
}

// Create allocates a new disk image via
// POST /nodes/{node}/storage/{storage}/content. Returns the new volume's
// id.
func (r *contentResource) Create(ctx context.Context, storageID string, opts *CreateVolumeOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("storage: content create options are required")
	}
	if opts.Filename == "" {
		return "", fmt.Errorf("storage: content create filename is required")
	}
	if opts.Size == "" {
		return "", fmt.Errorf("storage: content create size is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var volid string
	if err := r.client.Create(ctx, "/nodes/"+r.node+"/storage/"+storageID+"/content", &volid, p); err != nil {
		return "", err
	}

	return volid, nil
}

// Update modifies a volume's notes/protected attributes via
// PUT /nodes/{node}/storage/{storage}/content/{volume}.
func (r *contentResource) Update(ctx context.Context, storageID, volume string, opts *UpdateVolumeOptions) error {
	p, err := params.Encode(opts)
	if err != nil {
		return err
	}

	return r.client.Update(ctx, "/nodes/"+r.node+"/storage/"+storageID+"/content/"+volume, nil, p)
}

// Delete removes a volume via
// DELETE /nodes/{node}/storage/{storage}/content/{volume}. delay is the
// number of seconds (1-30) to wait for the removal to finish before
// falling back to a background task; zero requests Proxmox's default (a
// background task, its UPID returned immediately). The returned UPID is
// empty when the removal completed synchronously within delay.
func (r *contentResource) Delete(ctx context.Context, storageID, volume string, delay int) (string, error) {
	var p map[string]string
	if delay > 0 {
		p = map[string]string{"delay": strconv.Itoa(delay)}
	}

	var upid string
	if err := r.client.Delete(ctx, "/nodes/"+r.node+"/storage/"+storageID+"/content/"+volume, &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
