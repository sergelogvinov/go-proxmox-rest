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

package agent

import "github.com/sergelogvinov/go-proxmox-rest/types"

// IPAddress is identical between this package and nodes/lxc (see
// types/network.go's doc comment). It lives in the shared types package
// and is re-exported here as an alias so call sites keep reading as
// agent.IPAddress.
type IPAddress = types.IPAddress

// Info describes the guest agent's own version and the commands it
// supports, as returned by Client.Info (guest-info).
type Info struct {
	// Version is the guest agent's version string.
	Version string `json:"version,omitempty" url:"version,omitempty"`
	// SupportedCommands lists every command the agent recognizes and
	// whether each is currently enabled.
	SupportedCommands []SupportedCommand `json:"supported_commands,omitempty" url:"supported_commands,omitempty"`
}

// SupportedCommand describes one guest agent command's availability.
type SupportedCommand struct {
	// Name is the QGA command name, e.g. "guest-ping".
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Enabled is true if the command is currently allowed to run.
	Enabled bool `json:"enabled,omitempty" url:"enabled,omitempty"`
	// SuccessResponse is true if the command returns a value on
	// success (rather than an empty response).
	SuccessResponse bool `json:"success-response,omitempty" url:"success-response,omitempty"`
}

// FSFreezeState is a guest filesystem's freeze state, as returned by
// Client.FSFreezeStatus.
type FSFreezeState string

const (
	FSFreezeStateThawed FSFreezeState = "thawed"
	FSFreezeStateFrozen FSFreezeState = "frozen"
)

// FSTrimResult describes the outcome of Client.FSTrim, one entry per
// trimmed filesystem path.
type FSTrimResult struct {
	Paths []FSTrimPathResult `json:"paths,omitempty" url:"paths,omitempty"`
}

// FSTrimPathResult is a single filesystem's fstrim outcome.
type FSTrimPathResult struct {
	// Path is the filesystem's mount point.
	Path string `json:"path,omitempty" url:"path,omitempty"`
	// Error holds the error message if trimming this path failed;
	// empty on success.
	Error string `json:"error,omitempty" url:"error,omitempty"`
	// Trimmed is the number of bytes actually discarded.
	Trimmed int64 `json:"trimmed,omitempty" url:"trimmed,omitempty"`
	// Minimum is the minimum contiguous free range, in bytes, the
	// guest kernel considered for discarding.
	Minimum int64 `json:"minimum,omitempty" url:"minimum,omitempty"`
}

// NetworkInterface describes one guest network interface, as returned
// by Client.NetworkGetInterfaces.
type NetworkInterface struct {
	// Name is the interface name, e.g. "eth0".
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// HardwareAddress is the interface's MAC address.
	HardwareAddress string `json:"hardware-address,omitempty" url:"hardware-address,omitempty"`
	// IPAddresses lists the interface's configured addresses.
	IPAddresses []IPAddress `json:"ip-addresses,omitempty" url:"ip-addresses,omitempty"`
	// Statistics holds the interface's traffic counters, when the
	// guest agent reports them.
	Statistics *NetworkInterfaceStats `json:"statistics,omitempty" url:"statistics,omitempty"`
}

// NetworkInterfaceStats holds a guest network interface's traffic
// counters.
type NetworkInterfaceStats struct {
	RXBytes   int64 `json:"rx-bytes,omitempty"   url:"rx-bytes,omitempty"`
	RXPackets int64 `json:"rx-packets,omitempty" url:"rx-packets,omitempty"`
	RXErrs    int64 `json:"rx-errs,omitempty"    url:"rx-errs,omitempty"`
	RXDropped int64 `json:"rx-dropped,omitempty" url:"rx-dropped,omitempty"`
	TXBytes   int64 `json:"tx-bytes,omitempty"   url:"tx-bytes,omitempty"`
	TXPackets int64 `json:"tx-packets,omitempty" url:"tx-packets,omitempty"`
	TXErrs    int64 `json:"tx-errs,omitempty"    url:"tx-errs,omitempty"`
	TXDropped int64 `json:"tx-dropped,omitempty" url:"tx-dropped,omitempty"`
}

// VCPU describes one guest virtual CPU's hotplug state, as returned by
// Client.GetVCPUs.
type VCPU struct {
	// LogicalID is the vCPU's logical index.
	LogicalID int `json:"logical-id,omitempty" url:"logical-id,omitempty"`
	// Online is true if the vCPU is currently online.
	Online bool `json:"online,omitempty" url:"online,omitempty"`
	// CanOffline is true if the guest OS supports offlining this vCPU.
	CanOffline bool `json:"can-offline,omitempty" url:"can-offline,omitempty"`
}

