package pools

import (
	"fmt"

	"github.com/sergelogvinov/proxmox/go-proxmox-rest/internal/params"
)

// Pool describes a resource pool as returned by GET /pools and
// GET /pools/{poolid}.
type Pool struct {
	// PoolID is the pool identifier, e.g. "my-pool".
	PoolID string `json:"poolid,omitempty"`
	// Comment is the pool description.
	Comment string `json:"comment,omitempty"`
	// Members lists the guests and storages assigned to the pool.
	Members []PoolMember `json:"members,omitempty"`
}

// PoolMember describes a guest or storage assigned to a pool.
type PoolMember struct {
	// ID is the full resource path, e.g. "vm/100" or "storage/local".
	ID string `json:"id,omitempty"`
	// Type is the resource type: "qemu", "lxc", "storage", ...
	Type string `json:"type,omitempty"`
	// VMID is the numeric guest ID (only for qemu/lxc members).
	VMID int `json:"vmid,omitempty"`
	// Name is the guest or storage name.
	Name string `json:"name,omitempty"`
	// Node is the node the guest runs on (only for qemu/lxc members).
	Node string `json:"node,omitempty"`
	// Storage is the storage identifier (only for storage members).
	Storage string `json:"storage,omitempty"`
	// Status is the guest status, e.g. "running" (only for qemu/lxc).
	Status string `json:"status,omitempty"`
	// Template is 1 if the guest is a template (only for qemu/lxc).
	Template int `json:"template,omitempty"`
	// CGroupMode is the cgroup mode of the guest (only for qemu/lxc).
	CGroupMode string `json:"cgroup-mode,omitempty"`
	// Tags is the comma-separated tag list of the guest.
	Tags string `json:"tags,omitempty"`
	// Uptime is the guest uptime in seconds (only for qemu/lxc).
	Uptime int `json:"uptime,omitempty"`
	// MaxCPU is the maximum CPU count (only for qemu/lxc).
	MaxCPU int `json:"maxcpu,omitempty"`
	// MaxMem is the maximum memory in bytes (only for qemu/lxc).
	MaxMem int `json:"maxmem,omitempty"`
	// Mem is the used memory in bytes (only for qemu/lxc).
	Mem int `json:"mem,omitempty"`
	// Disk is the used disk space in bytes (only for qemu/lxc).
	Disk int `json:"disk,omitempty"`
	// MaxDisk is the maximum disk space in bytes (only for qemu/lxc).
	MaxDisk int `json:"maxdisk,omitempty"`
	// CPU is the CPU usage fraction (only for qemu/lxc).
	CPU float64 `json:"cpu,omitempty"`
	// NetIn is the network traffic in bytes received (only for qemu/lxc).
	NetIn int `json:"netin,omitempty"`
	// NetOut is the network traffic in bytes sent (only for qemu/lxc).
	NetOut int `json:"netout,omitempty"`
	// HAMax is the high-availability maximum (only for qemu/lxc).
	HAMax int `json:"hamax,omitempty"`
	// HANode is the high-availability node (only for qemu/lxc).
	HANode string `json:"hanode,omitempty"`
	// HAState is the high-availability state (only for qemu/lxc).
	HAState string `json:"hastate,omitempty"`
	// Lock is the guest lock state (only for qemu/lxc).
	Lock string `json:"lock,omitempty"`
	// PluginType is the storage plugin type (only for storage members).
	PluginType string `json:"plugintype,omitempty"`
	// Shared is 1 if the storage is shared (only for storage members).
	Shared int `json:"shared,omitempty"`
	// StorageContent is the comma-separated content types of the storage.
	StorageContent string `json:"content,omitempty"`
}

// CreateOptions holds the parameters for POST /pools.
//
// Fields are encoded to form parameters via the `url` struct tags:
// zero values are omitted, []string is comma-joined.
type CreateOptions struct {
	// Comment is the pool description.
	Comment string `url:"comment"`
	// Members are the resources to add to the pool on creation, e.g.
	// "vm/100", "storage/local".
	Members []string `url:"vms"`
}

// encode converts the options to form parameters.
func (o *CreateOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("pools: create options are required")
	}

	return params.Encode(o)
}

// UpdateOptions holds the parameters for PUT /pools/{poolid}.
//
// Pointer fields are always sent when non-nil (even when zero-valued),
// which is how "clear a field" is expressed; non-pointer fields are
// omitted when zero.
type UpdateOptions struct {
	// Comment is the pool description. Use a pointer to distinguish
	// "unset" from "clear".
	Comment *string `url:"comment"`
	// Members is the full list of resources assigned to the pool, e.g.
	// "vm/100", "storage/local". The list replaces the current members.
	Members []string `url:"vms"`
}

// encode converts the options to form parameters.
func (o *UpdateOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("pools: update options are required")
	}

	return params.Encode(o)
}
