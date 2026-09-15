package qemu

import (
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Arch filters CPUFlags to a single virtual processor architecture. An
// empty value defaults to the host's own architecture.
type Arch string

const (
	// ArchX8664 matches x86_64 CPU flags.
	ArchX8664 Arch = "x86_64"
	// ArchAarch64 matches aarch64 CPU flags (Proxmox reports none today).
	ArchAarch64 Arch = "aarch64"
)

// Accel is the acceleration type CPUFlags checks node compatibility for.
type Accel string

const (
	// AccelKVM checks flags supported under hardware-accelerated KVM.
	AccelKVM Accel = "kvm"
	// AccelTCG checks flags supported under software-emulated TCG.
	AccelTCG Accel = "tcg"
)

// CPUFlag describes a single available CPU flag, as returned by
// GET /cluster/qemu/cpu-flags.
type CPUFlag struct {
	// Name is the CPU flag name.
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Description describes the flag.
	Description string `json:"description,omitempty" url:"description,omitempty"`
	// SupportedOn lists the nodes that support this flag with the
	// requested acceleration type.
	SupportedOn []string `json:"supported-on,omitempty" url:"supported-on,omitempty"`
}

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
