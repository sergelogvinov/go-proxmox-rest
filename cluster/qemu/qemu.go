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

// Package qemu provides access to the Proxmox VE cluster-wide QEMU API
// (endpoints under /cluster/qemu): available CPU flags and custom CPU
// model definitions.
package qemu

import (
	"context"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the qemu package decoupled from
// the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /cluster/qemu resource tree.
type Client struct {
	client Getter
}

// New returns a new qemu client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// CPUFlags retrieves the CPU flags available cluster-wide via
// GET /cluster/qemu/cpu-flags.
//
// arch is currently only meaningfully implemented for ArchX8664 by
// Proxmox (ArchAarch64 always returns an empty list); an empty arch
// defaults to the host's own architecture. An empty accel defaults to
// AccelKVM.
//
// No fixed Proxmox privilege is required; the endpoint is declared
// user => all. The URL is GET /cluster/qemu/cpu-flags.
func (c *Client) CPUFlags(ctx context.Context, arch Arch, accel Accel) ([]CPUFlag, error) {
	var reqParams map[string]string
	if arch != "" || accel != "" {
		reqParams = map[string]string{}
		if arch != "" {
			reqParams["arch"] = string(arch)
		}
		if accel != "" {
			reqParams["accel"] = string(accel)
		}
	}

	var flags []CPUFlag
	if err := c.client.Get(ctx, "/cluster/qemu/cpu-flags", &flags, reqParams); err != nil {
		return nil, err
	}

	return flags, nil
}

// CustomCPUModels returns an accessor for the
// /cluster/qemu/custom-cpu-models resource, cluster-wide custom CPU model
// definitions.
func (c *Client) CustomCPUModels() *customCPUModelsResource {
	return &customCPUModelsResource{client: c.client}
}
