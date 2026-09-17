package lxc

import "github.com/sergelogvinov/go-proxmox-rest/types"

// Config describes an LXC container's configuration, as returned by
// GET /nodes/{node}/lxc/{vmid}/config and accepted by
// Client.UpdateConfig (PUT to the same path).
//
// Three numerically-suffixed families hold the container's mount points
// and network interfaces (net0-31, mp0-255, unused0-255): each is a
// map[int]string keyed by index, holding Proxmox's own property-string
// value verbatim (e.g. Net[0] == "name=eth0,bridge=vmbr0,ip=dhcp").
// RootFS (the container's root mount point) is a singular field, not
// part of this indexing — it's always exactly one entry. Parsing these
// property strings further is out of scope here — see
// PVE::LXC::Config's mount point / network format definitions for the
// authoritative grammar.
//
// Fields tagged "readonly" are populated from a GET response but never
// sent by UpdateConfig, even if a caller round-trips a fetched Config
// back through it — they are internal/snapshot-only state (parent
// snapshot, ...) or GET-only diagnostic info (LXC) that Proxmox computes
// itself and does not accept as write input.
type Config struct {
	// -- lifecycle / identity --

	// Hostname is the container's host name.
	Hostname string `json:"hostname,omitempty" url:"hostname,omitempty"`
	// Description is the container's description, shown in the web
	// UI's summary panel and saved as a comment inside the config file.
	Description string `json:"description,omitempty" url:"description,omitempty"`
	// Tags is the container's tag list (meta information only).
	Tags string `json:"tags,omitempty" url:"tags,omitempty"`
	// OSType is the guest OS type, used to select lxc setup scripts,
	// e.g. "debian", "alpine", "unmanaged".
	OSType string `json:"ostype,omitempty" url:"ostype,omitempty"`
	// Arch is the OS architecture, e.g. "amd64", "arm64".
	Arch string `json:"arch,omitempty" url:"arch,omitempty"`
	// Template marks the container as a template (see also
	// Client.Template, which performs the conversion).
	Template bool `json:"template,omitempty" url:"template,omitempty"`
	// Protection prevents CT/disk remove and update operations when
	// set.
	Protection bool `json:"protection,omitempty" url:"protection,omitempty"`
	// Lock is the current lock holder, if any (e.g. "backup",
	// "migrate", "mounted"). Include "lock" in Delete to force-unlock a
	// stuck container.
	Lock string `json:"lock,omitempty" url:"lock,omitempty"`
	// Digest is the configuration file's SHA1 digest. Set it on a write
	// to abort if the configuration changed concurrently since the
	// value was read.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
	// LXC holds the container's raw low-level lxc.conf entries as
	// [key, value] pairs. GET-only diagnostic information.
	LXC [][]string `json:"lxc,omitempty" url:"lxc,omitempty,readonly"`

	// -- boot / lifecycle behavior --

	// OnBoot starts the container automatically at host boot.
	OnBoot bool `json:"onboot,omitempty" url:"onboot,omitempty"`
	// Startup is the container's startup/shutdown ordering, e.g.
	// "order=2,up=30,down=60".
	Startup string `json:"startup,omitempty" url:"startup,omitempty"`
	// Console attaches a console device (/dev/console) to the
	// container.
	Console bool `json:"console,omitempty" url:"console,omitempty"`
	// TTY is the number of ttys available to the container.
	TTY int `json:"tty,omitempty" url:"tty,omitempty"`
	// CMode selects the console command's attach mode: "shell",
	// "console", or "tty".
	CMode string `json:"cmode,omitempty" url:"cmode,omitempty"`
	// Entrypoint is the command run as init, optionally with
	// arguments.
	Entrypoint string `json:"entrypoint,omitempty" url:"entrypoint,omitempty"`
	// Debug enables verbose debug log-level on start.
	Debug bool `json:"debug,omitempty" url:"debug,omitempty"`
	// HookScript is the volume id of a script run at various points in
	// the container's lifetime.
	HookScript string `json:"hookscript,omitempty" url:"hookscript,omitempty"`

	// -- CPU / memory --

	// Cores is the number of cores assigned to the container. Zero
	// (unset) allows using all available cores.
	Cores int `json:"cores,omitempty" url:"cores,omitempty"`
	// CPULimit caps CPU usage (in host CPUs); 0 means unlimited.
	CPULimit float64 `json:"cpulimit,omitempty" url:"cpulimit,omitempty"`
	// CPUUnits is the container's CPU scheduling weight, relative to
	// other running guests.
	CPUUnits int `json:"cpuunits,omitempty" url:"cpuunits,omitempty"`
	// Memory is the container's RAM in MB.
	Memory int `json:"memory,omitempty" url:"memory,omitempty"`
	// Swap is the container's swap space in MB.
	Swap int `json:"swap,omitempty" url:"swap,omitempty"`

	// -- networking --

	// SearchDomain sets the DNS search domain(s).
	SearchDomain string `json:"searchdomain,omitempty" url:"searchdomain,omitempty"`
	// Nameserver sets the DNS server IP address(es).
	Nameserver string `json:"nameserver,omitempty" url:"nameserver,omitempty"`

	// -- storage / platform --

	// RootFS is the container's root mount point as a property string,
	// e.g. "local-lvm:vm-100-disk-0,size=8G".
	RootFS string `json:"rootfs,omitempty" url:"rootfs,omitempty"`
	// TimeZone is the container's time zone, e.g. "host" or a zoneinfo
	// name.
	TimeZone string `json:"timezone,omitempty" url:"timezone,omitempty"`
	// Unprivileged runs the container as an unprivileged user.
	Unprivileged bool `json:"unprivileged,omitempty" url:"unprivileged,omitempty"`
	// Features configures advanced container features (nesting,
	// keyctl, mount types, ...) as a property string.
	Features string `json:"features,omitempty" url:"features,omitempty"`
	// Env is the container runtime environment as a NUL-separated
	// "KEY=value" list.
	Env string `json:"env,omitempty" url:"env,omitempty"`

	// -- internal / snapshot-only (read-only) --

	// Parent is the parent snapshot's name.
	Parent string `json:"parent,omitempty" url:"parent,omitempty,readonly"`
	// SnapTime is the snapshot's creation timestamp.
	SnapTime int64 `json:"snaptime,omitempty" url:"snaptime,omitempty,readonly"`

	// -- numerically-indexed hardware families, keyed by index --

	// Net holds netN entries (network interfaces), N in 0-31.
	Net map[int]string `json:"-" url:"-"`
	// MP holds mpN entries (additional mount points), N in 0-255.
	MP map[int]string `json:"-" url:"-"`
	// Unused holds unusedN entries (mount points detached from the
	// config but not deleted), N in 0-255.
	Unused map[int]string `json:"-" url:"-"`

	// -- write-only (never appear in a GET response) --

	// Delete lists config keys to reset to their default value, e.g.
	// []string{"description", "net0"}.
	Delete []string `json:"-" url:"delete,omitempty,writeonly"`
	// Revert lists pending changes to discard, reverting them to their
	// current value.
	Revert []string `json:"-" url:"revert,omitempty,writeonly"`
}

