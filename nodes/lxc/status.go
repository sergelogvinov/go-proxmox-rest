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
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// path builds the .../lxc/{vmid}/status/{action} URL for the given node,
// vmid, and status sub-path (which may be empty).
func path(node string, vmid int, action string) string {
	return "/nodes/" + node + "/lxc/" + strconv.Itoa(vmid) + "/status/" + action
}

// Status retrieves a container's current status via
// GET /nodes/{node}/lxc/{vmid}/status/current.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (c *Client) Status(ctx context.Context, vmid int) (*Status, error) {
	s := &Status{}
	if err := c.client.Get(ctx, path(c.node, vmid, "current"), s, nil); err != nil {
		return nil, err
	}

	return s, nil
}

// Start starts a container via POST /nodes/{node}/lxc/{vmid}/status/start.
// opts may be nil to use Proxmox's defaults. Returns the start task's
// UPID.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.PowerMgmt,match=all
func (c *Client) Start(ctx context.Context, vmid int, opts *StartOptions) (string, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return "", err
		}
	}

	return c.action(ctx, vmid, "start", p)
}

// Stop immediately stops a container via
// POST /nodes/{node}/lxc/{vmid}/status/stop — this abruptly stops every
// process running in the container; prefer Shutdown for a graceful
// power-off. opts may be nil to use Proxmox's defaults. Returns the stop
// task's UPID.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.PowerMgmt,match=all
func (c *Client) Stop(ctx context.Context, vmid int, opts *StopOptions) (string, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return "", err
		}
	}

	return c.action(ctx, vmid, "stop", p)
}

// Shutdown gracefully stops a container (see lxc-stop(1)) via
// POST /nodes/{node}/lxc/{vmid}/status/shutdown. opts may be nil to use
// Proxmox's defaults. Returns the shutdown task's UPID.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.PowerMgmt,match=all
func (c *Client) Shutdown(ctx context.Context, vmid int, opts *ShutdownOptions) (string, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return "", err
		}
	}

	return c.action(ctx, vmid, "shutdown", p)
}

// Reboot shuts a container down and starts it again, applying any
// pending configuration changes, via
// POST /nodes/{node}/lxc/{vmid}/status/reboot. opts may be nil to use
// Proxmox's defaults. Returns the reboot task's UPID.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.PowerMgmt,match=all
func (c *Client) Reboot(ctx context.Context, vmid int, opts *RebootOptions) (string, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return "", err
		}
	}

	return c.action(ctx, vmid, "reboot", p)
}

// Suspend suspends a running container (experimental, per Proxmox's own
// description) via POST /nodes/{node}/lxc/{vmid}/status/suspend. Returns
// the suspend task's UPID.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.PowerMgmt,match=all
func (c *Client) Suspend(ctx context.Context, vmid int) (string, error) {
	return c.action(ctx, vmid, "suspend", nil)
}

// Resume resumes a suspended container via
// POST /nodes/{node}/lxc/{vmid}/status/resume. Returns the resume task's
// UPID.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.PowerMgmt,match=all
func (c *Client) Resume(ctx context.Context, vmid int) (string, error) {
	return c.action(ctx, vmid, "resume", nil)
}

// action POSTs to the given .../status/{name} sub-path, returning the
// resulting task's UPID.
func (c *Client) action(ctx context.Context, vmid int, name string, p map[string]string) (string, error) {
	var upid string
	if err := c.client.Create(ctx, path(c.node, vmid, name), &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
