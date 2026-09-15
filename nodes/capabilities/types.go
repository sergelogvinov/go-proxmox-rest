package capabilities

// Arch filters CPUModels/CPUFlags/Machines to a single virtual processor
// architecture. An empty value defaults to the host's own architecture.
type Arch string

const (
	// ArchX8664 matches x86_64 capabilities.
	ArchX8664 Arch = "x86_64"
	// ArchAarch64 matches aarch64 capabilities.
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

// CPUFlag describes a single available CPU flag, as returned by
// GET /nodes/{node}/capabilities/qemu/cpu-flags.
type CPUFlag struct {
	// Name is the CPU flag name.
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Description describes the flag.
	Description string `json:"description,omitempty" url:"description,omitempty"`
	// SupportedOn lists the nodes that support this flag with the
	// requested acceleration type.
	SupportedOn []string `json:"supported-on,omitempty" url:"supported-on,omitempty"`
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
