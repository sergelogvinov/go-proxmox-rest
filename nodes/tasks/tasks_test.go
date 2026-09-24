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
	"testing"
)

// pathRecordingGetter records the last path it was asked to Get/Delete,
// so tests can assert which node a request was actually addressed to.
type pathRecordingGetter struct {
	gotPath string
}

func (p *pathRecordingGetter) Get(_ context.Context, path string, _ any, _ map[string]string) error {
	p.gotPath = path
	return nil
}

func (p *pathRecordingGetter) Delete(_ context.Context, path string, _ any, _ map[string]string) error {
	p.gotPath = path
	return nil
}

// TestTaskNode guards the fix for a real bug: a task's UPID doesn't
// necessarily name the node its triggering request was addressed to
// (e.g. a storage upload's imgcopy task can run on whichever node
// actually hosts that storage). Per-task endpoints must address the
// UPID's own embedded node, or Proxmox rejects the request outright
// ("Parameter verification failed") or reports "no such task".
func TestTaskNode(t *testing.T) {
	tests := []struct {
		name        string
		defaultNode string
		upid        string
		want        string
	}{
		{
			name:        "upid names a different node than the client",
			defaultNode: "rnd-3",
			upid:        "UPID:rnd-2:001523F4:00ECA3F2:6AB4F552:imgcopy::root@pam:",
			want:        "rnd-2",
		},
		{
			name:        "upid names the same node as the client",
			defaultNode: "rnd-3",
			upid:        "UPID:rnd-3:000BB898:008F23B9:6AB4F545:qmclone:1003:root@pam:",
			want:        "rnd-3",
		},
		{
			name:        "malformed upid falls back to the client's node",
			defaultNode: "rnd-3",
			upid:        "not-a-upid",
			want:        "rnd-3",
		},
		{
			name:        "empty upid falls back to the client's node",
			defaultNode: "rnd-3",
			upid:        "",
			want:        "rnd-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := taskNode(tt.defaultNode, tt.upid); got != tt.want {
				t.Fatalf("taskNode(%q, %q) = %q, want %q", tt.defaultNode, tt.upid, got, tt.want)
			}
		})
	}
}

const mismatchedUPID = "UPID:rnd-2:001523F4:00ECA3F2:6AB4F552:imgcopy::root@pam:"

func TestStatusUsesUPIDNode(t *testing.T) {
	getter := &pathRecordingGetter{}
	c := New(getter, "rnd-3")

	if _, err := c.Status(context.Background(), mismatchedUPID); err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if want := "/nodes/rnd-2/tasks/" + mismatchedUPID + "/status"; getter.gotPath != want {
		t.Fatalf("Status() requested %q, want %q (upid's own node, not the client's rnd-3)", getter.gotPath, want)
	}
}

func TestLogUsesUPIDNode(t *testing.T) {
	getter := &pathRecordingGetter{}
	c := New(getter, "rnd-3")

	if _, err := c.Log(context.Background(), mismatchedUPID, nil); err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	if want := "/nodes/rnd-2/tasks/" + mismatchedUPID + "/log"; getter.gotPath != want {
		t.Fatalf("Log() requested %q, want %q (upid's own node, not the client's rnd-3)", getter.gotPath, want)
	}
}

func TestStopUsesUPIDNode(t *testing.T) {
	getter := &pathRecordingGetter{}
	c := New(getter, "rnd-3")

	if err := c.Stop(context.Background(), mismatchedUPID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if want := "/nodes/rnd-2/tasks/" + mismatchedUPID; getter.gotPath != want {
		t.Fatalf("Stop() requested %q, want %q (upid's own node, not the client's rnd-3)", getter.gotPath, want)
	}
}

// TestListUsesClientNode ensures List — which has no upid to derive a
// node from — is untouched by the taskNode fix and still lists the
// client's own node's task history.
func TestListUsesClientNode(t *testing.T) {
	getter := &pathRecordingGetter{}
	c := New(getter, "rnd-3")

	if _, err := c.List(context.Background(), nil); err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if want := "/nodes/rnd-3/tasks"; getter.gotPath != want {
		t.Fatalf("List() requested %q, want %q", getter.gotPath, want)
	}
}
