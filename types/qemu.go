package types

// qemu.go: the QEMU CPU-flag shapes Proxmox reuses verbatim between the
// cluster-wide capability query (cluster/qemu, GET /cluster/qemu/cpu-flags)
// and the per-node one (nodes/capabilities, GET
// /nodes/{node}/capabilities/qemu/cpu-flags) — both back onto the same
// QEMU/KVM flag enumeration logic. CPUModel is deliberately not shared
// here even though both packages define a type with that name: cluster/
// qemu's CPUModel is a custom CPU model's full read/write definition
// (CPUType, ReportedModel, Hidden, ...), while nodes/capabilities'
// CPUModel is a lightweight summary of any available model, built-in or
// custom (Name, Custom, Abstract, Vendor) — same name, unrelated shapes.

// Arch filters a node/cluster's available CPU flags (and, for
// nodes/capabilities specifically, CPU models and machine types too) to
// a single virtual processor architecture. An empty value defaults to
// the host's own architecture.
type Arch string

const (
	// ArchX8664 matches x86_64 capabilities.
	ArchX8664 Arch = "x86_64"
	// ArchAarch64 matches aarch64 capabilities. Proxmox reports none
	// today for CPU flags specifically.
	ArchAarch64 Arch = "aarch64"
)

// Accel is the acceleration type CPU flag queries check node
// compatibility for.
type Accel string

const (
	// AccelKVM checks flags supported under hardware-accelerated KVM.
	AccelKVM Accel = "kvm"
	// AccelTCG checks flags supported under software-emulated TCG.
	AccelTCG Accel = "tcg"
)

// CPUFlag describes a single available CPU flag, as returned by both
// GET /cluster/qemu/cpu-flags and
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
