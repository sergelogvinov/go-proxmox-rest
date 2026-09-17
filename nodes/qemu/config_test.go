package qemu

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
		{"hostpci15", "hostpci", 15, true},
		{"scsi30", "scsi", 30, true},
		{"unused255", "unused", 255, true},
		// These end in a digit but aren't part of a growing indexed
		// family — each is a fixed, singular field name.
		{"smbios1", "", 0, false},
		{"efidisk0", "", 0, false},
		{"tpmstate0", "", 0, false},
		{"rng0", "", 0, false},
		{"audio0", "", 0, false},
		// No trailing digits at all.
		{"numa", "", 0, false},
		{"cores", "", 0, false},
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
		"name":     json.RawMessage(`"test-vm"`),
		"cores":    json.RawMessage(`"4"`),
		"numa":     json.RawMessage(`"1"`),
		"digest":   json.RawMessage(`"abc123"`),
		"smbios1":  json.RawMessage(`"uuid=deadbeef"`),
		"net0":     json.RawMessage(`"virtio=AA:BB,bridge=vmbr0"`),
		"net1":     json.RawMessage(`"virtio=CC:DD,bridge=vmbr1"`),
		"scsi0":    json.RawMessage(`"local-lvm:vm-100-disk-0,size=32G"`),
		"unused3":  json.RawMessage(`"local-lvm:vm-100-disk-3"`),
		"hostpci0": json.RawMessage(`"0000:01:00.0"`),
	}

	cfg, err := decodeConfig(raw)
	if err != nil {
		t.Fatalf("decodeConfig: unexpected error: %v", err)
	}

	if cfg.Name != "test-vm" {
		t.Errorf("Name = %q, want %q", cfg.Name, "test-vm")
	}
	if cfg.Cores != 4 {
		t.Errorf("Cores = %d, want 4", cfg.Cores)
	}
	if !cfg.NUMAEnabled {
		t.Errorf("NUMAEnabled = false, want true")
	}
	if cfg.Digest != "abc123" {
		t.Errorf("Digest = %q, want %q", cfg.Digest, "abc123")
	}
	if cfg.SMBios1 != "uuid=deadbeef" {
		t.Errorf("SMBios1 = %q, want %q", cfg.SMBios1, "uuid=deadbeef")
	}

	if got := cfg.Net[0]; got != "virtio=AA:BB,bridge=vmbr0" {
		t.Errorf("Net[0] = %q, want %q", got, "virtio=AA:BB,bridge=vmbr0")
	}
	if got := cfg.Net[1]; got != "virtio=CC:DD,bridge=vmbr1" {
		t.Errorf("Net[1] = %q, want %q", got, "virtio=CC:DD,bridge=vmbr1")
	}
	if len(cfg.Net) != 2 {
		t.Errorf("len(Net) = %d, want 2", len(cfg.Net))
	}
	if got := cfg.SCSI[0]; got != "local-lvm:vm-100-disk-0,size=32G" {
		t.Errorf("SCSI[0] = %q, want %q", got, "local-lvm:vm-100-disk-0,size=32G")
	}
	if got := cfg.Unused[3]; got != "local-lvm:vm-100-disk-3" {
		t.Errorf("Unused[3] = %q, want %q", got, "local-lvm:vm-100-disk-3")
	}
	if got := cfg.HostPCI[0]; got != "0000:01:00.0" {
		t.Errorf("HostPCI[0] = %q, want %q", got, "0000:01:00.0")
	}
}

func TestEncodeConfig(t *testing.T) {
	cfg := &Config{
		Name:   "test-vm",
		Cores:  4,
		Net:    map[int]string{0: "virtio=AA:BB,bridge=vmbr0"},
		SCSI:   map[int]string{0: "local-lvm:32"},
		Delete: []string{"net1", "description"},
	}

	p, err := encodeConfig(cfg)
	if err != nil {
		t.Fatalf("encodeConfig: unexpected error: %v", err)
	}

	want := map[string]string{
		"name":   "test-vm",
		"cores":  "4",
		"net0":   "virtio=AA:BB,bridge=vmbr0",
		"scsi0":  "local-lvm:32",
		"delete": "net1,description",
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

// TestEncodeConfigReadonlyFieldsExcluded verifies that Meta and the
// snapshot-only fields (Parent, SnapTime, VMState, RunningMachine,
// RunningCPU, RunningNetsHostMTU) are never sent, even when populated —
// the common case being a caller round-tripping a fetched Config back
// through UpdateConfig.
func TestEncodeConfigReadonlyFieldsExcluded(t *testing.T) {
	cfg := &Config{
		Meta:               "creation-qemu=9.0.2,ctime=1700000000",
		Parent:             "pre-upgrade",
		SnapTime:           1700000000,
		VMState:            "local-lvm:vm-100-state-pre-upgrade",
		RunningMachine:     "pc-i440fx-9.0+pve0",
		RunningCPU:         "kvm64",
		RunningNetsHostMTU: "net0=1500",
	}

	p, err := encodeConfig(cfg)
	if err != nil {
		t.Fatalf("encodeConfig: unexpected error: %v", err)
	}
	if len(p) != 0 {
		t.Errorf("encodeConfig sent readonly fields: %+v", p)
	}
}
