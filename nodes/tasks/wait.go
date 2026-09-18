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

package tasks

import (
	"context"
	"fmt"
	"time"
)

const (
	// defaultWaitPollInterval is Client.Wait's poll interval when
	// WaitOptions is nil or its PollInterval is zero.
	defaultWaitPollInterval = 5 * time.Second

	// defaultWaitTimeout is Client.Wait's timeout when WaitOptions is nil
	// or its Timeout is zero. Deliberately short (Proxmox's own quick
	// actions finish well within it); callers waiting on a long-running
	// task (a migration, a large clone, a disk resize, ...) should pass
	// an explicit, longer Timeout.
	defaultWaitTimeout = 30 * time.Second

	// taskExitOK is the ExitStatus string Proxmox reports for a task
	// that completed successfully.
	taskExitOK = "OK"
)

// WaitOptions configures Client.Wait.
type WaitOptions struct {
	// PollInterval is how often Wait re-checks the task's status via
	// Client.Status. Zero uses defaultWaitPollInterval (5s).
	PollInterval time.Duration
	// Timeout bounds how long Wait polls before giving up on a task
	// still running. Zero uses defaultWaitTimeout (30s).
	Timeout time.Duration
}

// FailedError is Client.Wait's error when a task finishes with an
// ExitStatus other than "OK" (e.g. an errored or manually stopped task).
// Use errors.As to recover UPID/ExitStatus from a Wait error.
type FailedError struct {
	// UPID is the task that failed.
	UPID string
	// ExitStatus is the task's non-"OK" exit status string, e.g. an
	// error message or "task stopped".
	ExitStatus string
}

// Error implements the error interface.
func (e *FailedError) Error() string {
	return fmt.Sprintf("tasks: task %s failed: %s", e.UPID, e.ExitStatus)
}

// Wait blocks until the task named by upid stops running, polling its
// status via Client.Status every opts.PollInterval. opts may be nil to
// use the defaults (5s poll interval, 30s timeout) — pass an explicit
// Timeout for any task expected to run longer than that.
//
// Returns a *FailedError if the task finished with an ExitStatus other
// than "OK"; a wrapped context error if ctx or opts.Timeout is exceeded
// while the task is still running (unwraps to context.DeadlineExceeded
// or context.Canceled via errors.Is, same as a bare ctx timeout would);
// or whatever error the underlying Status call itself returns (e.g. a
// transient network error, or proxmox.IsNotFound for an unknown upid).
func (c *Client) Wait(ctx context.Context, upid string, opts *WaitOptions) error {
	interval := defaultWaitPollInterval
	timeout := defaultWaitTimeout

	if opts != nil {
		if opts.PollInterval > 0 {
			interval = opts.PollInterval
		}
		if opts.Timeout > 0 {
			timeout = opts.Timeout
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		status, err := c.Status(ctx, upid)
		if err != nil {
			return err
		}

		if status.Status != StateRunning {
			if status.ExitStatus != "" && status.ExitStatus != taskExitOK {
				return &FailedError{UPID: upid, ExitStatus: status.ExitStatus}
			}

			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("tasks: waiting for %s: %w", upid, ctx.Err())
		case <-ticker.C:
		}
	}
}
