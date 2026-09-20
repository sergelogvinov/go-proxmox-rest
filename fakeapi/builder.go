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
	"time"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/storage"
)

// Node is the seeding/mutation handle for one cluster member, returned by
// Cluster.Node.
type Node struct {
	state *clusterState
	node  *nodeState
}

// SetResources sets the node's reported hardware totals — cores and
// memory in bytes — surfaced by GET /nodes/{node}/status and the node's
// entry in GET /cluster/resources. Not part of docs/fakeapi.md's v1
// sketch (which seeds only node names via WithNodes); added so a test or
// example can model a realistically sized node. Returns the receiver for
// chaining.
func (n *Node) SetResources(cores int, memoryBytes int64) *Node {
	n.state.mu.Lock()
	defer n.state.mu.Unlock()

	n.node.cores = cores
	n.node.memTotal = memoryBytes

	return n
}

// Storage is the seeding/mutation handle for one storage on a node,
// returned by Node.AddStorage.
type Storage struct {
	state *clusterState
	st    *storageState
}

// StorageOption configures a Storage at seed time.
type StorageOption func(*storageState)

// WithVolume seeds an existing volume in the storage's content list —
// useful for "disk already attached" or "orphaned volume" starting
// states.
func WithVolume(v storage.Volume) StorageOption {
	return func(s *storageState) { s.content = append(s.content, v) }
}

// WithCapacity sets the storage's total/used/avail bytes reported by
// GET /nodes/{node}/storage (List) and .../storage/{storage}/status
// (Status).
func WithCapacity(total, used, avail int64) StorageOption {
	return func(s *storageState) {
		s.total = total
		s.used = used
		s.avail = avail
	}
}

// WithShared marks the storage as shared across nodes — a config attribute
// independent of how many nodes it was actually seeded on (e.g. a CIFS/NFS
// share is normally configured shared regardless of how many cluster members
// currently mount it). Surfaced as cluster.Resource.Shared (GET
// /cluster/resources) and storage.Storage.Shared (GET /storage/{storage}).
func WithShared() StorageOption {
	return func(s *storageState) { s.shared = true }
}

// AddStorage seeds a storage visible on this node (dir/lvm/zfs/... —
// storageType is opaque to the fake beyond being echoed back).
func (n *Node) AddStorage(id, storageType string, opts ...StorageOption) *Storage {
	st := &storageState{id: id, typ: storageType}
	for _, opt := range opts {
		opt(st)
	}

	n.state.mu.Lock()
	n.node.storages[id] = st
	n.state.mu.Unlock()

	return &Storage{state: n.state, st: st}
}

// VM is the seeding/mutation handle for one QEMU guest, returned by
// Node.AddVM.
type VM struct {
	state *clusterState
	vm    *vmState
}

// VMOption configures a VM at seed time.
type VMOption func(*vmState)

// WithStatus seeds the guest's initial run state (default
// qemu.VMStatusStopped). Seeding as qemu.VMStatusRunning also seeds a
// startedAt of now, so an immediate Status(ctx) call reports a non-zero
// uptime.
func WithStatus(s qemu.VMStatus) VMOption {
	return func(v *vmState) {
		v.status = s
		if s == qemu.VMStatusRunning {
			v.startedAt = time.Now()
		}
	}
}

// AddVM seeds a QEMU guest on this node with the given config. cfg is a
// *qemu.Config exactly as Config/UpdateConfig/UpdateConfigAsync already
// use — no separate "fake VM" DTO. See vmState's doc comment (state.go)
// for how cfg is actually stored internally.
func (n *Node) AddVM(vmid int, cfg *qemu.Config, opts ...VMOption) *VM {
	vm := &vmState{
		vmid:   vmid,
		node:   n.node.name,
		cfg:    flattenQemuConfig(cfg),
		status: qemu.VMStatusStopped,
	}
	for _, opt := range opts {
		opt(vm)
	}

	n.state.mu.Lock()
	n.node.guests[vmid] = vm
	n.state.mu.Unlock()

	return &VM{state: n.state, vm: vm}
}

// Container is the seeding/mutation handle for one LXC container,
// returned by Node.AddContainer.
type Container struct {
	state *clusterState
	ct    *containerState
}

// ContainerOption configures a Container at seed time.
type ContainerOption func(*containerState)

// WithContainerStatus seeds the container's initial run state (default
// lxc.StateStopped). Seeding as lxc.StateRunning also seeds a startedAt
// of now, so an immediate Status(ctx) call reports a non-zero uptime.
func WithContainerStatus(s lxc.State) ContainerOption {
	return func(c *containerState) {
		c.status = s
		if s == lxc.StateRunning {
			c.startedAt = time.Now()
		}
	}
}

// AddContainer seeds an LXC container on this node with the given
// config. cfg is a *lxc.Config exactly as Config/UpdateConfig already
// use. LXC support is not part of docs/fakeapi.md's v1 scope (§13 lists
// it as a natural v2 addition); it's included here, following the same
// shape as AddVM, since the fakeapi.md worked example this package
// supports (examples/fakeapi) seeds both guest types.
func (n *Node) AddContainer(vmid int, cfg *lxc.Config, opts ...ContainerOption) *Container {
	ct := &containerState{
		vmid:   vmid,
		node:   n.node.name,
		cfg:    flattenLXCConfig(cfg),
		status: lxc.StateStopped,
	}
	for _, opt := range opts {
		opt(ct)
	}

	n.state.mu.Lock()
	n.node.containers[vmid] = ct
	n.state.mu.Unlock()

	return &Container{state: n.state, ct: ct}
}
