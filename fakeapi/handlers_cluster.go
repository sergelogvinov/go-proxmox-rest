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
	"net/http"
	"sort"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/cluster"
)

// wireStatusEntry mirrors cluster's own unexported statusEntry: the wire
// format for GET /cluster/status is a flat list mixing one "cluster"-type
// object with the per-node objects. Since encoding/json flattens an
// anonymous embedded field automatically (unlike internal/params.Decode,
// which needed a matching fix to read this same shape back — see
// decode.go's ",inline" handling), building this exact shape here and
// json-encoding it round-trips correctly through cluster.Client.Status.
type wireStatusEntry struct {
	cluster.NodeStatus

	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Type    string `json:"type,omitempty"`
	Quorate int    `json:"quorate,omitempty"`
	Version int    `json:"version,omitempty"`
}

// handleClusterStatus backs GET /cluster/status. A single-node cluster
// reports no "cluster"-type entry at all, matching real Proxmox's
// behavior on a standalone node (see cluster.Client.Status's doc
// comment); quorum for a multi-node cluster is the online-node majority.
func handleClusterStatus(state *clusterState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()

		names := sortedNodeNames(state)

		var entries []wireStatusEntry
		if len(names) > 1 {
			online := 0
			for _, n := range state.nodes {
				if n.online {
					online++
				}
			}

			quorate := 0
			if online*2 > len(names) {
				quorate = 1
			}

			entries = append(entries, wireStatusEntry{
				ID: state.name, Name: state.name, Type: "cluster",
				Quorate: quorate, Version: 1,
			})
		}

		for i, name := range names {
			n := state.nodes[name]
			// ID/Name/Type are set on the outer struct, not the embedded
			// NodeStatus: encoding/json's field-shadowing rule means an
			// outer field always wins over an embedded field with the
			// same JSON name, at every depth, regardless of the outer
			// field's value — so a NodeStatus.Name/.ID/.Type set here
			// instead would silently never reach the wire. NodeID/
			// Online/Local don't collide with an outer field, so they're
			// set on NodeStatus normally.
			entries = append(entries, wireStatusEntry{
				ID:   "node/" + name,
				Name: name,
				Type: "node",
				NodeStatus: cluster.NodeStatus{
					NodeID: i + 1,
					Online: boolToInt(n.online),
					Local:  boolToInt(i == 0),
				},
			})
		}

		writeData(w, entries)
	}
}

// handleClusterResources backs GET /cluster/resources, synthesizing
// node/storage/qemu/lxc entries from state. The "type" query parameter
// filters to a single cluster.ResourceType, matching
// cluster.Client.Resources().List.
func handleClusterResources(state *clusterState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := cluster.ResourceType(r.URL.Query().Get("type"))

		state.mu.Lock()
		defer state.mu.Unlock()

		var resources []cluster.Resource

		for _, name := range sortedNodeNames(state) {
			n := state.nodes[name]

			if filter == "" || filter == cluster.ResourceTypeNode {
				resources = append(resources, nodeResource(n))
			}

			if filter == "" || filter == cluster.ResourceTypeStorage {
				for _, id := range sortedStorageIDs(n) {
					resources = append(resources, storageResource(name, n.storages[id]))
				}
			}

			if filter == "" || filter == cluster.ResourceTypeVM {
				for _, vmid := range sortedGuestIDs(n) {
					resources = append(resources, qemuResource(name, n.guests[vmid]))
				}
				for _, vmid := range sortedContainerIDs(n) {
					resources = append(resources, lxcResource(name, n.containers[vmid]))
				}
			}
		}

		writeData(w, resources)
	}
}

func nodeResource(n *nodeState) cluster.Resource {
	status := "offline"
	if n.online {
		status = "online"
	}

	return cluster.Resource{
		ID:     "node/" + n.name,
		Type:   "node",
		Node:   n.name,
		Name:   n.name,
		Status: status,
		MaxCPU: n.cores,
		MaxMem: n.memTotal,
		Level:  "",
		IP:     "0.0.0.0",
	}
}

func storageResource(node string, st *storageState) cluster.Resource {
	return cluster.Resource{
		ID:         "storage/" + node + "/" + st.id,
		Type:       "storage",
		Node:       node,
		Storage:    st.id,
		Status:     "available",
		PluginType: st.typ,
		Shared:     boolToInt(st.shared),
		MaxDisk:    st.total,
		Disk:       st.used,
	}
}

func qemuResource(node string, vm *vmState) cluster.Resource {
	return cluster.Resource{
		ID:     "qemu/" + strconv.Itoa(vm.vmid),
		Type:   "qemu",
		Node:   node,
		VMID:   vm.vmid,
		Name:   vm.cfg["name"],
		Status: string(vm.status),
		MaxCPU: parseIntDefault(vm.cfg["cores"], 1),
		MaxMem: memoryMBToBytes(parseIntDefault(vm.cfg["memory"], 512)),
		Uptime: int(guestUptime(vm.status == "running", vm.startedAt).Seconds()),
	}
}

func lxcResource(node string, ct *containerState) cluster.Resource {
	return cluster.Resource{
		ID:     "lxc/" + strconv.Itoa(ct.vmid),
		Type:   "lxc",
		Node:   node,
		VMID:   ct.vmid,
		Name:   ct.cfg["hostname"],
		Status: string(ct.status),
		MaxCPU: parseIntDefault(ct.cfg["cores"], 1),
		MaxMem: memoryMBToBytes(parseIntDefault(ct.cfg["memory"], 512)),
		Uptime: int(guestUptime(ct.status == "running", ct.startedAt).Seconds()),
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// sortedNodeNames/sortedStorageIDs/sortedGuestIDs/sortedContainerIDs give
// deterministic iteration order over state's maps, so repeated calls
// against unchanged state return identically ordered lists.
func sortedNodeNames(state *clusterState) []string {
	names := make([]string, 0, len(state.nodes))
	for name := range state.nodes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedStorageIDs(n *nodeState) []string {
	ids := make([]string, 0, len(n.storages))
	for id := range n.storages {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedGuestIDs(n *nodeState) []int {
	ids := make([]int, 0, len(n.guests))
	for id := range n.guests {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func sortedContainerIDs(n *nodeState) []int {
	ids := make([]int, 0, len(n.containers))
	for id := range n.containers {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}
