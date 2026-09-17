package nodes

import (
	"context"
)

// Status describes a node's current runtime status as returned by
// GET /nodes/{node}/status.
//
// Proxmox declares this endpoint's return schema as "additionalProperties
// => 1" with most fields left as TODOs, so the fields below were verified
// directly against PVE::API2::Nodes's status handler (pve-manager's
// PVE/API2/Nodes.pm) rather than the (incomplete) documented schema.
type Status struct {
	// BootInfo describes the firmware the node booted through.
	BootInfo *BootInfo `json:"boot-info,omitempty" url:"boot-info,omitempty"`
	// CPU is the current CPU usage, a fraction between 0 and 1.
	CPU float64 `json:"cpu,omitempty" url:"cpu,omitempty"`
	// CPUInfo describes the node's physical CPU.
	CPUInfo *CPUInfo `json:"cpuinfo,omitempty" url:"cpuinfo,omitempty"`
	// CurrentKernel describes the currently booted kernel.
	CurrentKernel *CurrentKernel `json:"current-kernel,omitempty" url:"current-kernel,omitempty"`
	// Idle is the idle time counter from /proc/uptime, in seconds.
	Idle int64 `json:"idle,omitempty" url:"idle,omitempty"`
	// KSM reports kernel samepage-merging memory savings.
	KSM *KSM `json:"ksm,omitempty" url:"ksm,omitempty"`
	// Kversion is the legacy combined "sysname release version" string;
	// prefer CurrentKernel's split fields for new code.
	Kversion string `json:"kversion,omitempty" url:"kversion,omitempty"`
	// LoadAvg is the 1, 5 and 15 minute load averages, in that order.
	LoadAvg []string `json:"loadavg,omitempty" url:"loadavg,omitempty"`
	// Memory reports the node's RAM usage.
	Memory *Memory `json:"memory,omitempty" url:"memory,omitempty"`
	// PVEVersion is the "pve-manager/<version>" package/version string.
	PVEVersion string `json:"pveversion,omitempty" url:"pveversion,omitempty"`
	// RootFS reports disk usage of the root filesystem.
	RootFS *RootFS `json:"rootfs,omitempty" url:"rootfs,omitempty"`
	// Swap reports the node's swap usage.
	Swap *Swap `json:"swap,omitempty" url:"swap,omitempty"`
	// Uptime is the node's uptime in seconds.
	Uptime int64 `json:"uptime,omitempty" url:"uptime,omitempty"`
	// Wait is the CPU IO-wait time.
	Wait float64 `json:"wait,omitempty" url:"wait,omitempty"`
}

// BootInfo describes the firmware mode a node booted through.
type BootInfo struct {
	// Mode is "efi" or "legacy-bios".
	Mode string `json:"mode,omitempty" url:"mode,omitempty"`
	// SecureBoot is true if the node booted with EFI secure boot
	// enabled; only meaningful when Mode is "efi".
	SecureBoot bool `json:"secureboot,omitempty" url:"secureboot,omitempty"`
}

// CurrentKernel describes the currently running kernel, from uname(2).
type CurrentKernel struct {
	Sysname string `json:"sysname,omitempty" url:"sysname,omitempty"`
	Release string `json:"release,omitempty" url:"release,omitempty"`
	Version string `json:"version,omitempty" url:"version,omitempty"`
	Machine string `json:"machine,omitempty" url:"machine,omitempty"`
}

// CPUInfo describes the node's physical CPU.
type CPUInfo struct {
	// Cores is the number of physical cores.
	Cores int `json:"cores,omitempty" url:"cores,omitempty"`
	// CPUs is the number of logical threads.
	CPUs int `json:"cpus,omitempty" url:"cpus,omitempty"`
	// Model is the CPU model string.
	Model string `json:"model,omitempty" url:"model,omitempty"`
	// Sockets is the number of CPU sockets.
	Sockets int `json:"sockets,omitempty" url:"sockets,omitempty"`
}

// Memory reports a node's RAM usage, in bytes.
type Memory struct {
	Free      int64 `json:"free,omitempty" url:"free,omitempty"`
	Available int64 `json:"available,omitempty" url:"available,omitempty"`
	Total     int64 `json:"total,omitempty" url:"total,omitempty"`
	Used      int64 `json:"used,omitempty" url:"used,omitempty"`
}

// KSM reports kernel samepage-merging memory savings, in bytes.
type KSM struct {
	Shared int64 `json:"shared,omitempty" url:"shared,omitempty"`
}

// Swap reports a node's swap usage, in bytes.
type Swap struct {
	Free  int64 `json:"free,omitempty" url:"free,omitempty"`
	Total int64 `json:"total,omitempty" url:"total,omitempty"`
	Used  int64 `json:"used,omitempty" url:"used,omitempty"`
}

// RootFS reports disk usage of the root filesystem, in bytes.
type RootFS struct {
	Free  int64 `json:"free,omitempty" url:"free,omitempty"`
	Total int64 `json:"total,omitempty" url:"total,omitempty"`
	Used  int64 `json:"used,omitempty" url:"used,omitempty"`
	Avail int64 `json:"avail,omitempty" url:"avail,omitempty"`
}

// Status retrieves the node's runtime status via GET /nodes/{node}/status.
func (c *Client) Status(ctx context.Context) (*Status, error) {
	s := &Status{}
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/status", s, nil); err != nil {
		return nil, err
	}

	return s, nil
}
