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

package qemu

import (
	"context"
	"testing"
)

// recordingGetter is a minimal Getter that records the last Create call
// (UpdateConfigAsync's transport) instead of making any request.
type recordingGetter struct {
	path   string
	params map[string]string
}

func (g *recordingGetter) Get(context.Context, string, any, map[string]string) error { return nil }

func (g *recordingGetter) Create(_ context.Context, path string, out any, params map[string]string) error {
	g.path = path
	g.params = params
	if upid, ok := out.(*string); ok {
		*upid = "UPID:fake"
	}
	return nil
}

func (g *recordingGetter) Update(context.Context, string, any, map[string]string) error { return nil }

func (g *recordingGetter) Delete(context.Context, string, any, map[string]string) error { return nil }

func TestAttachISOValidation(t *testing.T) {
	c := New(&recordingGetter{}, "pve1")
	ctx := t.Context()

	if _, err := c.AttachISO(ctx, 100, nil); err == nil {
		t.Error("AttachISO with nil options: got nil error, want one")
	}
	if _, err := c.AttachISO(ctx, 100, &AttachISOOptions{Drive: "ide2"}); err == nil {
		t.Error("AttachISO with no volume: got nil error, want one")
	}
	if _, err := c.AttachISO(ctx, 100, &AttachISOOptions{Volume: "local:iso/debian.iso"}); err == nil {
		t.Error("AttachISO with no drive: got nil error, want one")
	}
	if _, err := c.AttachISO(ctx, 100, &AttachISOOptions{Drive: "virtio0", Volume: "local:iso/debian.iso"}); err == nil {
		t.Error("AttachISO on a virtio slot: got nil error, want one")
	}
	if _, err := c.AttachISO(ctx, 100, &AttachISOOptions{Drive: "bogus", Volume: "local:iso/debian.iso"}); err == nil {
		t.Error("AttachISO with an unparseable drive: got nil error, want one")
	}
}

func TestAttachISORequest(t *testing.T) {
	for _, tc := range []struct {
		drive string
		field string
	}{
		{"ide2", "ide2"},
		{"sata1", "sata1"},
		{"scsi3", "scsi3"},
	} {
		g := &recordingGetter{}
		c := New(g, "pve1")

		upid, err := c.AttachISO(t.Context(), 100, &AttachISOOptions{Drive: tc.drive, Volume: "local:iso/debian.iso"})
		if err != nil {
			t.Fatalf("AttachISO(%q): unexpected error: %v", tc.drive, err)
		}
		if upid != "UPID:fake" {
			t.Errorf("AttachISO(%q): upid = %q, want %q", tc.drive, upid, "UPID:fake")
		}

		const want = "/nodes/pve1/qemu/100/config"
		if g.path != want {
			t.Errorf("AttachISO(%q): path = %q, want %q", tc.drive, g.path, want)
		}

		got, ok := g.params[tc.field]
		if !ok {
			t.Fatalf("AttachISO(%q): params %v missing field %q", tc.drive, g.params, tc.field)
		}
		if want := "local:iso/debian.iso,media=cdrom"; got != want {
			t.Errorf("AttachISO(%q): params[%q] = %q, want %q", tc.drive, tc.field, got, want)
		}
	}
}

func TestDetachISO(t *testing.T) {
	g := &recordingGetter{}
	c := New(g, "pve1")

	if _, err := c.DetachISO(t.Context(), 100, ""); err == nil {
		t.Error("DetachISO with no drive: got nil error, want one")
	}

	upid, err := c.DetachISO(t.Context(), 100, "ide2")
	if err != nil {
		t.Fatalf("DetachISO: unexpected error: %v", err)
	}
	if upid != "UPID:fake" {
		t.Errorf("DetachISO: upid = %q, want %q", upid, "UPID:fake")
	}

	const want = "none,media=cdrom"
	if got := g.params["ide2"]; got != want {
		t.Errorf("DetachISO: params[\"ide2\"] = %q, want %q", got, want)
	}
}
