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

package ceph

import "github.com/sergelogvinov/go-proxmox-rest/types"

// Status, Health, MonMap, OSDMap, PGState, PGMap and MgrMap describe the
// Ceph status returned by GET /nodes/{node}/ceph/status: Proxmox's own
// documentation states the response is identical to
// GET /cluster/ceph/status, so these live in the shared types package
// (see its doc comment and docs/design.md §2) and are re-exported here
// as aliases so call sites keep reading as ceph.Status, ceph.Health, ... .
type (
	Status  = types.Status
	Health  = types.Health
	MonMap  = types.MonMap
	OSDMap  = types.OSDMap
	PGState = types.PGState
	PGMap   = types.PGMap
	MgrMap  = types.MgrMap
)

// Service names the ceph service scope for Client.Start/Stop/Restart.
// Proxmox accepts a bare kind (affecting every instance of that kind) or
// "kind.instance" (e.g. "mon.pve1", "osd.3") to target one instance; the
// constants below cover only the bare-kind form — build an
// instance-scoped value with a plain string conversion, e.g.
// Service("mon.pve1").
type Service string

const (
	// ServiceAll targets every Ceph service (ceph.target), Proxmox's
	// own default when Start/Stop/Restart's service is "".
	ServiceAll Service = "ceph"
	ServiceMon Service = "mon"
	ServiceMDS Service = "mds"
	ServiceOSD Service = "osd"
	ServiceMgr Service = "mgr"
)

// Release describes one known Ceph release and its installability on
// the queried node, as returned by GET /nodes/{node}/ceph/releases.
type Release struct {
	// Release is the Ceph release code name, e.g. "squid".
	Release string `json:"release,omitempty" url:"release,omitempty"`
	// Version is the Ceph release's major version, e.g. "19.2".
	Version string `json:"version,omitempty" url:"version,omitempty"`
	// Available is true if this release has packages for the node's
	// architecture and current Proxmox VE release.
	Available bool `json:"available,omitempty" url:"available,omitempty"`
	// IsDefault is true if this is the release recommended for new
	// installations.
	IsDefault bool `json:"is-default,omitempty" url:"is-default,omitempty"`
	// Unsupported is true if this release is not yet supported for
	// production use.
	Unsupported bool `json:"unsupported,omitempty" url:"unsupported,omitempty"`
}

// LogEntry is a single Ceph log line, as returned by
// GET /nodes/{node}/ceph/log.
type LogEntry struct {
	// N is the line number.
	N int64 `json:"n,omitempty" url:"n,omitempty"`
	// T is the line text.
	T string `json:"t,omitempty" url:"t,omitempty"`
}

// LogOptions filters the log lines returned by Client.Log. A nil
// *LogOptions (or the zero value) requests Proxmox's default window (the
// first ~50 lines).
type LogOptions struct {
	// Start is the first line number to return.
	Start int `url:"start,omitempty"`
	// Limit caps the number of lines returned.
	Limit int `url:"limit,omitempty"`
}
