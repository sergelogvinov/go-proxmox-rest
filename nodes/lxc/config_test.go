package lxc

import (
	"encoding/json"
	"testing"
)

func TestSplitIndexedKey(t *testing.T) {
	tests := []struct {
		key        string
		wantPrefix string
		wantIndex  int
		wantOK     bool
	}{
		{"net0", "net", 0, true},
		{"net31", "net", 31, true},
		{"mp0", "mp", 0, true},
		{"mp255", "mp", 255, true},
		{"unused255", "unused", 255, true},
		// rootfs is a fixed, singular field name, not part of a
		// growing indexed family, and has no trailing digit anyway.
		{"rootfs", "", 0, false},
		// No trailing digits at all.
		{"cores", "", 0, false},
		{"hostname", "", 0, false},
	}

	for _, tt := range tests {
		prefix, index, ok := splitIndexedKey(tt.key)
		if ok != tt.wantOK || (ok && (prefix != tt.wantPrefix || index != tt.wantIndex)) {
			t.Errorf("splitIndexedKey(%q) = (%q, %d, %v), want (%q, %d, %v)",
				tt.key, prefix, index, ok, tt.wantPrefix, tt.wantIndex, tt.wantOK)
		}
	}
}

func TestDecodeConfig(t *testing.T) {
	raw := map[string]json.RawMessage{
		"hostname": json.RawMessage(`"test-ct"`),
		"cores":    json.RawMessage(`"2"`),
		"onboot":   json.RawMessage(`"1"`),
		"digest":   json.RawMessage(`"abc123"`),
		"rootfs":   json.RawMessage(`"local-lvm:vm-100-disk-0,size=8G"`),
		"net0":     json.RawMessage(`"name=eth0,bridge=vmbr0,ip=dhcp"`),
		"net1":     json.RawMessage(`"name=eth1,bridge=vmbr1,ip=dhcp"`),
		"mp0":      json.RawMessage(`"local-lvm:vm-100-disk-1,mp=/data"`),
		"unused3":  json.RawMessage(`"local-lvm:vm-100-disk-3"`),
	}

	cfg, err := decodeConfig(raw)
	if err != nil {
		t.Fatalf("decodeConfig: unexpected error: %v", err)
	}

	if cfg.Hostname != "test-ct" {
		t.Errorf("Hostname = %q, want %q", cfg.Hostname, "test-ct")
	}
	if cfg.Cores != 2 {
		t.Errorf("Cores = %d, want 2", cfg.Cores)
	}
	if !cfg.OnBoot {
		t.Errorf("OnBoot = false, want true")
	}
	if cfg.Digest != "abc123" {
		t.Errorf("Digest = %q, want %q", cfg.Digest, "abc123")
	}
	if cfg.RootFS != "local-lvm:vm-100-disk-0,size=8G" {
		t.Errorf("RootFS = %q, want %q", cfg.RootFS, "local-lvm:vm-100-disk-0,size=8G")
	}

	if got := cfg.Net[0]; got != "name=eth0,bridge=vmbr0,ip=dhcp" {
		t.Errorf("Net[0] = %q, want %q", got, "name=eth0,bridge=vmbr0,ip=dhcp")
	}
	if got := cfg.Net[1]; got != "name=eth1,bridge=vmbr1,ip=dhcp" {
		t.Errorf("Net[1] = %q, want %q", got, "name=eth1,bridge=vmbr1,ip=dhcp")
	}
	if len(cfg.Net) != 2 {
		t.Errorf("len(Net) = %d, want 2", len(cfg.Net))
	}
	if got := cfg.MP[0]; got != "local-lvm:vm-100-disk-1,mp=/data" {
		t.Errorf("MP[0] = %q, want %q", got, "local-lvm:vm-100-disk-1,mp=/data")
	}
	if got := cfg.Unused[3]; got != "local-lvm:vm-100-disk-3" {
		t.Errorf("Unused[3] = %q, want %q", got, "local-lvm:vm-100-disk-3")
	}
}

func TestEncodeConfig(t *testing.T) {
	cfg := &Config{
		Hostname: "test-ct",
		Cores:    2,
		Net:      map[int]string{0: "name=eth0,bridge=vmbr0,ip=dhcp"},
		MP:       map[int]string{0: "local-lvm:8"},
		Delete:   []string{"net1", "description"},
	}

	p, err := encodeConfig(cfg)
	if err != nil {
		t.Fatalf("encodeConfig: unexpected error: %v", err)
	}

	want := map[string]string{
		"hostname": "test-ct",
		"cores":    "2",
		"net0":     "name=eth0,bridge=vmbr0,ip=dhcp",
		"mp0":      "local-lvm:8",
		"delete":   "net1,description",
	}
	for k, v := range want {
		if p[k] != v {
			t.Errorf("p[%q] = %q, want %q", k, p[k], v)
		}
	}
	if len(p) != len(want) {
		t.Errorf("encodeConfig produced %d params, want %d: %+v", len(p), len(want), p)
	}
}

func TestEncodeConfigNil(t *testing.T) {
	if _, err := encodeConfig(nil); err == nil {
		t.Errorf("encodeConfig(nil): expected an error, got nil")
	}
}

// TestEncodeConfigReadonlyFieldsExcluded verifies that LXC (GET-only
// diagnostic info) and the snapshot-only fields (Parent, SnapTime) are
// never sent, even when populated — the common case being a caller
// round-tripping a fetched Config back through UpdateConfig.
func TestEncodeConfigReadonlyFieldsExcluded(t *testing.T) {
	cfg := &Config{
		LXC:      [][]string{{"lxc.arch", "amd64"}},
		Parent:   "pre-upgrade",
		SnapTime: 1700000000,
	}

	p, err := encodeConfig(cfg)
	if err != nil {
		t.Fatalf("encodeConfig: unexpected error: %v", err)
	}
	if len(p) != 0 {
		t.Errorf("encodeConfig sent readonly fields: %+v", p)
	}
}
