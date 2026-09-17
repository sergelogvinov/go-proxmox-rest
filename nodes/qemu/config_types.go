package qemu

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/sergelogvinov/go-proxmox-rest/internal/property"
)

// Startup describes the VM startup and shutdown ordering. Proxmox accepts the
// order either as a bare first number or as order=<number>, followed by
// optional up/down delays in seconds.
type Startup struct {
	Order *int `cfg:"order,omitempty,default"`
	Up    *int `cfg:"up,omitempty"`
	Down  *int `cfg:"down,omitempty"`
}

// String converts the startup configuration to Proxmox's property-string
// format.
func (s Startup) String() string {
	value, _ := property.Marshal(s)
	return value
}

// UnmarshalJSON converts Proxmox's startup property string into Startup.
func (s *Startup) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("qemu: startup must be a property string: %w", err)
	}

	*s = Startup{}
	return property.Unmarshal(value, s)
}

// SMBios1 describes the SMBIOS type 1 property string used by Proxmox.
// String fields contain the values in the form expected by Proxmox. When
// Base64 is enabled, Proxmox expects those values to be base64 encoded; this
// type deliberately does not decode them so values can be round-tripped.
type SMBios1 struct {
	Base64       *bool  `cfg:"base64,omitempty"`
	Family       string `cfg:"family,omitempty"`
	Manufacturer string `cfg:"manufacturer,omitempty"`
	Product      string `cfg:"product,omitempty"`
	Serial       string `cfg:"serial,omitempty"`
	SKU          string `cfg:"sku,omitempty"`
	UUID         string `cfg:"uuid,omitempty"`
	Version      string `cfg:"version,omitempty"`
}

// AMDSev describes AMD Secure Encrypted Virtualization settings.
type AMDSev struct {
	Type         string `cfg:"type,omitempty,default"`
	AllowSMT     *bool  `cfg:"allow-smt,omitempty"`
	KernelHashes *bool  `cfg:"kernel-hashes,omitempty"`
	NoDebug      *bool  `cfg:"no-debug,omitempty"`
	NoKeySharing *bool  `cfg:"no-key-sharing,omitempty"`
}

// String converts AMD SEV settings to Proxmox's property-string format.
func (s AMDSev) String() string {
	value, _ := property.Marshal(s)
	return value
}

// UnmarshalJSON converts Proxmox's AMD SEV property string into AMDSev.
func (s *AMDSev) UnmarshalJSON(data []byte) error {
	return unmarshalPropertyJSON(data, s, "amd-sev")
}

// CPU describes the emulated CPU type and its flags.
type CPU struct {
	Type          string   `cfg:"cputype,omitempty,default"`
	Flags         []string `cfg:"flags,omitempty"`
	GuestPhysBits *int     `cfg:"guest-phys-bits,omitempty"`
	Hidden        *bool    `cfg:"hidden,omitempty"`
	HVVendorID    string   `cfg:"hv-vendor-id,omitempty"`
	Level         *int     `cfg:"level,omitempty"`
	PhysBits      string   `cfg:"phys-bits,omitempty"`
	ReportedModel string   `cfg:"reported-model,omitempty"`
}

// String converts CPU settings to Proxmox's property-string format.
func (s CPU) String() string {
	value, _ := property.Marshal(s)
	return value
}

// UnmarshalJSON converts Proxmox's CPU property string into CPU.
func (s *CPU) UnmarshalJSON(data []byte) error {
	return unmarshalPropertyJSON(data, s, "cpu")
}

// Memory describes the guest's memory configuration. Proxmox accepts the
// current amount either as a bare first number or as current=<number>, in
// MiB.
type Memory struct {
	Current *int `cfg:"current,omitempty,default"`
	Max     *int `cfg:"max,omitempty"`
	Min     *int `cfg:"min,omitempty"`
}

// String converts memory settings to Proxmox's property-string format.
func (s Memory) String() string {
	value, _ := property.Marshal(s)
	return value
}

// UnmarshalJSON converts Proxmox's memory property string into Memory.
func (s *Memory) UnmarshalJSON(data []byte) error {
	return unmarshalPropertyJSON(data, s, "memory")
}

// IntelTDX describes Intel Trust Domain Extensions settings.
type IntelTDX struct {
	Type        string `cfg:"type,omitempty,default"`
	Attestation *bool  `cfg:"attestation,omitempty"`
	VsockCID    *int   `cfg:"vsock-cid,omitempty"`
	VsockPort   *int   `cfg:"vsock-port,omitempty"`
}

// String converts Intel TDX settings to Proxmox's property-string format.
func (s IntelTDX) String() string {
	value, _ := property.Marshal(s)
	return value
}

