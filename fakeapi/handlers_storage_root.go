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
	"strings"

	rootstorage "github.com/sergelogvinov/go-proxmox-rest/storage"
)

// handleStorageRootList backs GET /storage, the cluster-wide storage
// configuration list (distinct from GET /nodes/{node}/storage, which is
// scoped to what one node currently sees). Each distinct storage id seeded
// on any node appears once, aggregated across every node that carries it.
func handleStorageRootList(state *clusterState) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()

		ids := sortedClusterStorageIDs(state)

		list := make([]rootstorage.Storage, 0, len(ids))
		for _, id := range ids {
			list = append(list, storageRootView(state, id, false))
		}

		writeData(w, list)
	}
}

// handleStorageRootGet backs GET /storage/{storage}.
func handleStorageRootGet(state *clusterState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("storage")

		state.mu.Lock()
		defer state.mu.Unlock()

		if !clusterHasStorage(state, id) {
			notFound(w, "storage")
			return
		}

		writeData(w, storageRootView(state, id, true))
	}
}

// storageRootView builds the root /storage config view for id, aggregating
// across every node that carries it: Nodes lists them all (comma-separated,
// matching Storage.Nodes' documented grammar), and Type/Shared/capacity come
// from the first such node (sorted) — every node seeding the same storage id
// is expected to agree on its config, mirroring real Proxmox's single shared
// /etc/pve/storage.cfg entry.
func storageRootView(state *clusterState, id string, includeCapacity bool) rootstorage.Storage {
	var nodes []string

	var first *storageState

	for _, name := range sortedNodeNames(state) {
		st := state.nodes[name].storages[id]
		if st == nil {
			continue
		}

		nodes = append(nodes, name)

		if first == nil {
			first = st
		}
	}

	s := rootstorage.Storage{
		Storage: id,
		Nodes:   strings.Join(nodes, ","),
	}

	if first != nil {
		s.Type = first.typ
		s.Shared = boolToInt(first.shared)
		s.Enabled = 1
		s.Active = 1

		if includeCapacity {
			s.TotalSpace = first.total
			s.SpaceUsed = first.used
			s.AvailableSpace = first.avail
		}
	}

	return s
}

// clusterHasStorage reports whether any seeded node carries storage id.
func clusterHasStorage(state *clusterState, id string) bool {
	for _, n := range state.nodes {
		if n.storages[id] != nil {
			return true
		}
	}

	return false
}

// sortedClusterStorageIDs lists every distinct storage id seeded on any
// node, sorted for deterministic output.
func sortedClusterStorageIDs(state *clusterState) []string {
	seen := map[string]bool{}

	for _, n := range state.nodes {
		for id := range n.storages {
			seen[id] = true
		}
	}

	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	return ids
}
