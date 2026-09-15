package qemu

// VMStatus is a QEMU process's running state, as reported by
// Status.Status.
type VMStatus string

const (
	VMStatusStopped VMStatus = "stopped"
	VMStatusRunning VMStatus = "running"
)

// Status describes a QEMU guest's current status, as returned by
// GET /nodes/{node}/qemu/{vmid}/status/current.
type Status struct {
	// VMID is the guest's id.
	VMID int `json:"vmid,omitempty" url:"vmid,omitempty"`
	// Status is the QEMU process's running state.
	Status VMStatus `json:"status,omitempty" url:"status,omitempty"`
	// Mem is the currently used memory in bytes.
	Mem int64 `json:"mem,omitempty" url:"mem,omitempty"`
	// MaxMem is the maximum memory in bytes.
	MaxMem int64 `json:"maxmem,omitempty" url:"maxmem,omitempty"`
	// MemHost is the current host-side memory usage in bytes.
	MemHost int64 `json:"memhost,omitempty" url:"memhost,omitempty"`
	// MaxDisk is the root disk size in bytes.
	MaxDisk int64 `json:"maxdisk,omitempty" url:"maxdisk,omitempty"`
	// DiskRead is the total bytes read from block devices since start.
	DiskRead int64 `json:"diskread,omitempty" url:"diskread,omitempty"`
	// DiskWrite is the total bytes written to block devices since
	// start.
	DiskWrite int64 `json:"diskwrite,omitempty" url:"diskwrite,omitempty"`
	// Name is the VM's (host)name.
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// NetIn is the total network bytes received since start.
	NetIn int64 `json:"netin,omitempty" url:"netin,omitempty"`
	// NetOut is the total network bytes sent since start.
	NetOut int64 `json:"netout,omitempty" url:"netout,omitempty"`
	// QMPStatus is the VM run state from QEMU's own "query-status" QMP
	// command.
	QMPStatus string `json:"qmpstatus,omitempty" url:"qmpstatus,omitempty"`
	// PID is the QEMU process's PID, if running.
	PID int `json:"pid,omitempty" url:"pid,omitempty"`
	// Uptime is the guest's uptime in seconds.
	Uptime int64 `json:"uptime,omitempty" url:"uptime,omitempty"`
	// CPU is the current CPU usage (fraction of a single CPU, 0-1 per
	// core).
	CPU float64 `json:"cpu,omitempty" url:"cpu,omitempty"`
	// CPUs is the maximum usable CPU count.
	CPUs float64 `json:"cpus,omitempty" url:"cpus,omitempty"`
	// Lock is the current config lock holder, if any (e.g. "backup",
	// "migrate").
	Lock string `json:"lock,omitempty" url:"lock,omitempty"`
	// Tags is the guest's configured tags.
	Tags string `json:"tags,omitempty" url:"tags,omitempty"`
	// RunningMachine is the currently running QEMU machine type, if
	// running.
	RunningMachine string `json:"running-machine,omitempty" url:"running-machine,omitempty"`
	// RunningQemu is the QEMU version currently in use, if running.
	RunningQemu string `json:"running-qemu,omitempty" url:"running-qemu,omitempty"`
	// Template marks the guest as a template.
	Template bool `json:"template,omitempty" url:"template,omitempty"`
	// Serial is true if the guest has a serial device configured.
	Serial bool `json:"serial,omitempty" url:"serial,omitempty"`
	// PressureCPUSome is the CPU "some" pressure stall average over the
	// last 10 seconds.
	PressureCPUSome float64 `json:"pressurecpusome,omitempty" url:"pressurecpusome,omitempty"`
	// PressureCPUFull is the CPU "full" pressure stall average over the
	// last 10 seconds.
	PressureCPUFull float64 `json:"pressurecpufull,omitempty" url:"pressurecpufull,omitempty"`
	// PressureIOSome is the IO "some" pressure stall average over the
	// last 10 seconds.
	PressureIOSome float64 `json:"pressureiosome,omitempty" url:"pressureiosome,omitempty"`
	// PressureIOFull is the IO "full" pressure stall average over the
	// last 10 seconds.
	PressureIOFull float64 `json:"pressureiofull,omitempty" url:"pressureiofull,omitempty"`
	// PressureMemorySome is the memory "some" pressure stall average
	// over the last 10 seconds.
	PressureMemorySome float64 `json:"pressurememorysome,omitempty" url:"pressurememorysome,omitempty"`
	// PressureMemoryFull is the memory "full" pressure stall average
	// over the last 10 seconds.
	PressureMemoryFull float64 `json:"pressurememoryfull,omitempty" url:"pressurememoryfull,omitempty"`
	// HA is the guest's HA manager service status, an untyped object
	// (Proxmox's own schema declares it as a bare "object" with no
	// fixed properties).
	HA map[string]any `json:"ha,omitempty" url:"ha,omitempty"`
	// Spice is true if the guest's VGA configuration supports SPICE.
	Spice bool `json:"spice,omitempty" url:"spice,omitempty"`
	// Agent is true if the QEMU Guest Agent is enabled in the guest's
	// configuration.
	Agent bool `json:"agent,omitempty" url:"agent,omitempty"`
	// Clipboard is the configured clipboard mode, currently only "vnc"
	// when set.
	Clipboard string `json:"clipboard,omitempty" url:"clipboard,omitempty"`
}