// UnmarshalJSON converts Proxmox's Intel TDX property string into IntelTDX.
func (s *IntelTDX) UnmarshalJSON(data []byte) error {
	return unmarshalPropertyJSON(data, s, "intel-tdx")
}

func unmarshalPropertyJSON(data []byte, target any, name string) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("qemu: %s must be a property string: %w", name, err)
	}

	rv := reflect.ValueOf(target)
	rv.Elem().Set(reflect.Zero(rv.Elem().Type()))
	return property.Unmarshal(value, target)
}

// String converts SMBIOS type 1 fields to Proxmox's property-string format.
func (s SMBios1) String() string {
	value, _ := property.Marshal(s)
	return value
}

// UnmarshalJSON converts Proxmox's SMBIOS property string into SMBios1.
func (s *SMBios1) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("qemu: smbios1 must be a property string: %w", err)
	}

	*s = SMBios1{}
	return property.Unmarshal(value, s)
}

// Config describes a QEMU guest's configuration, as returned by
// GET /nodes/{node}/qemu/{vmid}/config and accepted by
// Client.UpdateConfig/UpdateConfigAsync (PUT/POST to the same path).
//
// Proxmox's VM config is dominated by numerically-suffixed hardware
// attachment families (net0, net1, ..., ide0-3, sata0-5, scsi0-30,
// virtio0-15, unused0-255, usb0-13, hostpci0-15, serial0-3, parallel0-2,
// numa0-7, virtiofs0-N, ipconfig0-31): rather than a field per possible
// index (thousands of fields, most always unset), each family is a
// map[int]string keyed by index, holding Proxmox's own property-string
// value verbatim (e.g. Net[0] == "virtio=AA:BB:...,bridge=vmbr0").
// Parsing those property strings further (disk size/format/cache flags,
// NIC model/bridge/vlan, ...) is out of scope here — see
// PVE::QemuServer::Drive / ::Network for the authoritative grammar.
//
// Fields tagged "readonly" are populated from a GET response but never
// sent by Update/UpdateAsync, even if a caller round-trips a fetched
// Config back through them — they are internal/snapshot-only state
// (parent snapshot, running machine/cpu type, ...) that Proxmox computes
// itself and does not accept as write input.
type Config struct {
	// -- lifecycle / identity --

	// Name is the guest's name, shown in the web UI only.
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Description is the guest's description, shown in the web UI's
	// summary panel and saved as a comment inside the config file.
	Description string `json:"description,omitempty" url:"description,omitempty"`
	// Tags is the guest's tag list (meta information only).
	Tags []string `json:"tags,omitempty" url:"tags,omitempty"`
	// OSType selects guest-OS-specific optimizations, e.g. "l26"
	// (Linux 2.6+), "win10", "other".
	OSType *string `json:"ostype,omitempty" url:"ostype,omitempty"`
	// Template marks the guest as a template (see also Client.Template,
	// which performs the conversion).
	Template *bool `json:"template,omitempty" url:"template,omitempty"`
	// Protection disables remove-VM and remove-disk operations when set.
	Protection *bool `json:"protection,omitempty" url:"protection,omitempty"`
	// Lock is the current lock holder, if any (e.g. "backup",
	// "migrate", "clone"). Include "lock" in Delete to force-unlock a
	// stuck guest.
	Lock string `json:"lock,omitempty" url:"lock,omitempty"`
	// Meta is read-only meta-information Proxmox maintains about the
	// guest (e.g. its creation timestamp).
	Meta string `json:"meta,omitempty" url:"meta,omitempty,readonly"`
	// Digest is the configuration file's SHA1 digest. Set it on a write
	// to abort if the configuration changed concurrently since the
	// value was read.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`

	// -- boot / firmware --

	// Boot is the guest's boot order, e.g. "order=scsi0;ide2;net0".
	Boot *string `json:"boot,omitempty" url:"boot,omitempty"`
	// BIOS selects the BIOS implementation: "seabios" (default) or
	// "ovmf" (UEFI).
	BIOS *string `json:"bios,omitempty" url:"bios,omitempty"`
	// Machine is the QEMU machine type, e.g. "q35" or
	// "pc-i440fx-9.0+pve0".
	Machine *string `json:"machine,omitempty" url:"machine,omitempty"`
	// Arch is the guest CPU architecture, e.g. "x86_64", "aarch64".
	Arch *string `json:"arch,omitempty" url:"arch,omitempty"`
	// SMBios1 holds SMBIOS type 1 fields (UUID, serial, ...) as a
	// property string.
	SMBios1 *SMBios1 `json:"smbios1,omitempty" url:"smbios1,omitempty"`
	// VMGenID sets the VM Generation ID device; "1" autogenerates one,
	// "0" disables it explicitly.
	VMGenID *string `json:"vmgenid,omitempty" url:"vmgenid,omitempty"`
	// HookScript is the volume id of a script run at various points in
	// the guest's lifetime.
	HookScript *string `json:"hookscript,omitempty" url:"hookscript,omitempty"`
	// Args is arbitrary extra kvm command-line arguments. Experts only.
	Args *string `json:"args,omitempty" url:"args,omitempty"`
	// StartDate sets the RTC's initial date: "now", "YYYY-MM-DD", or
	// "YYYY-MM-DDTHH:MM:SS".
	StartDate *string `json:"startdate,omitempty" url:"startdate,omitempty"`
	// Startup is the guest's startup/shutdown ordering, e.g.
	// "order=2,up=30,down=60".
	Startup *Startup `json:"startup,omitempty" url:"startup,omitempty"`
	// OnBoot starts the guest automatically at host boot.
	OnBoot *bool `json:"onboot,omitempty" url:"onboot,omitempty"`
	// Reboot allows a guest-initiated reboot; if false the guest exits
	// instead of rebooting.
	Reboot *bool `json:"reboot,omitempty" url:"reboot,omitempty"`
	// Freeze starts the guest's CPU frozen (resume with the QEMU
	// monitor's "c" command).
	Freeze *bool `json:"freeze,omitempty" url:"freeze,omitempty"`
	// Autostart enables an automatic restart after a crash.
	Autostart *bool `json:"autostart,omitempty" url:"autostart,omitempty"`

	// -- CPU / memory --

	// Sockets is the number of CPU sockets.
	Sockets *int `json:"sockets,omitempty" url:"sockets,omitempty"`
	// Cores is the number of cores per socket.
	Cores *int `json:"cores,omitempty" url:"cores,omitempty"`
	// SMP is the total CPU count; prefer Sockets instead.
	SMP *int `json:"smp,omitempty" url:"smp,omitempty"`
	// VCPUs is the number of hotplugged vCPUs.
	VCPUs *int `json:"vcpus,omitempty" url:"vcpus,omitempty"`
	// CPU is the emulated CPU type, e.g. "host" or "kvm64".
	CPU *CPU `json:"cpu,omitempty" url:"cpu,omitempty"`
	// CPULimit caps CPU usage (in host CPUs); 0 means unlimited.
	CPULimit *float64 `json:"cpulimit,omitempty" url:"cpulimit,omitempty"`
	// CPUUnits is the guest's CPU scheduling weight, relative to other
	// running guests.
	CPUUnits *int `json:"cpuunits,omitempty" url:"cpuunits,omitempty"`
	// Affinity lists host cores used to execute guest processes, e.g.
	// "0,5,8-11".
	Affinity string `json:"affinity,omitempty" url:"affinity,omitempty"`
	// AMDSev configures AMD SEV confidential-computing features.
	AMDSev *AMDSev `json:"amd-sev,omitempty" url:"amd-sev,omitempty"`
	// IntelTDX configures Intel TDX confidential-computing features.
	IntelTDX *IntelTDX `json:"intel-tdx,omitempty" url:"intel-tdx,omitempty"`
	// Memory is the guest's memory configuration in MiB, e.g.
	// "2048" or "current=2048,max=8192".
	Memory *Memory `json:"memory,omitempty" url:"memory,omitempty"`
	// Balloon is the target balloon RAM in MiB; 0 disables ballooning.
	Balloon *int `json:"balloon,omitempty" url:"balloon,omitempty"`
	// Shares is the guest's memory-sharing weight for auto-ballooning;
	// 0 disables auto-ballooning for this guest.
	Shares *int `json:"shares,omitempty" url:"shares,omitempty"`
	// NUMA enables/disables NUMA (not to be confused with the NUMA
	// field below, which configures individual NUMA nodes).
	NUMAEnabled *bool `json:"numa,omitempty" url:"numa,omitempty"`
	// Hugepages sets the guest's hugepage size: "any", "2", or "1024"
	// (MiB).
	Hugepages *string `json:"hugepages,omitempty" url:"hugepages,omitempty"`
	// KeepHugepages keeps hugepages allocated after guest shutdown, for
	// reuse on a subsequent start.
	KeepHugepages *bool `json:"keephugepages,omitempty" url:"keephugepages,omitempty"`
	// AllowKSM allows this guest's memory pages to be merged via KSM.
	AllowKSM *bool `json:"allow-ksm,omitempty" url:"allow-ksm,omitempty"`

	// -- virtualization / platform --

	// KVM enables/disables hardware virtualization.
	KVM *bool `json:"kvm,omitempty" url:"kvm,omitempty"`
	// ACPI enables/disables ACPI.
	ACPI *bool `json:"acpi,omitempty" url:"acpi,omitempty"`
	// TDF enables/disables the time-drift fix.
	TDF *bool `json:"tdf,omitempty" url:"tdf,omitempty"`
	// LocalTime sets the RTC to local time instead of UTC.
	LocalTime *bool `json:"localtime,omitempty" url:"localtime,omitempty"`
	// Agent configures the QEMU Guest Agent as a property string, e.g.
	// "1" or "1,fstrim_cloned_disks=1".
	Agent string `json:"agent,omitempty" url:"agent,omitempty"`
	// Watchdog configures a virtual hardware watchdog device as a
	// property string.
	Watchdog string `json:"watchdog,omitempty" url:"watchdog,omitempty"`
	// VGA configures the VGA hardware as a property string, e.g.
	// "std" or "qxl,memory=32".
	VGA string `json:"vga,omitempty" url:"vga,omitempty"`
	// Tablet enables/disables the USB tablet device (absolute mouse
	// positioning for VNC).
	Tablet *bool `json:"tablet,omitempty" url:"tablet,omitempty"`
	// Keyboard is the VNC server's keyboard layout. Rarely needed.
	Keyboard string `json:"keyboard,omitempty" url:"keyboard,omitempty"`
	// SCSIHW is the SCSI controller model, e.g. "virtio-scsi-single".
	SCSIHW string `json:"scsihw,omitempty" url:"scsihw,omitempty"`
	// IVSHMem configures inter-VM shared memory as a property string.
	IVSHMem string `json:"ivshmem,omitempty" url:"ivshmem,omitempty"`
	// Audio0 configures an audio device as a property string.
	Audio0 string `json:"audio0,omitempty" url:"audio0,omitempty"`
	// SpiceEnhancements configures additional SPICE features as a
	// property string.
	SpiceEnhancements string `json:"spice_enhancements,omitempty" url:"spice_enhancements,omitempty"`
	// RNG0 configures a VirtIO random number generator as a property
	// string.
	RNG0 string `json:"rng0,omitempty" url:"rng0,omitempty"`
	// CDROM is an alias for the ide2 drive.
	CDROM string `json:"cdrom,omitempty" url:"cdrom,omitempty"`
	// HotPlug selectively enables hotplug features, e.g.
	// "network,disk,usb"; "0" disables hotplug entirely.
	HotPlug string `json:"hotplug,omitempty" url:"hotplug,omitempty"`

	// -- migration --

	// MigrateSpeed caps migration bandwidth in MB/s; 0 means no limit.
	MigrateSpeed int `json:"migrate_speed,omitempty" url:"migrate_speed,omitempty"`
	// MigrateDowntime caps tolerated migration downtime in seconds.
	MigrateDowntime float64 `json:"migrate_downtime,omitempty" url:"migrate_downtime,omitempty"`
	// VMStateStorage is the default storage for VM state
	// volumes/files (used by suspend-to-disk).
	VMStateStorage string `json:"vmstatestorage,omitempty" url:"vmstatestorage,omitempty"`

	// -- cloud-init --

	// CIType selects the cloud-init data format: "configdrive2",
	// "nocloud", or "opennebula".
	CIType string `json:"citype,omitempty" url:"citype,omitempty"`
	// CIUser overrides the cloud image's default user for SSH
	// keys/password.
	CIUser string `json:"ciuser,omitempty" url:"ciuser,omitempty"`
	// CIPassword sets the cloud-init user's password. GET responses
	// redact this to "**********" — do not round-trip a fetched
	// Config's CIPassword back through Update/UpdateAsync, or the
	// redacted placeholder overwrites the real password.
	CIPassword string `json:"cipassword,omitempty" url:"cipassword,omitempty"`
	// CIUpgrade runs an automatic package upgrade on first boot.
	CIUpgrade bool `json:"ciupgrade,omitempty" url:"ciupgrade,omitempty"`
	// CICustom points to custom cloud-init files that replace the
	// automatically generated ones, as a property string.
	CICustom string `json:"cicustom,omitempty" url:"cicustom,omitempty"`
	// SearchDomain sets the cloud-init DNS search domain.
	SearchDomain string `json:"searchdomain,omitempty" url:"searchdomain,omitempty"`
	// Nameserver sets the cloud-init DNS server address(es).
	Nameserver string `json:"nameserver,omitempty" url:"nameserver,omitempty"`
	// SSHKeys sets the cloud-init authorized SSH public keys
	// (urlencoded, one key per line).
	SSHKeys string `json:"sshkeys,omitempty" url:"sshkeys,omitempty"`

	// -- internal / snapshot-only (read-only) --

	// Parent is the parent snapshot's name.
	Parent string `json:"parent,omitempty" url:"parent,omitempty,readonly"`
	// SnapTime is the snapshot's creation timestamp.
	SnapTime int64 `json:"snaptime,omitempty" url:"snaptime,omitempty,readonly"`
	// VMState references the volume holding suspended VM state.
	VMState string `json:"vmstate,omitempty" url:"vmstate,omitempty,readonly"`
	// RunningMachine is the QEMU machine type the guest is/was running
	// with.
	RunningMachine string `json:"runningmachine,omitempty" url:"runningmachine,omitempty,readonly"`
	// RunningCPU is the QEMU "-cpu" parameter the guest is/was running
	// with.
	RunningCPU string `json:"runningcpu,omitempty" url:"runningcpu,omitempty,readonly"`
	// RunningNetsHostMTU lists VirtIO NICs and their effective
	// host_mtu setting at snapshot time.
	RunningNetsHostMTU string `json:"running-nets-host-mtu,omitempty" url:"running-nets-host-mtu,omitempty,readonly"`

	// -- numerically-indexed hardware families, keyed by index --

	// Net holds netN entries (network interfaces), N in 0-31.
	Net map[int]string `json:"-" url:"-"`
	// IDE holds ideN entries (IDE drives), N in 0-3.
	IDE map[int]string `json:"-" url:"-"`
	// SATA holds sataN entries (SATA drives), N in 0-5.
	SATA map[int]string `json:"-" url:"-"`
	// SCSI holds scsiN entries (SCSI drives), N in 0-30.
	SCSI map[int]string `json:"-" url:"-"`
	// VirtIO holds virtioN entries (VirtIO block drives), N in 0-15.
	VirtIO map[int]string `json:"-" url:"-"`
	// VirtioFS holds virtiofsN entries (virtiofs shares).
	VirtioFS map[int]string `json:"-" url:"-"`
	// Unused holds unusedN entries (disks detached from the config but
	// not deleted), N in 0-255.
	Unused map[int]string `json:"-" url:"-"`
	// USB holds usbN entries (USB device passthrough), N in 0-13.
	USB map[int]string `json:"-" url:"-"`
	// HostPCI holds hostpciN entries (PCI(e) device passthrough), N in
	// 0-15.
	HostPCI map[int]string `json:"-" url:"-"`
	// Serial holds serialN entries (serial devices), N in 0-3.
	Serial map[int]string `json:"-" url:"-"`
	// Parallel holds parallelN entries (parallel ports, deprecated), N
	// in 0-2.
	Parallel map[int]string `json:"-" url:"-"`
	// IPConfig holds ipconfigN entries (cloud-init per-NIC IP
	// configuration), N in 0-31.
	IPConfig map[int]string `json:"-" url:"-"`
	// NUMA holds numaN entries (per-NUMA-node CPU/memory pinning), N in
	// 0-7. Not to be confused with NUMAEnabled above.
	NUMA map[int]string `json:"-" url:"-"`

	// -- write-only (never appear in a GET response) --

	// Delete lists config keys to reset to their default value, e.g.
	// []string{"description", "net0"}.
	Delete []string `json:"-" url:"delete,omitempty,writeonly"`
	// Revert lists pending changes to discard, reverting them to their
	// current value.
	Revert []string `json:"-" url:"revert,omitempty,writeonly"`
	// Force allows a Delete of a currently in-use resource (a mounted
	// disk, an active NIC). Only meaningful together with Delete.
	Force bool `json:"-" url:"force,omitempty,writeonly"`
	// Skiplock bypasses the guest's configuration lock. Root only.
	Skiplock bool `json:"-" url:"skiplock,omitempty,writeonly"`
	// BackgroundDelay is the number of seconds (1-30) to wait for
	// UpdateConfigAsync's task to finish before falling back to
	// returning its UPID. UpdateConfigAsync (POST) only — UpdateConfig
	// (PUT) rejects this parameter.
	BackgroundDelay int `json:"-" url:"background_delay,omitempty,writeonly"`
	// ImportWorkingStorage is the file-based storage used as
	// intermediary extraction storage during an OVA/OVF import.
	// UpdateConfigAsync (POST) only — UpdateConfig (PUT) rejects this
	// parameter.
	ImportWorkingStorage string `json:"-" url:"import-working-storage,omitempty,writeonly"`
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
