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

package cluster

import (
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
	"github.com/sergelogvinov/go-proxmox-rest/types"
)

// Status contains the cluster status returned by GET /cluster/status.
//
// The response mixes two kinds of entries in a single list: the overall
// cluster info (type "cluster") and one entry per node (type "node").
type Status struct {
	// ID is the cluster name (only set on the "cluster" entry).
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Name is the cluster name (only set on the "cluster" entry).
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Quorate indicates whether the cluster has quorum
	// (only set on the "cluster" entry).
	Quorate int `json:"quorate,omitempty" url:"quorate,omitempty"`
	// Version is the cluster configuration version
	// (only set on the "cluster" entry).
	Version int `json:"version,omitempty" url:"version,omitempty"`

	// Nodes is the list of cluster member nodes (type "node").
	Nodes []NodeStatus `json:"nodes,omitempty" url:"nodes,omitempty"`
}

// NodeStatus describes a single cluster member node.
type NodeStatus struct {
	// ID is the node ID, e.g. "node/pve1".
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Type is always "node".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Level is the node's cluster membership level (0 or 1).
	Level string `json:"level,omitempty" url:"level,omitempty"`

	// NodeID is the numeric cluster node identifier.
	NodeID int `json:"nodeid,omitempty" url:"nodeid,omitempty"`
	// Name is the node hostname, e.g. "pve1".
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// IP is the node's cluster communication IP address.
	IP string `json:"ip,omitempty" url:"ip,omitempty"`
	// Online is 1 if the node is online, 0 otherwise.
	Online int `json:"online,omitempty" url:"online,omitempty"`
	// Local is 1 if this node is the one serving the request.
	Local int `json:"local,omitempty" url:"local,omitempty"`
}

// LogEntry describes a single cluster log entry returned by
// GET /cluster/log.
type LogEntry struct {
	// ID uniquely identifies the entry: "<uid>:<node>".
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// UID is the entry's sequence number within the node's log ring
	// buffer (not globally unique across nodes; combine with Node, or
	// use ID, for a unique key).
	UID int64 `json:"uid,omitempty" url:"uid,omitempty"`
	// Time is the UNIX timestamp the entry was logged at.
	Time int64 `json:"time,omitempty" url:"time,omitempty"`
	// Priority is the syslog priority level (0 emerg .. 7 debug).
	Priority int `json:"pri,omitempty" url:"pri,omitempty"`
	// Tag is the originating service/tag, e.g. "pvedaemon".
	Tag string `json:"tag,omitempty" url:"tag,omitempty"`
	// PID is the process ID that logged the entry.
	PID int `json:"pid,omitempty" url:"pid,omitempty"`
	// Node is the node the entry was logged on.
	Node string `json:"node,omitempty" url:"node,omitempty"`
	// User is the user that triggered the entry, e.g. "root@pam".
	User string `json:"user,omitempty" url:"user,omitempty"`
	// Message is the log message text.
	Message string `json:"msg,omitempty" url:"msg,omitempty"`
}

// Task describes a single recent task entry returned by GET /cluster/tasks,
// the cluster-wide aggregation of every node's task list.
type Task struct {
	// UPID is the unique process/task identifier.
	UPID string `json:"upid,omitempty" url:"upid,omitempty"`
	// Node is the node the task ran/runs on.
	Node string `json:"node,omitempty" url:"node,omitempty"`
	// PID is the worker process ID.
	PID int `json:"pid,omitempty" url:"pid,omitempty"`
	// PStart is the worker process start time (used with PID to detect
	// PID reuse), not a wall-clock time.
	PStart int `json:"pstart,omitempty" url:"pstart,omitempty"`
	// StartTime is the UNIX timestamp the task started at.
	StartTime int64 `json:"starttime,omitempty" url:"starttime,omitempty"`
	// Type is the task kind, e.g. "vzdump", "qmstart", "vncshell".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// ID is the task's subject id, meaning depends on Type (e.g. a VMID
	// for guest tasks).
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// User is the user that started the task, e.g. "root@pam". If the
	// task was started via an API token, this is the bare username and
	// TokenID holds the token id separately.
	User string `json:"user,omitempty" url:"user,omitempty"`
	// TokenID is the API token id (without the user prefix) if the task
	// was started via an API token.
	TokenID string `json:"tokenid,omitempty" url:"tokenid,omitempty"`
	// EndTime is the UNIX timestamp the task finished at. Zero while the
	// task is still running.
	EndTime int64 `json:"endtime,omitempty" url:"endtime,omitempty"`
	// Status is "OK" on success, or an error message, once the task has
	// finished. Empty while the task is still running.
	Status string `json:"status,omitempty" url:"status,omitempty"`
}

