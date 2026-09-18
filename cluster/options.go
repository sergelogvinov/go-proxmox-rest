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

package cluster

import (
	"context"
)

// optionsResource provides access to GET/PUT /cluster/options, the
// cluster-wide datacenter configuration.
type optionsResource struct {
	client Getter
}

// Options returns an accessor for GET/PUT /cluster/options.
func (c *Client) Options() *optionsResource {
	return &optionsResource{client: c.client}
}

// Get executes GET /cluster/options and returns the current cluster-wide
// configuration.
// Without this permission, Proxmox returns only a restricted subset.
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Audit,match=any
func (o *optionsResource) Get(ctx context.Context) (*Options, error) {
	var opts Options
	if err := o.client.Get(ctx, "/cluster/options", &opts, nil); err != nil {
		return nil, err
	}

	return &opts, nil
}

// Update executes PUT /cluster/options, applying the given options to the
// cluster-wide configuration. Only non-zero fields are sent; use
// Options.Delete to explicitly reset a field to its default.
//
// +proxmox:rbac:path=/,method=PUT,privs=Sys.Modify,match=all
func (o *optionsResource) Update(ctx context.Context, opts *Options) error {
	params, err := opts.encode()
	if err != nil {
		return err
	}

	return o.client.Update(ctx, "/cluster/options", nil, params)
}
