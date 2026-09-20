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
	want := []string{"prod", "web", "team-a"}
	if !reflect.DeepEqual(cfg.Tags.Tags, want) {
		t.Fatalf("Tags.Tags = %#v, want %#v", cfg.Tags.Tags, want)
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
	p, err := encodeConfig(&Config{Tags: &Tags{Tags: []string{"prod", "web", "team-a"}}})
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