// ConfigOptions filters/selects the configuration view returned by
// Client.Config. A nil *ConfigOptions (or the zero value) requests the
// configuration with pending changes applied.
type ConfigOptions struct {
	// Current requests the current configuration instead of the
	// default (pending changes applied).
	Current bool `url:"current,omitempty"`
	// Snapshot fetches configuration values from the given snapshot
	// instead. Mutually exclusive with Current.
	Snapshot string `url:"snapshot,omitempty"`
}

// State is an LXC container's running state, as reported by
// Status.Status.
type State string

const (
	StateStopped State = "stopped"
	StateRunning State = "running"
)

// Status describes a container's current status, as returned by
// GET /nodes/{node}/lxc/{vmid}/status/current.
type Status struct {
	// VMID is the container's id.
	VMID int `json:"vmid,omitempty" url:"vmid,omitempty"`
	// Status is the container's running state.
	Status State `json:"status,omitempty" url:"status,omitempty"`
	// Mem is the currently used memory in bytes.
	Mem int64 `json:"mem,omitempty" url:"mem,omitempty"`
	// MaxMem is the maximum memory in bytes.
	MaxMem int64 `json:"maxmem,omitempty" url:"maxmem,omitempty"`
	// MaxSwap is the maximum swap in bytes.
	MaxSwap int64 `json:"maxswap,omitempty" url:"maxswap,omitempty"`
	// Disk is the root disk image's current space usage in bytes.
	Disk int64 `json:"disk,omitempty" url:"disk,omitempty"`
	// MaxDisk is the root disk image size in bytes.
	MaxDisk int64 `json:"maxdisk,omitempty" url:"maxdisk,omitempty"`
	// DiskRead is the total bytes read from block devices since start.
	DiskRead int64 `json:"diskread,omitempty" url:"diskread,omitempty"`
	// DiskWrite is the total bytes written to block devices since
	// start.
	DiskWrite int64 `json:"diskwrite,omitempty" url:"diskwrite,omitempty"`
	// Name is the container's name.
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// NetIn is the total network bytes received since start.
	NetIn int64 `json:"netin,omitempty" url:"netin,omitempty"`
	// NetOut is the total network bytes sent since start.
	NetOut int64 `json:"netout,omitempty" url:"netout,omitempty"`
	// Uptime is the container's uptime in seconds.
	Uptime int64 `json:"uptime,omitempty" url:"uptime,omitempty"`
	// CPU is the current CPU usage (fraction of a single CPU, 0-1 per
	// core).
	CPU float64 `json:"cpu,omitempty" url:"cpu,omitempty"`
	// CPUs is the maximum usable CPU count.
	CPUs float64 `json:"cpus,omitempty" url:"cpus,omitempty"`
	// Lock is the current config lock holder, if any.
	Lock string `json:"lock,omitempty" url:"lock,omitempty"`
	// Tags is the container's configured tags.
	Tags string `json:"tags,omitempty" url:"tags,omitempty"`
	// Template marks the container as a template.
	Template bool `json:"template,omitempty" url:"template,omitempty"`
	// PressureCPUSome is the CPU "some" pressure stall average over the
	// last 10 seconds.
	PressureCPUSome float64 `json:"pressurecpusome,omitempty" url:"pressurecpusome,omitempty"`
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
	// HA is the container's HA manager service status, an untyped
	// object (Proxmox's own schema declares it as a bare "object" with
	// no fixed properties).
	HA map[string]any `json:"ha,omitempty" url:"ha,omitempty"`
}

