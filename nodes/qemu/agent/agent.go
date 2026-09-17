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

// Package agent provides access to the Proxmox VE per-guest QEMU Guest
// Agent API (endpoints under /nodes/{node}/qemu/{vmid}/agent).
//
// Every command requires a running guest with a responsive QEMU Guest
// Agent; Proxmox returns an error otherwise (e.g. the agent isn't
// enabled in the guest's config, or hasn't connected yet).
//
// Most commands (everything except Exec/ExecStatus/FileRead/FileWrite/
// SetUserPassword) share a single wire shape Proxmox itself declares as
// opaque: `{"result": <command-specific value>}`, with no further schema
// — its own API viewer marks the return type as a bare "object" for
// every one of them. This package still types each command's Result
// according to the QEMU Guest Agent protocol's own (separately stable
// and documented) wire format, since that shape is well-known even
// though Proxmox's schema doesn't pin it down.
package agent

import (
	"context"
	"strconv"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the agent package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /nodes/{node}/qemu/{vmid}/agent
// resource tree, scoped to the node given to New. Every method takes the
// guest's VMID as a call argument, since this package has no persistent
// per-guest scope of its own.
type Client struct {
	client Getter
	node   string
}

// New returns a new agent client backed by the given root client, scoped
// to node.
func New(c Getter, node string) *Client {
	return &Client{client: c, node: node}
}

// path builds the .../agent/{command} URL for the given node and vmid.
func path(node string, vmid int, command string) string {
	return "/nodes/" + node + "/qemu/" + strconv.Itoa(vmid) + "/agent/" + command
}

// resultEnvelope decodes the `{"result": ...}` wrapper shared by every
// auto-generated QGA command endpoint.
type resultEnvelope[T any] struct {
	Result T `json:"result,omitempty" url:"result,omitempty"`
}

// getResult GETs path and unwraps its `{"result": ...}` envelope.
func getResult[T any](ctx context.Context, c Getter, p string, params map[string]string) (T, error) {
	var env resultEnvelope[T]
	if err := c.Get(ctx, p, &env, params); err != nil {
		var zero T
		return zero, err
	}

	return env.Result, nil
}

// postResult POSTs to path and unwraps its `{"result": ...}` envelope.
func postResult[T any](ctx context.Context, c Getter, p string, params map[string]string) (T, error) {
	var env resultEnvelope[T]
	if err := c.Create(ctx, p, &env, params); err != nil {
		var zero T
		return zero, err
	}

	return env.Result, nil
}
