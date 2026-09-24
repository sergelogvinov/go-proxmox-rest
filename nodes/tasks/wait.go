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
	"strings"
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
// while the task is still running, or its outcome still can't be
// determined (see isNoSuchTask/taskFromHistory below) — unwraps to
// context.DeadlineExceeded or context.Canceled via errors.Is, same as a
// bare ctx timeout would; or whatever error the underlying Status call
// itself returns (e.g. a transient network error, or proxmox.IsNotFound
// for an unknown upid).
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
		switch {
		case err != nil && isNoSuchTask(err):
			if done, herr := c.taskFromHistory(ctx, upid); done {
				return herr
			}
			// Not found in history either (yet) — keep polling below.
		case err != nil:
			return err
		case status.Status != StateRunning:
			if status.ExitStatus != "" && status.ExitStatus != taskExitOK {
				return &FailedError{UPID: upid, ExitStatus: status.ExitStatus}
			}

			return nil
		default:
			// Still running; keep polling below.
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("tasks: waiting for %s: %w", upid, ctx.Err())
		case <-ticker.C:
		}
	}
}

// isNoSuchTask reports whether err is Proxmox's "no such task" response
// from Status. A UPID returned by a just-started task can briefly be
// unqueryable — the task exists but pvedaemon hasn't indexed it for live
// status lookups yet — which especially bites a task fast enough to
// finish before Wait's very first poll (e.g. a small file upload: by the
// time Status is asked, the task has already both started and finished,
// and Status may never have had a window in which to see it). Wait falls
// back to the node's task history rather than failing a request that in
// fact succeeded — see taskFromHistory.
func isNoSuchTask(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such task")
}

// taskFromHistory looks upid up via List (which, unlike Status, also
// covers tasks too short-lived for Status to ever have found) as a
// fallback for Status reporting "no such task". done reports whether a
// *finished* task entry was found; err then mirrors Status's own
// FailedError/nil outcome and must be returned as-is. done is false
// (err always nil) when the task isn't in the list yet, is still
// running, or the List call itself failed — Wait keeps polling in every
// such case, since List is a best-effort fallback, not the source of
// truth Status is.
//
// Lists against upid's own node (see taskNode/Status's doc comment) —
// c.node itself may not even be the node that ran this task.
func (c *Client) taskFromHistory(ctx context.Context, upid string) (done bool, err error) {
	scoped := c
	if node := taskNode(c.node, upid); node != c.node {
		scoped = &Client{client: c.client, node: node}
	}

	list, err := scoped.List(ctx, &ListOptions{Source: SourceAll, Limit: 50})
	if err != nil {
		return false, nil
	}

	for _, t := range list {
		if t.UPID != upid || t.EndTime == 0 {
			continue
		}

		if t.Status != "" && t.Status != taskExitOK {
			return true, &FailedError{UPID: upid, ExitStatus: t.Status}
		}

		return true, nil
	}

	return false, nil
}
