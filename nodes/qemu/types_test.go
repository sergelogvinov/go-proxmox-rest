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
	"testing"
)

func TestSMBios1UnmarshalJSON(t *testing.T) {
	var value SMBios1
	if err := json.Unmarshal([]byte(`"base64=1,manufacturer=proxmox,serial=abc,uuid=1234"`), &value); err != nil {
		t.Fatalf("unmarshal SMBIOS1: %v", err)
	}

	if value.Base64 == nil || !*value.Base64 {
		t.Fatal("expected base64 to be enabled")
	}
	if value.Manufacturer != "proxmox" || value.Serial != "abc" || value.UUID != "1234" {
		t.Fatalf("unexpected SMBIOS1 value: %+v", value)
	}
}

func TestSMBios1String(t *testing.T) {
	base64 := true
	value := SMBios1{Base64: &base64, Manufacturer: "proxmox", Serial: "abc", UUID: "1234"}

	const expected = "base64=1,manufacturer=proxmox,serial=abc,uuid=1234"
	if actual := value.String(); actual != expected {
		t.Fatalf("String() = %q, want %q", actual, expected)
	}
}

func TestTagsUnmarshalJSON(t *testing.T) {
	var value Tags
	if err := json.Unmarshal([]byte(`"prod;web;team-a"`), &value); err != nil {
		t.Fatalf("unmarshal Tags: %v", err)
	}

	want := []string{"prod", "web", "team-a"}
	if len(value.Tags) != len(want) {
		t.Fatalf("unexpected Tags value: %+v", value)
	}
	for i := range want {
		if value.Tags[i] != want[i] {
			t.Fatalf("unexpected Tags value: %+v", value)
		}
	}
}

func TestTagsString(t *testing.T) {
	value := Tags{Tags: []string{"prod", "web", "team-a"}}

	const expected = "prod;web;team-a"
	if actual := value.String(); actual != expected {
		t.Fatalf("String() = %q, want %q", actual, expected)
	}
}

func TestStartupUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Startup
	}{
		{
			name:  "bare order",
			input: `"2,up=30,down=60"`,
			want:  Startup{Order: new(2), Up: new(30), Down: new(60)},
		},
		{
			name:  "named order",
			input: `"order=3"`,
			want:  Startup{Order: new(3)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Startup
			if err := got.UnmarshalJSON([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if got.Order == nil || *got.Order != *tt.want.Order ||
				!sameIntPtr(got.Up, tt.want.Up) || !sameIntPtr(got.Down, tt.want.Down) {
				t.Fatalf("UnmarshalJSON() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestStartupString(t *testing.T) {
	value := Startup{Order: new(2), Up: new(30), Down: new(60)}
	if got, want := value.String(), "2,up=30,down=60"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestAMDSev(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  AMDSev
	}{
		{
			name:  "default type and options",
			input: `"snp,allow-smt=1,kernel-hashes=1,no-debug=0,no-key-sharing=1"`,
			want: AMDSev{
				Type:         "snp",
				AllowSMT:     new(true),
				KernelHashes: new(true),
				NoDebug:      new(false),
				NoKeySharing: new(true),
			},
		},
		{
			name:  "named type",
			input: `"type=std"`,
			want:  AMDSev{Type: "std"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got AMDSev
			if err := got.UnmarshalJSON([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if got.String() != tt.want.String() {
				t.Fatalf("UnmarshalJSON() = %q, want %q", got.String(), tt.want.String())
			}
		})
	}
}

func TestIntelTDX(t *testing.T) {
	value := IntelTDX{
		Type:        "tdx",
		Attestation: new(true),
		VsockCID:    new(12),
		VsockPort:   new(4050),
	}

	const expected = "tdx,attestation=1,vsock-cid=12,vsock-port=4050"
	if got := value.String(); got != expected {
		t.Fatalf("String() = %q, want %q", got, expected)
	}

	var decoded IntelTDX
	if err := decoded.UnmarshalJSON([]byte(`"tdx,attestation=1,vsock-cid=12,vsock-port=4050"`)); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if got := decoded.String(); got != expected {
		t.Fatalf("UnmarshalJSON() result = %q, want %q", got, expected)
	}
}

func TestCPU(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bare cputype",
			input: `"host"`,
			want:  "host",
		},
		{
			name:  "named cputype with flags",
			input: `"cputype=host,flags=+aes;-aes;pcid,hidden=1"`,
			want:  "host,flags=+aes;-aes;pcid,hidden=1",
		},
		{
			name:  "full options",
			input: `"cputype=kvm64,level=30,phys-bits=host,guest-phys-bits=40,hv-vendor-id=tux"`,
			want:  "kvm64,guest-phys-bits=40,hv-vendor-id=tux,level=30,phys-bits=host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got CPU
			if err := got.UnmarshalJSON([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("UnmarshalJSON() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestMemory(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bare current",
			input: `"2048"`,
			want:  "2048",
		},
		{
			name:  "named current with max",
			input: `"current=2048,max=8192"`,
			want:  "2048,max=8192",
		},
		{
			name:  "full options",
			input: `"current=2048,max=8192,min=512"`,
			want:  "2048,max=8192,min=512",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Memory
			if err := got.UnmarshalJSON([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("UnmarshalJSON() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestAgent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bare enabled",
			input: `"1"`,
			want:  "1",
		},
		{
			name:  "enabled with options",
			input: `"1,fstrim_cloned_disks=1,type=virtio"`,
			want:  "1,fstrim_cloned_disks=1,type=virtio",
		},
		{
			name:  "named enabled disabled",
			input: `"enabled=0,freeze-fs=1"`,
			want:  "0,freeze-fs=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Agent
			if err := got.UnmarshalJSON([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("UnmarshalJSON() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestVGA(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bare type",
			input: `"std"`,
			want:  "std",
		},
		{
			name:  "type with memory",
			input: `"qxl,memory=32"`,
			want:  "qxl,memory=32",
		},
		{
			name:  "named type with clipboard",
			input: `"type=virtio-gl,clipboard=vnc"`,
			want:  "virtio-gl,clipboard=vnc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got VGA
			if err := got.UnmarshalJSON([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("UnmarshalJSON() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestSpiceEnhancements(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "folder sharing",
			input: `"foldersharing=1"`,
			want:  "foldersharing=1",
		},
		{
			name:  "all options",
			input: `"foldersharing=1,videostreaming=all"`,
			want:  "foldersharing=1,videostreaming=all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got SpiceEnhancements
			if err := got.UnmarshalJSON([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("UnmarshalJSON() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestRNG0(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bare source",
			input: `"/dev/urandom"`,
			want:  "/dev/urandom",
		},
		{
			name:  "named source with limits",
			input: `"source=/dev/random,max_bytes=512,period=2000"`,
			want:  "/dev/random,max_bytes=512,period=2000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got RNG0
			if err := got.UnmarshalJSON([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("UnmarshalJSON() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestTPMState(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bare volume",
			input: `"local-lvm:vm-100-disk-1"`,
			want:  "local-lvm:vm-100-disk-1",
		},
		{
			name:  "named file with version",
			input: `"file=local-lvm:vm-100-disk-1,version=v2.0"`,
			want:  "local-lvm:vm-100-disk-1,version=v2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got TPMState
			if err := got.UnmarshalJSON([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("UnmarshalJSON() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestMachine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bare type",
			input: `"q35"`,
			want:  "q35",
		},
		{
			name:  "type with viommu",
			input: `"q35,viommu=virtio"`,
			want:  "q35,viommu=virtio",
		},
		{
			name:  "full options",
			input: `"type=pc,aw-bits=39,enable-s3=1,enable-s4=0"`,
			want:  "pc,aw-bits=39,enable-s3=1,enable-s4=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Machine
			if err := got.UnmarshalJSON([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("UnmarshalJSON() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func sameIntPtr(left, right *int) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}
