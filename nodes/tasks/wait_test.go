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
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeGetter feeds Client.Status and Client.List independent, scripted
// response sequences (one stage per call; the last stage repeats for any
// extra calls), routing on the request path. A List call with no scripted
// stages returns an empty list — "not (yet) in history" — matching the
// default assumption in tests that only exercise Status.
type fakeGetter struct {
	statusCalls  int
	statusStages []func(out any) error

	listCalls  int
	listStages []func(out any) error
}

func (f *fakeGetter) Get(_ context.Context, path string, out any, _ map[string]string) error {
	if strings.HasSuffix(path, "/status") {
		i := f.statusCalls
		if i >= len(f.statusStages) {
			i = len(f.statusStages) - 1
		}
		f.statusCalls++

		return f.statusStages[i](out)
	}

	f.listCalls++
	if len(f.listStages) == 0 {
		*out.(*[]Task) = nil //nolint:errcheck
		return nil
	}

	i := f.listCalls - 1
	if i >= len(f.listStages) {
		i = len(f.listStages) - 1
	}

	return f.listStages[i](out)
}

func (f *fakeGetter) Delete(context.Context, string, any, map[string]string) error {
	return nil
}

const testUPID = "UPID:node1:00000001:00000001:00000000:upload:100:root@pam:"

// TestWaitRecoversFromNoSuchTask guards against a regression where Wait
// failed a request that actually succeeded: a UPID can briefly be
// unqueryable right after a task starts (pvedaemon hasn't indexed it for
// status lookups yet), especially for a task fast enough to finish before
// Wait's very first poll — a small file upload, say. Wait must treat that
// "no such task" response the same as a still-running task and keep
// polling, not fail immediately — here, history has no record of it
// either (yet), so Wait must fall through to its next Status poll rather
// than trusting the empty list.
func TestWaitRecoversFromNoSuchTask(t *testing.T) {
	getter := &fakeGetter{
		statusStages: []func(out any) error{
			func(_ any) error {
				return errors.New("proxmox API error 400: Bad Request: map[upid:no such task]")
			},
			func(out any) error {
				*out.(*Status) = Status{Status: StateStopped, ExitStatus: taskExitOK}
				return nil
			},
		},
	}

	c := New(getter, "node1")

	err := c.Wait(context.Background(), testUPID, &WaitOptions{PollInterval: 10 * time.Millisecond, Timeout: time.Second})
	if err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}
	if getter.statusCalls < 2 {
		t.Fatalf("Status called %d time(s), want >= 2 (should have polled through the no-such-task race)", getter.statusCalls)
	}
}

// TestWaitRecoversFromNoSuchTaskViaHistory covers the case that motivated
// taskFromHistory: a task so fast it finishes before Status ever has a
// window to see it, so Status reports "no such task" for as long as Wait
// keeps asking (it never becomes queryable) — previously this meant Wait
// always ran out the full Timeout and returned a misleading "context
// deadline exceeded" for a request that had in fact already succeeded.
// The node's task history still has the finished entry, and Wait must
// use it instead of hanging until Timeout.
func TestWaitRecoversFromNoSuchTaskViaHistory(t *testing.T) {
	getter := &fakeGetter{
		statusStages: []func(out any) error{
			func(_ any) error {
				return errors.New("proxmox API error 400: Bad Request: map[upid:no such task]")
			},
		},
		listStages: []func(out any) error{
			func(out any) error {
				*out.(*[]Task) = []Task{{UPID: testUPID, EndTime: 1000, Status: taskExitOK}}
				return nil
			},
		},
	}

	c := New(getter, "node1")

	start := time.Now()

	err := c.Wait(context.Background(), testUPID, &WaitOptions{PollInterval: 200 * time.Millisecond, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}
	if elapsed := time.Since(start); elapsed >= 2*time.Second {
		t.Fatalf("Wait() took %s, want it to resolve via history well before the 2s Timeout", elapsed)
	}
}

// TestWaitFailedTaskViaHistory mirrors TestWaitRecoversFromNoSuchTaskViaHistory
// for a task history shows as failed, confirming taskFromHistory's outcome
// (not just its presence) is honored.
func TestWaitFailedTaskViaHistory(t *testing.T) {
	getter := &fakeGetter{
		statusStages: []func(out any) error{
			func(_ any) error {
				return errors.New("proxmox API error 400: Bad Request: map[upid:no such task]")
			},
		},
		listStages: []func(out any) error{
			func(out any) error {
				*out.(*[]Task) = []Task{{UPID: testUPID, EndTime: 1000, Status: "some error"}}
				return nil
			},
		},
	}

	c := New(getter, "node1")

	err := c.Wait(context.Background(), testUPID, &WaitOptions{PollInterval: 200 * time.Millisecond, Timeout: 2 * time.Second})

	var failedErr *FailedError
	if !errors.As(err, &failedErr) {
		t.Fatalf("Wait() error = %v, want *FailedError", err)
	}
	if failedErr.ExitStatus != "some error" {
		t.Fatalf("FailedError.ExitStatus = %q, want %q", failedErr.ExitStatus, "some error")
	}
}

// TestWaitPropagatesOtherErrors ensures the no-such-task leniency doesn't
// swallow every Status error — an unrelated failure (network error,
// permission error, ...) must still fail Wait immediately.
func TestWaitPropagatesOtherErrors(t *testing.T) {
	wantErr := errors.New("boom")
	getter := &fakeGetter{
		statusStages: []func(out any) error{
			func(_ any) error { return wantErr },
		},
	}

	c := New(getter, "node1")

	err := c.Wait(context.Background(), testUPID, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Wait() error = %v, want %v", err, wantErr)
	}
	if getter.statusCalls != 1 {
		t.Fatalf("Status called %d time(s), want exactly 1 (must not retry a non-no-such-task error)", getter.statusCalls)
	}
}

// TestWaitFailedTask ensures a task that finishes with a non-OK exit
// status still surfaces as *FailedError, unaffected by the no-such-task
// handling above.
func TestWaitFailedTask(t *testing.T) {
	getter := &fakeGetter{
		statusStages: []func(out any) error{
			func(out any) error {
				*out.(*Status) = Status{Status: StateStopped, ExitStatus: "some error"}
				return nil
			},
		},
	}

	c := New(getter, "node1")

	err := c.Wait(context.Background(), testUPID, nil)

	var failedErr *FailedError
	if !errors.As(err, &failedErr) {
		t.Fatalf("Wait() error = %v, want *FailedError", err)
	}
	if failedErr.ExitStatus != "some error" {
		t.Fatalf("FailedError.ExitStatus = %q, want %q", failedErr.ExitStatus, "some error")
	}
}
