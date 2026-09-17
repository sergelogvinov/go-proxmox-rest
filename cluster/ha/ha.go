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

// Package ha provides access to the Proxmox VE cluster high-availability API
// (endpoints under /cluster/ha).
package ha

import (
	"context"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the ha package decoupled from
// the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /cluster/ha resource tree, the cluster-wide
// high-availability configuration.
type Client struct {
	client Getter
}

// New returns a new ha client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}
