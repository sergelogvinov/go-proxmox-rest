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

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Options provides access to GET/PUT {path}, a firewall's
// configuration. T is the scope's options type: the per-guest firewalls
// share types.GuestOptions, while the cluster-wide firewall has its own
// option set.
type Options[T any] struct {
	client Getter
	path   string
}

// NewOptions returns the options singleton at path.
func NewOptions[T any](c Getter, path string) *Options[T] {
	return &Options[T]{client: c, path: path}
}

// Get executes GET {path} and returns the current firewall
// configuration.
func (o *Options[T]) Get(ctx context.Context) (*T, error) {
	var opts T
	if err := o.client.Get(ctx, o.path, &opts, nil); err != nil {
		return nil, err
	}

	return &opts, nil
}

// Update executes PUT {path}, applying the given options to the
// firewall configuration. Only non-zero fields are sent; use the
// options type's Delete field to explicitly reset a field to its
// default.
func (o *Options[T]) Update(ctx context.Context, opts *T) error {
	if opts == nil {
		return fmt.Errorf("firewall: options are required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return err
	}

	return o.client.Update(ctx, o.path, nil, p)
}