// Interface describes one of a container's network interfaces, as
// returned by Client.Interfaces. Proxmox reads these directly from the
// container's network namespace, so — unlike nodes/qemu/agent's
// NetworkInterface — no guest agent is required. Proxmox's own schema
// declares both a legacy hwaddr/inet/inet6 triple and a
// hardware-address/ip-addresses pair mirroring the guest-agent response
// shape; both are populated on the same response, so this models all
// five fields rather than picking one representation.
type Interface struct {
	// Name is the interface name, e.g. "eth0".
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// HWAddr is the interface's MAC address (legacy field name).
	HWAddr string `json:"hwaddr,omitempty" url:"hwaddr,omitempty"`
	// HardwareAddress is the interface's MAC address, identical to
	// HWAddr — Proxmox reports both under separate keys.
	HardwareAddress string `json:"hardware-address,omitempty" url:"hardware-address,omitempty"`
	// Inet is the interface's IPv4 address and subnet, e.g.
	// "10.0.3.2/24".
	Inet string `json:"inet,omitempty" url:"inet,omitempty"`
	// Inet6 is the interface's IPv6 address and subnet, e.g.
	// "fe80::be24:11ff:fe6f:0a35/64".
	Inet6 string `json:"inet6,omitempty" url:"inet6,omitempty"`
	// IPAddresses lists the interface's configured addresses in the
	// guest-agent-style representation (same information as
	// Inet/Inet6, split into typed entries).
	IPAddresses []IPAddress `json:"ip-addresses,omitempty" url:"ip-addresses,omitempty"`
}

// IPAddress is identical between this package and nodes/qemu/agent (see
// types/network.go's doc comment). It lives in the shared types package
// and is re-exported here as an alias so call sites read as
// lxc.IPAddress.
type IPAddress = types.IPAddress

// StartOptions holds the parameters for Client.Start
// (POST .../status/start).
type StartOptions struct {
	// Skiplock bypasses the container's configuration lock. Root only.
	Skiplock bool `url:"skiplock,omitempty"`
	// Debug enables very verbose debug log-level on start.
	Debug bool `url:"debug,omitempty"`
}

// StopOptions holds the parameters for Client.Stop
// (POST .../status/stop). Stop abruptly stops all processes running in
// the container; see ShutdownOptions for a graceful shutdown.
type StopOptions struct {
	// Skiplock bypasses the container's configuration lock. Root only.
	Skiplock bool `url:"skiplock,omitempty"`
	// OverruleShutdown aborts any active graceful-shutdown task for
	// this container before stopping it.
	OverruleShutdown bool `url:"overrule-shutdown,omitempty"`
}

// ShutdownOptions holds the parameters for Client.Shutdown
// (POST .../status/shutdown), a graceful shutdown (see lxc-stop(1)).
type ShutdownOptions struct {
	// Timeout is the maximum number of seconds to wait. Proxmox
	// defaults to 60.
	Timeout int `url:"timeout,omitempty"`
	// ForceStop falls back to an immediate stop if the graceful
	// shutdown doesn't complete.
	ForceStop bool `url:"forceStop,omitempty"`
}

// RebootOptions holds the parameters for Client.Reboot
// (POST .../status/reboot).
type RebootOptions struct {
	// Timeout is the maximum number of seconds to wait for the
	// shutdown half of the reboot.
	Timeout int `url:"timeout,omitempty"`
}
