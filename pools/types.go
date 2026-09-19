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

package pools

import (
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Pool describes a resource pool as returned by GET /pools and
// GET /pools/{poolid}.
type Pool struct {
	// PoolID is the pool identifier, e.g. "my-pool".
	PoolID string `json:"poolid,omitempty" url:"poolid,omitempty"`
	// Comment is the pool description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
	// Members lists the guests and storages assigned to the pool.
	Members []PoolMember `json:"members,omitempty" url:"members,omitempty"`
}

// PoolMember describes a guest or storage assigned to a pool.
type PoolMember struct {
	// ID is the full resource path, e.g. "vm/100" or "storage/local".
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Type is the resource type: "qemu", "lxc", "storage", ...
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// VMID is the numeric guest ID (only for qemu/lxc members).
	VMID int `json:"vmid,omitempty" url:"vmid,omitempty"`
	// Name is the guest or storage name.
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Node is the node the guest runs on (only for qemu/lxc members).
	Node string `json:"node,omitempty" url:"node,omitempty"`
	// Storage is the storage identifier (only for storage members).
	Storage string `json:"storage,omitempty" url:"storage,omitempty"`
	// Status is the guest status, e.g. "running" (only for qemu/lxc).
	Status string `json:"status,omitempty" url:"status,omitempty"`
	// Template is 1 if the guest is a template (only for qemu/lxc).
	Template int `json:"template,omitempty" url:"template,omitempty"`
	// CGroupMode is the cgroup mode of the guest (only for qemu/lxc).
	CGroupMode string `json:"cgroup-mode,omitempty" url:"cgroup-mode,omitempty"`
	// Tags is the comma-separated tag list of the guest.
	Tags string `json:"tags,omitempty" url:"tags,omitempty"`
	// Uptime is the guest uptime in seconds (only for qemu/lxc).
	Uptime int `json:"uptime,omitempty" url:"uptime,omitempty"`
	// MaxCPU is the maximum CPU count (only for qemu/lxc).
	MaxCPU int `json:"maxcpu,omitempty" url:"maxcpu,omitempty"`
	// MaxMem is the maximum memory in bytes (only for qemu/lxc).
	MaxMem int `json:"maxmem,omitempty" url:"maxmem,omitempty"`
	// Mem is the used memory in bytes (only for qemu/lxc).
	Mem int `json:"mem,omitempty" url:"mem,omitempty"`
	// Disk is the used disk space in bytes (only for qemu/lxc).
	Disk int `json:"disk,omitempty" url:"disk,omitempty"`
	// MaxDisk is the maximum disk space in bytes (only for qemu/lxc).
	MaxDisk int `json:"maxdisk,omitempty" url:"maxdisk,omitempty"`
	// CPU is the CPU usage fraction (only for qemu/lxc).
	CPU float64 `json:"cpu,omitempty" url:"cpu,omitempty"`
	// NetIn is the network traffic in bytes received (only for qemu/lxc).
	NetIn int `json:"netin,omitempty" url:"netin,omitempty"`
	// NetOut is the network traffic in bytes sent (only for qemu/lxc).
	NetOut int `json:"netout,omitempty" url:"netout,omitempty"`
	// HAMax is the high-availability maximum (only for qemu/lxc).
	HAMax int `json:"hamax,omitempty" url:"hamax,omitempty"`
	// HANode is the high-availability node (only for qemu/lxc).
	HANode string `json:"hanode,omitempty" url:"hanode,omitempty"`
	// HAState is the high-availability state (only for qemu/lxc).
	HAState string `json:"hastate,omitempty" url:"hastate,omitempty"`
	// Lock is the guest lock state (only for qemu/lxc).
	Lock string `json:"lock,omitempty" url:"lock,omitempty"`
	// PluginType is the storage plugin type (only for storage members).
	PluginType string `json:"plugintype,omitempty" url:"plugintype,omitempty"`
	// Shared is 1 if the storage is shared (only for storage members).
	Shared int `json:"shared,omitempty" url:"shared,omitempty"`
	// StorageContent is the comma-separated content types of the storage.
	StorageContent string `json:"content,omitempty" url:"content,omitempty"`
}

// CreateOptions holds the write parameters for POST /pools (Create).
// Proxmox's create endpoint accepts only a comment — a pool starts out
// empty; use Client.Update to add members afterwards.
type CreateOptions struct {
	// Comment is the pool description.
	Comment string `url:"comment,omitempty"`
}

// encode converts the options to form parameters.
func (o *CreateOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("pools: create options are required")
	}

	return params.Encode(o)
}

// UpdateOptions holds the write parameters for PUT /pools/{poolid}
// (Update).
//
// VMIDs and Storage are additive, not a replacement list: Proxmox adds
// the named guests/storages to the pool, or — when Remove is true —
// removes them instead. There is no "replace the whole member list"
// operation; to fully replace membership, issue a Remove call for the
// old members and a separate add call for the new ones.
type UpdateOptions struct {
	// Comment replaces the pool description. Use a pointer to
	// distinguish "leave unchanged" (nil) from "clear" (pointer to "").
	Comment *string `url:"comment"`
	// VMIDs lists guest VMIDs to add to (or, with Remove, remove from)
	// the pool.
	VMIDs []int `url:"vms,omitempty"`
	// Storage lists storage IDs to add to (or, with Remove, remove
	// from) the pool.
	Storage []string `url:"storage,omitempty"`
	// AllowMove lets a guest already assigned to another pool be moved
	// into this one (removed from its current pool) instead of Proxmox
	// rejecting the request.
	AllowMove *bool `url:"allow-move"`
	// Remove, instead of adding VMIDs/Storage to the pool, removes them
	// from it.
	Remove *bool `url:"delete"`
}

// encode converts the options to form parameters.
func (o *UpdateOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("pools: update options are required")
	}

	return params.Encode(o)
}