// StartOptions holds the parameters for statusResource.Start
// (POST .../status/start). This intentionally omits Proxmox's
// migration-internal parameters (stateuri, migratedfrom, migration_type,
// migration_network, targetstorage, force-cpu, with-conntrack-state,
// nets-host-mtu) — those exist for node-to-node live migration, not for
// a general client starting a guest.
type StartOptions struct {
	// Skiplock bypasses the guest's configuration lock. Root only.
	Skiplock bool `url:"skiplock,omitempty"`
	// Machine forces a specific QEMU machine type for this start,
	// overriding the guest's configured one.
	Machine string `url:"machine,omitempty"`
	// Timeout is the maximum number of seconds to wait for the guest to
	// come up. Proxmox defaults to max(30, memory in GiB).
	Timeout int `url:"timeout,omitempty"`
}

// StopOptions holds the parameters for statusResource.Stop
// (POST .../status/stop). Stop is immediate (like pulling the power
// plug); see ShutdownOptions for a graceful ACPI shutdown.
type StopOptions struct {
	// Skiplock bypasses the guest's configuration lock. Root only.
	Skiplock bool `url:"skiplock,omitempty"`
	// Timeout is the maximum number of seconds to wait.
	Timeout int `url:"timeout,omitempty"`
	// KeepActive leaves the guest's storage volumes active instead of
	// deactivating them. Root only.
	KeepActive bool `url:"keepActive,omitempty"`
	// OverruleShutdown aborts any active graceful-shutdown task for
	// this guest before stopping it.
	OverruleShutdown bool `url:"overrule-shutdown,omitempty"`
}

// ResetOptions holds the parameters for statusResource.Reset
// (POST .../status/reset).
type ResetOptions struct {
	// Skiplock bypasses the guest's configuration lock. Root only.
	Skiplock bool `url:"skiplock,omitempty"`
}

// ShutdownOptions holds the parameters for statusResource.Shutdown
// (POST .../status/shutdown), a graceful ACPI power-off.
type ShutdownOptions struct {
	// Skiplock bypasses the guest's configuration lock. Root only.
	Skiplock bool `url:"skiplock,omitempty"`
	// Timeout is the maximum number of seconds to wait.
	Timeout int `url:"timeout,omitempty"`
	// ForceStop falls back to an immediate stop if the graceful
	// shutdown doesn't complete.
	ForceStop bool `url:"forceStop,omitempty"`
	// KeepActive leaves the guest's storage volumes active instead of
	// deactivating them. Root only.
	KeepActive bool `url:"keepActive,omitempty"`
}

// RebootOptions holds the parameters for statusResource.Reboot
// (POST .../status/reboot).
type RebootOptions struct {
	// Timeout is the maximum number of seconds to wait for the
	// shutdown half of the reboot.
	Timeout int `url:"timeout,omitempty"`
}

// SuspendOptions holds the parameters for statusResource.Suspend
// (POST .../status/suspend).
type SuspendOptions struct {
	// Skiplock bypasses the guest's configuration lock. Root only.
	Skiplock bool `url:"skiplock,omitempty"`
	// ToDisk suspends the guest to disk (hibernate) instead of to RAM;
	// it resumes automatically on the next start.
	ToDisk bool `url:"todisk,omitempty"`
	// StateStorage is the storage to hold the suspend-to-disk state.
	// Only meaningful, and only usable, with ToDisk.
	StateStorage string `url:"statestorage,omitempty"`
}

// ResumeOptions holds the parameters for statusResource.Resume
// (POST .../status/resume).
type ResumeOptions struct {
	// Skiplock bypasses the guest's configuration lock. Root only.
	Skiplock bool `url:"skiplock,omitempty"`
}
