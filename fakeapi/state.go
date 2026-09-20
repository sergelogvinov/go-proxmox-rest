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

package fakeapi

import (
	"sync"
	"time"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/storage"
)

// defaultNodeCores and defaultNodeMemory are the node hardware totals
// reported by a node seeded via WithNodes before Node.SetResources
// customizes them.
const (
	defaultNodeCores  = 4
	defaultNodeMemory = 8 << 30 // 8 GiB
)

// clusterState is the single mutex-guarded model backing a Cluster. Every
// handler reads/writes through it while holding mu, so the fake is safe
// under concurrent requests and t.Parallel().
type clusterState struct {
	mu sync.Mutex

	name     string
	nodes    map[string]*nodeState
	haGroups map[string]*haGroupState
	tasks    map[string]*taskState
	tickets  map[string]string // PVEAuthCookie value -> username
	taskSeq  int64
	manual   bool
}

// nodeState is one cluster member: its online/failure state, seeded
// storages, and seeded guests.
type nodeState struct {
	// cs is the owning clusterState, whose mutex guards every field
	// below (including nested storageState/vmState/containerState
	// values) — a back-reference so node-scoped handlers, which receive
	// only a *nodeState from withNode, can still lock/unlock it.
	cs *clusterState

	name     string
	online   bool
	failing  bool
	failMode FailureMode

	// cores/memTotal are the node's reported hardware totals (GET
	// /nodes/{node}/status and the "node" entries in GET
	// /cluster/resources). Not part of docs/fakeapi.md's v1 sketch, which
	// only seeds node names — added so a caller can model realistically
	// sized nodes; see Node.SetResources.
	cores    int
	memTotal int64

	storages   map[string]*storageState
	guests     map[int]*vmState        // QEMU, keyed by VMID
	containers map[int]*containerState // LXC, keyed by VMID
}

// haGroupState is one HA group, backing GET /cluster/ha/groups and
// GET/PUT/DELETE /cluster/ha/groups/{group}. A typed struct rather than
// the flat wire-parameter map vmState/containerState use for guest
// config: HA groups have a small, fixed set of fields and no unexported
// encode/decode to route around (see vmState's doc comment for why guest
// config takes the other approach), so it follows storageState's shape
// instead.
type haGroupState struct {
	id         string
	nodes      string
	comment    string
	nofailback bool
	restricted bool
}

// storageState is one storage as seen from a single node.
type storageState struct {
	id      string
	typ     string
	shared  bool
	content []storage.Volume
	total   int64
	used    int64
	avail   int64
}

// vmState is one QEMU guest.
//
// docs/fakeapi.md §6 sketches cfg as a live qemu.Config so "a guest's disks
// are not a separate model." In practice qemu.Config's own encode/decode
// (the numerically-suffixed netN/scsiN/... families, property-string
// scalars) lives in unexported functions in nodes/qemu/config.go, so this
// package cannot reuse them directly. Storing the same flat
// wire-parameter map[string]string the client itself sends
// (params.Encode's output, plus one "prefix+index" entry per indexed
// family — see flattenQemuConfig) achieves the same goal by construction:
// GET just re-serves the map as a JSON object of strings, which
// decodeConfig/params.Decode parse exactly as they would a real
// Proxmox response (params.Decode already tolerates numbers/bools
// arriving as JSON strings). No config grammar is duplicated here.
type vmState struct {
	vmid      int
	node      string
	cfg       map[string]string
	status    qemu.VMStatus
	startedAt time.Time
}

// containerState is one LXC container, the same shape as vmState. LXC
// coverage is called out as a v2/future addition in docs/fakeapi.md §13;
// it's included here from the start since the fakeapi.md worked example
// this package supports (examples/fakeapi) seeds both guest types.
type containerState struct {
	vmid      int
	node      string
	cfg       map[string]string
	status    lxc.State
	startedAt time.Time
}

// taskState is one async task/UPID (see tasks.go).
type taskState struct {
	upid      string
	node      string
	typ       string
	id        string
	user      string
	running   bool
	ok        bool
	errMsg    string
	startTime int64
	endTime   int64

	// apply is the staged mutation under WithManualTasks; nil once the
	// task has completed (instant mode never needs it, so it's set only
	// long enough to let TaskController.Complete run it).
	apply func() error
}

// newClusterState builds an empty state; ClusterOptions populate it.
func newClusterState() *clusterState {
	return &clusterState{
		name:     "fakeapi",
		nodes:    map[string]*nodeState{},
		haGroups: map[string]*haGroupState{},
		tasks:    map[string]*taskState{},
	}
}

func newNodeState(cs *clusterState, name string) *nodeState {
	return &nodeState{
		cs:         cs,
		name:       name,
		online:     true,
		cores:      defaultNodeCores,
		memTotal:   defaultNodeMemory,
		storages:   map[string]*storageState{},
		guests:     map[int]*vmState{},
		containers: map[int]*containerState{},
	}
}
