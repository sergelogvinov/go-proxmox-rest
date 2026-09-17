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

// Package firewall provides access to the Proxmox VE cluster firewall API
// (endpoints under /cluster/firewall).
package firewall

import (
	"context"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the firewall package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /cluster/firewall resource tree, the
// cluster-wide firewall configuration (rules, security groups, aliases, IP
// sets and options).
type Client struct {
	client Getter
}

// New returns a new firewall client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// Options returns an accessor for GET/PUT /cluster/firewall/options, the
// cluster-wide firewall configuration.
func (c *Client) Options() *optionsResource {
	return &optionsResource{client: c.client}
}

// Rules returns an accessor for the cluster-wide firewall rules under
// /cluster/firewall/rules. For a security group's rules, see
// Groups().Rules(name).
func (c *Client) Rules() *ruleResource {
	return &ruleResource{client: c.client, base: "/cluster/firewall/rules"}
}

// Groups returns an accessor for the /cluster/firewall/groups resource,
// cluster-wide security groups.
func (c *Client) Groups() *groupsResource {
	return &groupsResource{client: c.client}
}

// Aliases returns an accessor for the /cluster/firewall/aliases resource,
// cluster-wide IP/network aliases.
func (c *Client) Aliases() *aliasesResource {
	return &aliasesResource{client: c.client}
}

// IPSet returns an accessor for the /cluster/firewall/ipset resource,
// cluster-wide IP sets.
func (c *Client) IPSet() *ipsetResource {
	return &ipsetResource{client: c.client}
}

// Refs retrieves the aliases and/or IP sets that may be referenced from a
// rule's Source/Dest fields via GET /cluster/firewall/refs. An empty
// refType returns both kinds; refType narrows the result to just aliases
// or just IP sets.
func (c *Client) Refs(ctx context.Context, refType RefType) ([]Ref, error) {
	var params map[string]string
	if refType != "" {
		params = map[string]string{"type": string(refType)}
	}

	var refs []Ref
	if err := c.client.Get(ctx, "/cluster/firewall/refs", &refs, params); err != nil {
		return nil, err
	}

	return refs, nil
}
