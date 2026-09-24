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
)

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

// TestDecodeConfigNet guards against decodeConfig (via setIndexedNet)
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
