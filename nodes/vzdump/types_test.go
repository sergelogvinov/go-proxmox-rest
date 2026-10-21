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

package vzdump

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

func TestOptionsDecodeDefaults(t *testing.T) {
	// GET /nodes/{node}/vzdump/defaults returns property-string fields as
	// parsed objects.
	data := []byte(`{
		"storage": "backup",
		"performance": {"max-workers": 16, "pbs-entries-max": 1048576},
		"fleecing": {"enabled": 1, "storage": "local-lvm"},
		"prune-backups": {"keep-last": 3, "keep-daily": 7}
	}`)

	var opts Options
	if err := params.Decode(data, &opts); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if opts.Performance != "max-workers=16,pbs-entries-max=1048576" {
		t.Errorf("Performance = %q", opts.Performance)
	}
	if opts.Fleecing != "enabled=1,storage=local-lvm" {
		t.Errorf("Fleecing = %q", opts.Fleecing)
	}
	if opts.PruneBackups != "keep-daily=7,keep-last=3" {
		t.Errorf("PruneBackups = %q", opts.PruneBackups)
	}
}

func TestOptionsDecodePropertyStrings(t *testing.T) {
	data := []byte(`{"performance": "max-workers=8", "prune-backups": "keep-all=1"}`)

	var opts Options
	if err := params.Decode(data, &opts); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if opts.Performance != "max-workers=8" || opts.PruneBackups != "keep-all=1" {
		t.Errorf("unexpected options: %+v", opts)
	}
}

func TestOptionsEncodePropertyStrings(t *testing.T) {
	p, err := params.Encode(&Options{All: true, Performance: "max-workers=8"})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if p["performance"] != "max-workers=8" {
		t.Errorf("performance = %q, want %q", p["performance"], "max-workers=8")
	}
}
