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
)

// optionsResource provides access to GET/PUT {path}, a guest's
// firewall configuration. Obtain it via Client.Options(vmid).
type optionsResource struct {
	client Getter
	path   string
}

// Get executes GET {path} and returns the guest's current firewall
// configuration.
// GET /nodes/{node}/qemu/{vmid}/firewall/options.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (o *optionsResource) Get(ctx context.Context) (*Options, error) {
	var opts Options
	if err := o.client.Get(ctx, o.path, &opts, nil); err != nil {
		return nil, err
	}

	return &opts, nil
}

// Update executes PUT {path}, applying the given options to the
// guest's firewall configuration. Only non-zero fields are sent; use
// Options.Delete to explicitly reset a field to its default.
// PUT /nodes/{node}/qemu/{vmid}/firewall/options.
//
// +proxmox:rbac:path=/vms/{vmid},method=PUT,privs=VM.Config.Network,match=all
func (o *optionsResource) Update(ctx context.Context, opts *Options) error {
	params, err := opts.Encode()
	if err != nil {
		return err
	}

	return o.client.Update(ctx, o.path, nil, params)
}
