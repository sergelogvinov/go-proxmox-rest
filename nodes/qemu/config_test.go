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
	"encoding/json"
	"reflect"
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// decodeConfig decodes raw the way Client.Config does: the root client
// hands the response's data object to params.Decode.
func decodeConfig(raw map[string]json.RawMessage) (*Config, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := params.Decode(b, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func TestDecodeConfigTags(t *testing.T) {
	raw := map[string]json.RawMessage{
		"tags": json.RawMessage(`"prod;web;team-a"`),
	}

	cfg, err := decodeConfig(raw)
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}

	if cfg.Tags == nil {
		t.Fatal("Tags = nil, want populated")
	}
	want := Tags{"prod", "web", "team-a"}
	if !reflect.DeepEqual(*cfg.Tags, want) {
		t.Fatalf("Tags = %#v, want %#v", *cfg.Tags, want)
	}
}

func TestDecodeConfigNoTags(t *testing.T) {
	cfg, err := decodeConfig(map[string]json.RawMessage{})
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}
	if cfg.Tags != nil {
		t.Fatalf("Tags = %#v, want nil", cfg.Tags)
	}
}

func TestEncodeConfigTags(t *testing.T) {
	p, err := encodeConfig(&Config{Tags: &Tags{"prod", "web", "team-a"}})
	if err != nil {
		t.Fatalf("encodeConfig() error = %v", err)
	}

	if got, want := p["tags"], "prod;web;team-a"; got != want {
		t.Fatalf("params[\"tags\"] = %q, want %q", got, want)
	}
}

func TestEncodeConfigNoTags(t *testing.T) {
	p, err := encodeConfig(&Config{})
	if err != nil {
		t.Fatalf("encodeConfig() error = %v", err)
	}
	if _, ok := p["tags"]; ok {
		t.Fatalf("params contains %q for a nil Tags: %v", "tags", p)
	}
}

// TestDecodeConfigNet guards against config decoding
// dropping the model=macaddr alias — property.Unmarshal alone can't
// recognize it since the model name ("virtio") is a dynamic key, not one
// of Net's cfg tags. A regression here silently strips Model/MACAddr, so
// a decoded config re-sent through UpdateConfig loses the required
// "netN.model" property.
func TestDecodeConfigNet(t *testing.T) {
	raw := map[string]json.RawMessage{
		"net0": json.RawMessage(`"virtio=BC:24:11:CD:B9:41,bridge=vmbr0"`),
	}

	cfg, err := decodeConfig(raw)
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}

	got, ok := cfg.Net[0]
	if !ok {
		t.Fatal("Net[0] missing, want populated")
	}

	want := Net{Model: "virtio", MACAddr: "BC:24:11:CD:B9:41", Bridge: "vmbr0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Net[0] = %#v, want %#v", got, want)
	}

	if got := got.String(); got != "virtio=BC:24:11:CD:B9:41,bridge=vmbr0" {
		t.Fatalf("Net[0].String() = %q, want round-trip of the original value", got)
	}
}

// TestConfigIndexedRoundTrip checks that every indexed family survives
// decode -> encode unchanged, and that families sharing a prefix
// (virtio/virtiofs) and indexed-looking scalars (efidisk0, smbios1)
// don't claim each other's keys.
func TestConfigIndexedRoundTrip(t *testing.T) {
	wire := map[string]string{
		"net0":      "virtio=BC:24:11:CD:B9:41,bridge=vmbr0",
		"ide2":      "local:iso/debian.iso,media=cdrom",
		"sata1":     "local-lvm:vm-100-disk-2,size=8G",
		"scsi0":     "local-lvm:vm-100-disk-0,size=32G",
		"virtio3":   "local-lvm:vm-100-disk-1,size=4G",
		"virtiofs0": "share",
		"unused0":   "local-lvm:vm-100-disk-9",
		"usb1":      "1234:5678",
		"hostpci0":  "0000:01:00.0",
		"serial0":   "socket",
		"parallel1": "/dev/parport0",
		"ipconfig0": "ip=dhcp",
		"numa0":     "cpus=0-3,memory=2048",
		"efidisk0":  "local-lvm:vm-100-disk-3,efitype=4m",
	}

	raw := map[string]json.RawMessage{}
	for k, v := range wire {
		b, _ := json.Marshal(v) //nolint:errchkjson
		raw[k] = b
	}

	cfg, err := decodeConfig(raw)
	if err != nil {
		t.Fatalf("decodeConfig() error = %v", err)
	}

	if len(cfg.VirtIO) != 1 || len(cfg.VirtioFS) != 1 {
		t.Fatalf("VirtIO = %v, VirtioFS = %v; want one entry each", cfg.VirtIO, cfg.VirtioFS)
	}

	if got := cfg.Parallel[1]; got != "/dev/parport0" {
		t.Fatalf("Parallel[1] = %q, want %q", got, "/dev/parport0")
	}

	p, err := encodeConfig(cfg)
	if err != nil {
		t.Fatalf("encodeConfig() error = %v", err)
	}

	if !reflect.DeepEqual(p, wire) {
		t.Fatalf("round trip mismatch:\n got  %v\n want %v", p, wire)
	}
}
