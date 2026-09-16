package capabilities

import "github.com/sergelogvinov/go-proxmox-rest/types"

// Arch, Accel and CPUFlag are identical between this package and
// cluster/qemu: both back onto the same QEMU/KVM CPU-flag enumeration
// logic. They live in the shared types package (see its doc comment)
// and are re-exported here as aliases so call sites keep reading as
// capabilities.Arch, capabilities.CPUFlag, ... . CPUModel below is a
// different shape despite the shared name with cluster/qemu.CPUModel —
// see the types package doc comment.
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

// CPUModel describes a single CPU model available for QEMU guests on this
// node (built-in or custom), as returned by
// GET /nodes/{node}/capabilities/qemu/cpu.
type CPUModel struct {
	// Name identifies the model for subsequent API calls (e.g. as a
	// guest's cpu type); custom models are prefixed "custom-".
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Custom is true if this is a custom CPU model.
	Custom bool `json:"custom,omitempty" url:"custom,omitempty"`
	// Abstract is true for PVE-internal abstract profiles (e.g.
	// "x86-64-v2"), which have no corresponding QEMU CPU type and
	// cannot be used as a custom model's reported-model.
	Abstract bool `json:"abstract,omitempty" url:"abstract,omitempty"`
	// Vendor is the CPU vendor visible to the guest when this model is
	// selected (the reported model's vendor, for custom models).
	Vendor string `json:"vendor,omitempty" url:"vendor,omitempty"`
}

// MachineKind is a QEMU machine type's chipset family.
type MachineKind string

const (
	MachineKindQ35    MachineKind = "q35"
	MachineKindI440FX MachineKind = "i440fx"
)

// MachineType describes a single supported QEMU/KVM machine type/version,
// as returned by GET /nodes/{node}/capabilities/qemu/machines.
type MachineType struct {
	// ID is the full machine type and version, e.g. "pc-q35-9.0+pve1".
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Type is the machine's chipset family.
	Type MachineKind `json:"type,omitempty" url:"type,omitempty"`
	// Version is the machine version, e.g. "9.0+pve1".
	Version string `json:"version,omitempty" url:"version,omitempty"`
	// Changes describes notable changes in this version; only set for
	// "+pveX" Proxmox-specific revisions.
	Changes string `json:"changes,omitempty" url:"changes,omitempty"`
}

// MigrationCapabilities describes a node's QEMU live-migration
// capabilities, as returned by
// GET /nodes/{node}/capabilities/qemu/migration.
type MigrationCapabilities struct {
	// HasDBusVMState is true if the node supports live-migrating
	// additional VM state via the dbus-vmstate helper.
	HasDBusVMState bool `json:"has-dbus-vmstate,omitempty" url:"has-dbus-vmstate,omitempty"`
}
