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

import "testing"

func TestAttachDriveValidation(t *testing.T) {
	c := New(&recordingGetter{}, "pve1")
	ctx := t.Context()

	if _, err := c.AttachDrive(ctx, 100, nil); err == nil {
		t.Error("AttachDrive with nil options: got nil error, want one")
	}
	if _, err := c.AttachDrive(ctx, 100, &AttachDriveOptions{Drive: "scsi0"}); err == nil {
		t.Error("AttachDrive with no volume: got nil error, want one")
	}
	if _, err := c.AttachDrive(ctx, 100, &AttachDriveOptions{Options: Drive{File: "local-lvm:vm-100-disk-0"}}); err == nil {
		t.Error("AttachDrive with no drive slot: got nil error, want one")
	}
	if _, err := c.AttachDrive(ctx, 100, &AttachDriveOptions{Drive: "net0", Options: Drive{File: "local-lvm:vm-100-disk-0"}}); err == nil {
		t.Error("AttachDrive on a non-drive slot: got nil error, want one")
	}
	if _, err := c.AttachDrive(ctx, 100, &AttachDriveOptions{Drive: "bogus", Options: Drive{File: "local-lvm:vm-100-disk-0"}}); err == nil {
		t.Error("AttachDrive with an unparseable drive: got nil error, want one")
	}
}

func TestAttachDriveRequest(t *testing.T) {
	for _, tc := range []struct {
		drive string
		field string
	}{
		{"ide1", "ide1"},
		{"sata3", "sata3"},
		{"scsi0", "scsi0"},
		{"virtio2", "virtio2"},
	} {
		g := &recordingGetter{}
		c := New(g, "pve1")

		ssd := true
		upid, err := c.AttachDrive(t.Context(), 100, &AttachDriveOptions{
			Drive:   tc.drive,
			Options: Drive{File: "local-lvm:vm-100-disk-0", SSD: &ssd},
		})
		if err != nil {
			t.Fatalf("AttachDrive(%q): unexpected error: %v", tc.drive, err)
		}
		if upid != "UPID:fake" {
			t.Errorf("AttachDrive(%q): upid = %q, want %q", tc.drive, upid, "UPID:fake")
		}

		const want = "/nodes/pve1/qemu/100/config"
		if g.path != want {
			t.Errorf("AttachDrive(%q): path = %q, want %q", tc.drive, g.path, want)
		}

		got, ok := g.params[tc.field]
		if !ok {
			t.Fatalf("AttachDrive(%q): params %v missing field %q", tc.drive, g.params, tc.field)
		}
		if want := "local-lvm:vm-100-disk-0,ssd=1"; got != want {
			t.Errorf("AttachDrive(%q): params[%q] = %q, want %q", tc.drive, tc.field, got, want)
		}
	}
}

func TestDetachDriveValidation(t *testing.T) {
	c := New(&recordingGetter{}, "pve1")
	ctx := t.Context()

	if _, err := c.DetachDrive(ctx, 100, nil); err == nil {
		t.Error("DetachDrive with nil options: got nil error, want one")
	}
	if _, err := c.DetachDrive(ctx, 100, &DetachDriveOptions{}); err == nil {
		t.Error("DetachDrive with no drive: got nil error, want one")
	}
	if _, err := c.DetachDrive(ctx, 100, &DetachDriveOptions{Drive: "net0"}); err == nil {
		t.Error("DetachDrive on a non-drive slot: got nil error, want one")
	}
	if _, err := c.DetachDrive(ctx, 100, &DetachDriveOptions{Drive: "bogus"}); err == nil {
		t.Error("DetachDrive with an unparseable drive: got nil error, want one")
	}
}

func TestDetachDriveRequest(t *testing.T) {
	g := &recordingGetter{}
	c := New(g, "pve1")

	upid, err := c.DetachDrive(t.Context(), 100, &DetachDriveOptions{Drive: "scsi0", Force: true})
	if err != nil {
		t.Fatalf("DetachDrive: unexpected error: %v", err)
	}
	if upid != "UPID:fake" {
		t.Errorf("DetachDrive: upid = %q, want %q", upid, "UPID:fake")
	}

	if got := g.params["delete"]; got != "scsi0" {
		t.Errorf("DetachDrive: params[\"delete\"] = %q, want %q", got, "scsi0")
	}
	if got := g.params["force"]; got != "1" {
		t.Errorf("DetachDrive: params[\"force\"] = %q, want %q", got, "1")
	}
}
