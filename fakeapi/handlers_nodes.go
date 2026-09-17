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

	"github.com/sergelogvinov/go-proxmox-rest/nodes"
)

// wireNodeSummary is the shape real Proxmox returns per entry in GET
// /nodes. Nothing in this module currently wraps that endpoint (the
// nodes package is scoped to a single node — see nodes.New), so this is
// a plain locally-defined struct rather than a reused exported type.
type wireNodeSummary struct {
	Node   string  `json:"node"`
	Status string  `json:"status"`
	Type   string  `json:"type"`
	CPU    float64 `json:"cpu,omitempty"`
	MaxCPU int     `json:"maxcpu,omitempty"`
	Mem    int64   `json:"mem,omitempty"`
	MaxMem int64   `json:"maxmem,omitempty"`
	Uptime int64   `json:"uptime,omitempty"`
	Level  string  `json:"level,omitempty"`
}

// handleNodesList backs GET /nodes.
func handleNodesList(state *clusterState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()

		list := make([]wireNodeSummary, 0, len(state.nodes))
		for _, name := range sortedNodeNames(state) {
			n := state.nodes[name]

			status := "offline"
			if n.online {
				status = "online"
			}

			cpuUsed, memUsed := nodeUsage(n)

			list = append(list, wireNodeSummary{
				Node: name, Status: status, Type: "node",
				CPU: cpuUsed, MaxCPU: n.cores,
				Mem: memUsed, MaxMem: n.memTotal,
			})
		}

		writeData(w, list)
	}
}

// handleNodeStatus backs GET /nodes/{node}/status.
func handleNodeStatus(w http.ResponseWriter, r *http.Request, n *nodeState) {
	n.cs.mu.Lock()
	defer n.cs.mu.Unlock()

	cpuUsed, memUsed := nodeUsage(n)

	status := &nodes.Status{
		PVEVersion: "pve-manager/9.0.0/fakeapi",
		Uptime:     3600,
		CPU:        cpuUsed,
		CPUInfo: &nodes.CPUInfo{
			Cores: n.cores, CPUs: n.cores, Sockets: 1,
			Model: "FakeAPI vCPU",
		},
		Memory: &nodes.Memory{
			Total:     n.memTotal,
			Used:      memUsed,
			Free:      n.memTotal - memUsed,
			Available: n.memTotal - memUsed,
		},
	}

	writeData(w, status)
}

// nodeUsage estimates a node's CPU fraction and used memory in bytes from
// a small fixed overhead plus its running guests' configured memory —
// enough for callers exercising "is this node busy" logic without
// tracking real resource accounting. Must be called with state.mu held.
func nodeUsage(n *nodeState) (cpu float64, memUsed int64) {
	const baseMemory = 512 << 20 // fakeapi's own "hypervisor overhead"

	memUsed = baseMemory
	running := 0

	for _, vm := range n.guests {
		if vm.status == "running" {
			running++
			memUsed += memoryMBToBytes(parseIntDefault(vm.cfg["memory"], 512))
		}
	}
	for _, ct := range n.containers {
		if ct.status == "running" {
			running++
			memUsed += memoryMBToBytes(parseIntDefault(ct.cfg["memory"], 512))
		}
	}

	cpu = 0.02 + float64(running)*0.05
	if cpu > 1 {
		cpu = 1
	}
	if memUsed > n.memTotal {
		memUsed = n.memTotal
	}

	return cpu, memUsed
}