// Resource describes a single entry returned by GET /cluster/resources.
// The field set varies by Type: only the fields relevant to that type are
// populated.
type Resource struct {
	// ID is the full resource path, e.g. "node/pve1", "storage/local",
	// "qemu/100".
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Type is the resource kind: "node", "storage", "qemu", "lxc", "sdn"
	// or "pool".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Node is the node the resource belongs to or runs on.
	Node string `json:"node,omitempty" url:"node,omitempty"`
	// Storage is the storage identifier (storage entries only).
	Storage string `json:"storage,omitempty" url:"storage,omitempty"`
	// SDN is the SDN object identifier (sdn entries only).
	SDN string `json:"sdn,omitempty" url:"sdn,omitempty"`
	// Pool is the pool the guest belongs to (vm entries only).
	Pool string `json:"pool,omitempty" url:"pool,omitempty"`
	// VMID is the numeric guest ID (vm entries only).
	VMID int `json:"vmid,omitempty" url:"vmid,omitempty"`
	// Name is the guest, storage, or node name.
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Status is the resource status, e.g. "running", "available", "online".
	Status string `json:"status,omitempty" url:"status,omitempty"`
	// Template is 1 if the guest is a template (vm entries only).
	Template int `json:"template,omitempty" url:"template,omitempty"`
	// Tags is the guest's tag list (vm entries only).
	Tags types.Tags `json:"tags,omitempty" url:"tags,omitempty"`
	// Uptime is the uptime in seconds.
	Uptime int `json:"uptime,omitempty" url:"uptime,omitempty"`
	// Level is the node's cluster support level (node entries only).
	Level string `json:"level,omitempty" url:"level,omitempty"`
	// IP is the node's cluster communication IP address (node entries only).
	IP string `json:"ip,omitempty" url:"ip,omitempty"`
	// PluginType is the storage plugin type (storage entries only).
	PluginType string `json:"plugintype,omitempty" url:"plugintype,omitempty"`
	// Shared is 1 if the storage is shared (storage entries only).
	Shared int `json:"shared,omitempty" url:"shared,omitempty"`
	// StorageContent is the comma-separated content types of the storage
	// (storage entries only).
	StorageContent string `json:"content,omitempty" url:"content,omitempty"`
	// MaxCPU is the number of configured CPUs (vm/node entries).
	MaxCPU int `json:"maxcpu,omitempty" url:"maxcpu,omitempty"`
	// CPU is the CPU usage fraction (vm/node entries).
	CPU float64 `json:"cpu,omitempty" url:"cpu,omitempty"`
	// MaxMem is the maximum memory in bytes (vm/node entries).
	MaxMem int64 `json:"maxmem,omitempty" url:"maxmem,omitempty"`
	// Mem is the used memory in bytes (vm/node entries).
	Mem int64 `json:"mem,omitempty" url:"mem,omitempty"`
	// MaxDisk is the maximum/total disk space in bytes
	// (vm/node/storage entries).
	MaxDisk int64 `json:"maxdisk,omitempty" url:"maxdisk,omitempty"`
	// Disk is the used disk space in bytes (vm/node/storage entries).
	Disk int64 `json:"disk,omitempty" url:"disk,omitempty"`
	// DiskRead is the total bytes read from disk (vm entries only).
	DiskRead int64 `json:"diskread,omitempty" url:"diskread,omitempty"`
	// DiskWrite is the total bytes written to disk (vm entries only).
	DiskWrite int64 `json:"diskwrite,omitempty" url:"diskwrite,omitempty"`
	// NetIn is the network traffic in bytes received (vm entries only).
	NetIn int64 `json:"netin,omitempty" url:"netin,omitempty"`
	// NetOut is the network traffic in bytes sent (vm entries only).
	NetOut int64 `json:"netout,omitempty" url:"netout,omitempty"`
	// HAState is the high-availability state (vm entries only).
	HAState string `json:"hastate,omitempty" url:"hastate,omitempty"`
	// Lock is the guest lock state (vm entries only).
	Lock string `json:"lock,omitempty" url:"lock,omitempty"`
}