// FSInfo describes one mounted guest filesystem, as returned by
// Client.GetFSInfo.
type FSInfo struct {
	// Name is the filesystem's guest-side device name.
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Mountpoint is the filesystem's mount path.
	Mountpoint string `json:"mountpoint,omitempty" url:"mountpoint,omitempty"`
	// Type is the filesystem type, e.g. "ext4".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// UsedBytes is the used space in bytes, when the guest agent
	// reports it.
	UsedBytes int64 `json:"used-bytes,omitempty" url:"used-bytes,omitempty"`
	// TotalBytes is the total space in bytes, when the guest agent
	// reports it.
	TotalBytes int64 `json:"total-bytes,omitempty" url:"total-bytes,omitempty"`
	// Disk lists the underlying block device(s) backing this
	// filesystem.
	Disk []FSDisk `json:"disk,omitempty" url:"disk,omitempty"`
}

// FSDisk identifies one block device backing a guest filesystem.
type FSDisk struct {
	// BusType is the device's bus type, e.g. "scsi", "virtio".
	BusType string `json:"bus-type,omitempty" url:"bus-type,omitempty"`
	Bus     int    `json:"bus,omitempty"      url:"bus,omitempty"`
	Unit    int    `json:"unit,omitempty"     url:"unit,omitempty"`
	Target  int    `json:"target,omitempty"   url:"target,omitempty"`
	// Dev is the device's guest-side path, e.g. "/dev/sda1".
	Dev string `json:"dev,omitempty" url:"dev,omitempty"`
	// Serial is the device's serial number, when available.
	Serial string `json:"serial,omitempty" url:"serial,omitempty"`
}

// MemoryBlock describes one guest memory block's hotplug state, as
// returned by Client.GetMemoryBlocks.
type MemoryBlock struct {
	// PhysIndex is the memory block's physical index.
	PhysIndex int `json:"phys-index,omitempty" url:"phys-index,omitempty"`
	// Online is true if the memory block is currently online.
	Online bool `json:"online,omitempty" url:"online,omitempty"`
	// CanOffline is true if the guest OS supports offlining this
	// memory block.
	CanOffline bool `json:"can-offline,omitempty" url:"can-offline,omitempty"`
}

// MemoryBlockInfo describes the guest's memory block size, as returned
// by Client.GetMemoryBlockInfo.
type MemoryBlockInfo struct {
	// Size is the size of one memory block, in bytes.
	Size int64 `json:"size,omitempty" url:"size,omitempty"`
}

// HostnameInfo holds the guest's host name, as returned by
// Client.GetHostname.
type HostnameInfo struct {
	Hostname string `json:"host-name,omitempty" url:"host-name,omitempty"`
}

// OSInfo describes the guest operating system, as returned by
// Client.GetOSInfo. Every field is optional — not every guest OS/agent
// version reports all of them.
type OSInfo struct {
	ID            string `json:"id,omitempty"             url:"id,omitempty"`
	Name          string `json:"name,omitempty"           url:"name,omitempty"`
	PrettyName    string `json:"pretty-name,omitempty"    url:"pretty-name,omitempty"`
	Version       string `json:"version,omitempty"        url:"version,omitempty"`
	VersionID     string `json:"version-id,omitempty"     url:"version-id,omitempty"`
	KernelRelease string `json:"kernel-release,omitempty" url:"kernel-release,omitempty"`
	KernelVersion string `json:"kernel-version,omitempty" url:"kernel-version,omitempty"`
	Machine       string `json:"machine,omitempty"        url:"machine,omitempty"`
	Variant       string `json:"variant,omitempty"        url:"variant,omitempty"`
	VariantID     string `json:"variant-id,omitempty"     url:"variant-id,omitempty"`
}

// User describes one logged-in guest user, as returned by
// Client.GetUsers.
type User struct {
	// User is the username.
	User string `json:"user,omitempty" url:"user,omitempty"`
	// Domain is the user's login domain (Windows guests only).
	Domain string `json:"domain,omitempty" url:"domain,omitempty"`
	// LoginTime is the login time, in seconds since the Unix epoch.
	LoginTime float64 `json:"login-time,omitempty" url:"login-time,omitempty"`
}

// Timezone describes the guest's configured time zone, as returned by
// Client.GetTimezone.
type Timezone struct {
	// Zone is the time zone name, e.g. "Europe/Vienna"; empty if the
	// guest agent could only report the offset.
	Zone string `json:"zone,omitempty" url:"zone,omitempty"`
	// Offset is the UTC offset in seconds.
	Offset int `json:"offset,omitempty" url:"offset,omitempty"`
}
