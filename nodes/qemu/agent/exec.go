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

package agent

import (
	"context"
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// SetUserPasswordOptions holds the parameters for Client.SetUserPassword
// (POST .../agent/set-user-password).
type SetUserPasswordOptions struct {
	// Username is the guest user to set the password for. Required.
	Username string `url:"username"`
	// Password is the new password, in plain text — Proxmox itself
	// base64-encodes it before forwarding to the guest agent. Required.
	Password string `url:"password"`
	// Crypted indicates Password has already been run through crypt().
	Crypted bool `url:"crypted,omitempty"`
}

// SetUserPassword sets a guest user's password via
// POST /nodes/{node}/qemu/{vmid}/agent/set-user-password
// (guest-set-user-password).
func (c *Client) SetUserPassword(ctx context.Context, vmid int, opts *SetUserPasswordOptions) error {
	if opts == nil {
		return fmt.Errorf("agent: set-user-password options are required")
	}
	if opts.Username == "" {
		return fmt.Errorf("agent: set-user-password username is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return err
	}

	_, err = postResult[any](ctx, c.client, path(c.node, vmid, "set-user-password"), p)

	return err
}

// ExecOptions holds the parameters for Client.Exec
// (POST .../agent/exec).
type ExecOptions struct {
	// Command is the program and its arguments, e.g.
	// []string{"/bin/echo", "hello"}. Required.
	Command []string `url:"command"`
	// InputData is passed to the guest as the command's stdin.
	InputData string `url:"input-data,omitempty"`
}

// ExecResult identifies a process started by Client.Exec.
type ExecResult struct {
	// PID is the process ID started by the guest agent.
	PID int `json:"pid,omitempty" url:"pid,omitempty"`
}

// Exec runs a command inside the guest via
// POST /nodes/{node}/qemu/{vmid}/agent/exec (guest-exec). Returns the
// started process's PID; poll its outcome with Client.ExecStatus.
func (c *Client) Exec(ctx context.Context, vmid int, opts *ExecOptions) (*ExecResult, error) {
	if opts == nil {
		return nil, fmt.Errorf("agent: exec options are required")
	}
	if len(opts.Command) == 0 {
		return nil, fmt.Errorf("agent: exec command is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return nil, err
	}

	res := &ExecResult{}
	if err := c.client.Create(ctx, path(c.node, vmid, "exec"), res, p); err != nil {
		return nil, err
	}

	return res, nil
}

// ExecStatus describes a process started by Client.Exec, as returned by
// Client.ExecStatus.
type ExecStatus struct {
	// Exited is true once the process has finished.
	Exited bool `json:"exited,omitempty" url:"exited,omitempty"`
	// ExitCode is the process's exit code, once Exited and the process
	// terminated normally.
	ExitCode int `json:"exitcode,omitempty" url:"exitcode,omitempty"`
	// Signal is the signal number (or exception code) that terminated
	// the process abnormally, when applicable.
	Signal int `json:"signal,omitempty" url:"signal,omitempty"`
	// OutData is the process's captured stdout.
	OutData string `json:"out-data,omitempty" url:"out-data,omitempty"`
	// ErrData is the process's captured stderr.
	ErrData string `json:"err-data,omitempty" url:"err-data,omitempty"`
	// OutTruncated is true if OutData was not fully captured.
	OutTruncated bool `json:"out-truncated,omitempty" url:"out-truncated,omitempty"`
	// ErrTruncated is true if ErrData was not fully captured.
	ErrTruncated bool `json:"err-truncated,omitempty" url:"err-truncated,omitempty"`
}

// ExecStatus retrieves a process started by Client.Exec's status via
// GET /nodes/{node}/qemu/{vmid}/agent/exec-status
// (guest-exec-status).
func (c *Client) ExecStatus(ctx context.Context, vmid, pid int) (*ExecStatus, error) {
	p := map[string]string{"pid": fmt.Sprintf("%d", pid)}

	status := &ExecStatus{}
	if err := c.client.Get(ctx, path(c.node, vmid, "exec-status"), status, p); err != nil {
		return nil, err
	}

	return status, nil
}

// FileReadOptions holds the parameters for Client.FileRead
// (GET .../agent/file-read).
type FileReadOptions struct {
	// File is the path to the file to read, inside the guest.
	// Required.
	File string `url:"file"`
	// Count caps the number of bytes read; Proxmox defaults to and
	// enforces a 16 MiB maximum.
	Count int `url:"count,omitempty"`
	// Offset is the byte offset to start reading at.
	Offset int `url:"offset,omitempty"`
	// Decode controls whether the base64-encoded data the guest agent
	// returns is decoded before being returned here. Proxmox defaults
	// to true; pass a false pointer to receive it still base64-encoded.
	Decode *bool `url:"decode"`
}

// FileReadResult is a guest file's content, as returned by
// Client.FileRead.
type FileReadResult struct {
	// Content is the file's content — decoded, unless
	// FileReadOptions.Decode was set to false.
	Content string `json:"content,omitempty" url:"content,omitempty"`
	// Truncated is true if the read did not reach the end of the file
	// (e.g. Count was reached first).
	Truncated bool `json:"truncated,omitempty" url:"truncated,omitempty"`
}

// FileRead reads a file from inside the guest via
// GET /nodes/{node}/qemu/{vmid}/agent/file-read, capped at 16 MiB.
func (c *Client) FileRead(ctx context.Context, vmid int, opts *FileReadOptions) (*FileReadResult, error) {
	if opts == nil {
		return nil, fmt.Errorf("agent: file-read options are required")
	}
	if opts.File == "" {
		return nil, fmt.Errorf("agent: file-read file is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return nil, err
	}

	result := &FileReadResult{}
	if err := c.client.Get(ctx, path(c.node, vmid, "file-read"), result, p); err != nil {
		return nil, err
	}

	return result, nil
}

// FileWriteOptions holds the parameters for Client.FileWrite
// (POST .../agent/file-write).
type FileWriteOptions struct {
	// File is the path to the file to write, inside the guest.
	// Required.
	File string `url:"file"`
	// Content is the data to write, at most 60 KiB per call.
	// Required.
	Content string `url:"content"`
	// Encode controls whether Content is base64-encoded before being
	// sent to the guest agent (which requires base64). Proxmox defaults
	// to true; pass a false pointer only if Content is already
	// base64-encoded.
	Encode *bool `url:"encode"`
}

// FileWrite writes a file inside the guest via
// POST /nodes/{node}/qemu/{vmid}/agent/file-write.
func (c *Client) FileWrite(ctx context.Context, vmid int, opts *FileWriteOptions) error {
	if opts == nil {
		return fmt.Errorf("agent: file-write options are required")
	}
	if opts.File == "" {
		return fmt.Errorf("agent: file-write file is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return err
	}

	return c.client.Create(ctx, path(c.node, vmid, "file-write"), nil, p)
}
