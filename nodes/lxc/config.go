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

package lxc

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// configPath builds the /nodes/{node}/lxc/{vmid}/config URL for the
// given node and vmid.
func configPath(node string, vmid int) string {
	return "/nodes/" + node + "/lxc/" + strconv.Itoa(vmid) + "/config"
}

// Config retrieves a container's configuration via
// GET /nodes/{node}/lxc/{vmid}/config. opts may be nil to request the
// configuration with pending changes applied.
// The env field is conditionally hidden without VM.Config.Options.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (c *Client) Config(ctx context.Context, vmid int, opts *ConfigOptions) (*Config, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var cfg Config
	if err := c.client.Get(ctx, configPath(c.node, vmid), &cfg, p); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// UpdateConfig sets a container's configuration via
// PUT /nodes/{node}/lxc/{vmid}/config.
// The endpoint gate accepts any listed privilege; changed fields trigger the
// additional field-, storage-, and SDN-dependent checks documented above.
//
// +proxmox:rbac:path=/vms/{vmid},method=PUT,privs=VM.Config.Disk;VM.Config.CPU;VM.Config.Memory;VM.Config.Network;VM.Config.Options,match=any
func (c *Client) UpdateConfig(ctx context.Context, vmid int, cfg *Config) error {
	p, err := encodeConfig(cfg)
	if err != nil {
		return err
	}

	return c.client.Update(ctx, configPath(c.node, vmid), nil, p)
}

// encodeConfig encodes cfg for a config write. The indexed families
// (netN, mpN, unusedN) are encoded by params.Encode itself, via their
// `url:"<prefix>,indexed"` tags.
func encodeConfig(cfg *Config) (map[string]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("lxc: config is required")
	}

	return params.Encode(cfg)
}