// Options describes the cluster-wide configuration exposed by
// GET/PUT /cluster/options. Every field is optional: Proxmox only includes
// a key in the GET response once it has been explicitly set, and PUT only
// changes the fields that are non-zero (via the `url` tag) — to explicitly
// reset a field to its default, list it in Delete instead of trying to
// send a zero value.
//
// Several fields (Bwlimit, Migration, HA, CRS, Notify, NextID, TagStyle,
// U2F, Webauthn) are Proxmox "property strings" (e.g.
// "type=secure,network=10.0.0.0/24") rather than nested JSON objects, so
// they are kept as opaque strings, matching how similar property strings
// (e.g. storage.Storage.PruneBackups) are handled elsewhere.
type Options struct {
	// Description is a cluster-wide description/comment (Datacenter
	// "Notes" field in the UI).
	Description string `json:"description,omitempty" url:"description,omitempty"`
	// EmailFrom is the sender address used for outbound notifications.
	EmailFrom string `json:"email_from,omitempty" url:"email_from,omitempty"`
	// Keyboard is the default keyboard layout for VNC/console access.
	Keyboard string `json:"keyboard,omitempty" url:"keyboard,omitempty"`
	// Language is the default UI language.
	Language string `json:"language,omitempty" url:"language,omitempty"`
	// Console is the default console viewer: "applet", "vv", "html5" or
	// "xtermjs".
	Console string `json:"console,omitempty" url:"console,omitempty"`
	// HTTPProxy is the proxy used for outbound HTTP requests, e.g. when
	// downloading appliance images.
	HTTPProxy string `json:"http_proxy,omitempty" url:"http_proxy,omitempty"`
	// MacPrefix is the OUI prefix used when generating guest MAC
	// addresses.
	MacPrefix string `json:"mac_prefix,omitempty" url:"mac_prefix,omitempty"`
	// MaxWorkers is the maximal number of worker processes per node.
	MaxWorkers int `json:"max_workers,omitempty" url:"max_workers,omitempty"`
	// Bwlimit is the property-string of default I/O bandwidth limits,
	// e.g. "clone=10240,default=0".
	Bwlimit string `json:"bwlimit,omitempty" url:"bwlimit,omitempty"`
	// Migration is the property-string controlling live migration, e.g.
	// "type=secure,network=10.0.0.0/24".
	Migration string `json:"migration,omitempty" url:"migration,omitempty"`
	// HA is the property-string configuring cluster-wide HA behavior,
	// e.g. "shutdown_policy=freeze".
	HA string `json:"ha,omitempty" url:"ha,omitempty"`
	// CRS is the property-string configuring the cluster resource
	// scheduler.
	CRS string `json:"crs,omitempty" url:"crs,omitempty"`
	// Notify is the property-string of default notification targets.
	Notify string `json:"notify,omitempty" url:"notify,omitempty"`
	// NextID is the property-string constraining VMID auto-allocation,
	// e.g. "lower=100,upper=999999999".
	NextID string `json:"next-id,omitempty" url:"next-id,omitempty"`
	// RegisteredTags is the list of tags that are managed/registered
	// cluster-wide.
	RegisteredTags types.Tags `json:"registered-tags,omitempty" url:"registered-tags,omitempty"`
	// TagStyle is the property-string configuring tag appearance and
	// ordering in the UI.
	TagStyle string `json:"tag-style,omitempty" url:"tag-style,omitempty"`
	// U2F is the property-string of U2F authentication settings
	// (superseded by Webauthn).
	U2F string `json:"u2f,omitempty" url:"u2f,omitempty"`
	// Webauthn is the property-string of WebAuthn authentication
	// settings.
	Webauthn string `json:"webauthn,omitempty" url:"webauthn,omitempty"`

	// Delete lists properties to reset to their default value. Write-only:
	// Proxmox never returns it, so Get leaves it empty.
	Delete []string `json:"-" url:"delete"`
}

// encode converts the options to form parameters for PUT /cluster/options.
func (o *Options) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("cluster: options are required")
	}

	return params.Encode(o)
}
