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

// Package firewall is the single implementation of the Proxmox VE
// firewall resource tree (rules, aliases, IP sets and their entries,
// options, refs and the log), parameterised by the tree's base path.
//
// Proxmox backs the cluster-wide (/cluster/firewall) and per-guest
// (/nodes/{node}/{qemu,lxc}/{vmid}/firewall) firewalls with the same
// PVE::Firewall::Rules/Aliases/IPSet modules, so each sub-resource has
// the same request shape and the same quirks whatever scope it lives
// under. The public cluster/firewall, nodes/qemu/firewall and
// nodes/lxc/firewall packages are thin adapters: each supplies its base
// path (and, for the cluster scope, security groups and its own Options
// type) and hands back the resources defined here.
package firewall

import (
	"context"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
	"github.com/sergelogvinov/go-proxmox-rest/types"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client and by the Getter interfaces of the
// public firewall packages.
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Refs retrieves the aliases and/or IP sets that may be referenced from
// a rule's Source/Dest fields via GET {path}. An empty refType returns
// both kinds; refType narrows the result to just aliases or just IP
// sets.
func Refs(ctx context.Context, c Getter, path string, refType types.RefType) ([]types.Ref, error) {
	var p map[string]string
	if refType != "" {
		p = map[string]string{"type": string(refType)}
	}

	var refs []types.Ref
	if err := c.Get(ctx, path, &refs, p); err != nil {
		return nil, err
	}

	return refs, nil
}

// Log retrieves firewall log entries via GET {path}. opts may be nil to
// request Proxmox's default window.
func Log(ctx context.Context, c Getter, path string, opts *types.LogOptions) ([]types.LogEntry, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var entries []types.LogEntry
	if err := c.Get(ctx, path, &entries, p); err != nil {
		return nil, err
	}

	return entries, nil
}
