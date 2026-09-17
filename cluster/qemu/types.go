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
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
	"github.com/sergelogvinov/go-proxmox-rest/types"
)

// Arch, Accel and CPUFlag are identical between this package and
// nodes/capabilities: both back onto the same QEMU/KVM CPU-flag
// enumeration logic. They live in the shared types package (see its doc
// comment) and are re-exported here as aliases so call sites keep
// reading as qemu.Arch, qemu.CPUFlag, ... . CPUModel below is a
// different shape despite the shared name with
// nodes/capabilities.CPUModel — see the types package doc comment.
type (
	Arch    = types.Arch
	Accel   = types.Accel
	CPUFlag = types.CPUFlag
)

const (
	ArchX8664   = types.ArchX8664
	ArchAarch64 = types.ArchAarch64

	AccelKVM = types.AccelKVM
	AccelTCG = types.AccelTCG
)

// CPUModel describes a custom CPU model definition as returned by
// GET /cluster/qemu/custom-cpu-models and
// GET /cluster/qemu/custom-cpu-models/{cputype}.
type CPUModel struct {
	// CPUType is the model identifier, always prefixed with "custom-",
	// e.g. "custom-my-model".
	CPUType string `json:"cputype,omitempty" url:"cputype,omitempty"`
	// ReportedModel is the CPU model/vendor reported to the guest, e.g.
	// "kvm64". Must be one of the built-in QEMU/KVM CPU models.
	ReportedModel string `json:"reported-model,omitempty" url:"reported-model,omitempty"`
	// Hidden hides the hypervisor CPUID leaf so the guest does not
	// identify as running under KVM (x86_64 vCPUs only).
	Hidden bool `json:"hidden,omitempty" url:"hidden,omitempty"`
	// HVVendorID is the Hyper-V vendor ID reported to Windows guests.
	HVVendorID string `json:"hv-vendor-id,omitempty" url:"hv-vendor-id,omitempty"`
	// Flags is a ';'-separated list of additional CPU flags, e.g.
	// "+aes;-hypervisor".
	Flags string `json:"flags,omitempty" url:"flags,omitempty"`
	// GuestPhysBits is the number of physical address bits available to
	// the guest.
	GuestPhysBits int `json:"guest-phys-bits,omitempty" url:"guest-phys-bits,omitempty"`
	// PhysBits is the physical memory address bits reported to the
	// guest OS: a number (8-64) or "host".
	PhysBits string `json:"phys-bits,omitempty" url:"phys-bits,omitempty"`
	// Level is the maximum CPUID leaf the guest can query (x86_64 only).
	Level int64 `json:"level,omitempty" url:"level,omitempty"`
	// Digest is the configuration digest, usable with
	// CPUModelOptions.Digest to guard concurrent updates.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// CPUModelOptions holds the write parameters shared by
// POST /cluster/qemu/custom-cpu-models (Create) and
// PUT /cluster/qemu/custom-cpu-models/{cputype} (Update).
//
// CPUType and ReportedModel are both required by Create. Update ignores
// CPUType (customCPUModelsResource.Update fills it in from the requested
// cputype, since Proxmox's update schema requires it in the body too, even
// though the same value is already part of the URL) and treats
// ReportedModel as optional, but Proxmox refuses to remove it via Delete.
// Pointer fields are always sent when non-nil (even when zero/empty),
// which is how Update expresses "clear this field" — Create simply sends
// whatever is set. Delete and Digest are meaningful to Update only.
type CPUModelOptions struct {
	// CPUType is the model identifier; the "custom-" prefix is optional
	// and added automatically if missing. Required by Create; not
	// settable on Update.
	CPUType string `url:"cputype"`
	// ReportedModel is the CPU model/vendor reported to the guest, e.g.
	// "kvm64". Required by Create.
	ReportedModel *string `url:"reported-model"`
	// Hidden hides the hypervisor CPUID leaf so the guest does not
	// identify as running under KVM (x86_64 vCPUs only).
	Hidden *bool `url:"hidden"`
	// HVVendorID is the Hyper-V vendor ID reported to Windows guests.
	HVVendorID *string `url:"hv-vendor-id"`
	// Flags is a ';'-separated list of additional CPU flags, e.g.
	// "+aes;-hypervisor".
	Flags *string `url:"flags"`
	// GuestPhysBits is the number of physical address bits available to
	// the guest.
	GuestPhysBits *int `url:"guest-phys-bits"`
	// PhysBits is the physical memory address bits reported to the
	// guest OS: a number (8-64) or "host".
	PhysBits *string `url:"phys-bits"`
	// Level is the maximum CPUID leaf the guest can query (x86_64 only).
	Level *int64 `url:"level"`
	// Delete lists properties to reset to their default value (Update
	// only). Proxmox refuses "cputype" and "reported-model" here.
	Delete []string `url:"delete"`
	// Digest prevents changes if the current configuration has changed
	// in between (value from the corresponding Get). Update only.
	Digest string `url:"digest"`
}

// encode converts the options to form parameters.
func (o *CPUModelOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("qemu: custom cpu model options are required")
	}

	return params.Encode(o)
}
