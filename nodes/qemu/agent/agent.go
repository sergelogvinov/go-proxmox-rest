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
// enabled in the guest's config, or hasn't connected yet). A generic
// command that fails inside the guest returns a *GuestError.
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
	"encoding/json"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
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
// auto-generated QGA command endpoint. The result stays raw until
// unwrapResult knows whether it is the command's value or the guest's
// error.
type resultEnvelope struct {
	Result json.RawMessage `json:"result,omitempty" url:"result,omitempty"`
}

// GuestError is a failure the guest agent reported from inside the guest.
// Proxmox answers such a generic command with HTTP 200 and the agent's own
// {"error": {"class": ..., "desc": ...}} as the command's result, so it
// never reaches the client as an APIError. Use errors.As to recover Class
// and Desc.
type GuestError struct {
	// Class is the agent's error class, e.g. "GenericError".
	Class string
	// Desc is the agent's description of the failure.
	Desc string
}

// Error implements the error interface.
func (e *GuestError) Error() string {
	return "agent: guest error: " + e.Class + ": " + e.Desc
}

// guestError returns the guest's error if result is an object whose
// "error" key holds an object with a non-empty "desc", the condition
// Proxmox's own check_agent_error uses. Keys match exactly, which
// encoding/json's struct decoding would not do.
func guestError(result json.RawMessage) *GuestError {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(result, &object); err == nil {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(object["error"], &fields); err == nil {
			var e GuestError
			if err := json.Unmarshal(fields["desc"], &e.Desc); err == nil && e.Desc != "" {
				// A class of the wrong type still means the guest failed,
				// so it only leaves Class empty.
				_ = json.Unmarshal(fields["class"], &e.Class)

				return &e
			}
		}
	}

	return nil
}

// unwrapResult returns the envelope's result as a T, or the guest's
// error as a *GuestError.
func unwrapResult[T any](env resultEnvelope) (T, error) {
	var v T

	if e := guestError(env.Result); e != nil {
		return v, e
	}

	if err := params.Decode(env.Result, &v); err != nil {
		var zero T
		return zero, err
	}

	return v, nil
}

// getResult GETs path and unwraps its `{"result": ...}` envelope.
func getResult[T any](ctx context.Context, c Getter, p string, params map[string]string) (T, error) {
	var env resultEnvelope
	if err := c.Get(ctx, p, &env, params); err != nil {
		var zero T
		return zero, err
	}

	return unwrapResult[T](env)
}

// postResult POSTs to path and unwraps its `{"result": ...}` envelope.
func postResult[T any](ctx context.Context, c Getter, p string, params map[string]string) (T, error) {
	var env resultEnvelope
	if err := c.Create(ctx, p, &env, params); err != nil {
		var zero T
		return zero, err
	}

	return unwrapResult[T](env)
}
