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
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/sergelogvinov/go-proxmox-rest/pools"
)

// poolView renders a pool, with its guest members when withMembers is set.
// Called with state.mu held.
func poolView(state *clusterState, p *poolState, withMembers bool) pools.Pool {
	view := pools.Pool{PoolID: p.id, Comment: p.comment}
	if !withMembers {
		return view
	}

	view.Members = []pools.PoolMember{}

	for _, vmid := range p.vmids {
		for _, n := range state.nodes {
			if vm := n.guests[vmid]; vm != nil {
				view.Members = append(view.Members, pools.PoolMember{
					ID:     "qemu/" + strconv.Itoa(vmid),
					Type:   "qemu",
					VMID:   vmid,
					Node:   n.name,
					Name:   vm.cfg["name"],
					Status: string(vm.status),
				})
			}
		}
	}

	return view
}

// poolOf returns the pool vmid belongs to, or nil. Called with state.mu
// held.
func poolOf(state *clusterState, vmid int) *poolState {
	for _, p := range state.pools {
		if slices.Contains(p.vmids, vmid) {
			return p
		}
	}

	return nil
}

// handlePoolsCollection backs GET/POST /pools — List and Create. The list
// carries no members, like real Proxmox's.
func handlePoolsCollection(state *clusterState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()

		switch r.Method {
		case http.MethodGet:
			list := make([]pools.Pool, 0, len(state.pools))
			for _, id := range slices.Sorted(maps.Keys(state.pools)) {
				list = append(list, poolView(state, state.pools[id], false))
			}

			writeData(w, list)

		case http.MethodPost:
			q := r.URL.Query()

			id := q.Get("poolid")
			if id == "" {
				writeError(w, http.StatusBadRequest, "parameter verification failed", map[string]string{"poolid": "property is missing and it is not optional"})
				return
			}

			if state.pools[id] != nil {
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("create pool failed: pool '%s' already exists", id), nil)
				return
			}

			state.pools[id] = &poolState{id: id, comment: q.Get("comment")}
			writeData(w, nil)

		default:
			methodNotAllowed(w)
		}
	}
}

// handlePoolsItem backs GET/PUT/DELETE /pools/?poolid=... — Get, Update and
// Delete, which address the pool by query parameter. An update adds the
// guests in "vms" to the pool, or with "delete" removes them; adding a
// guest that belongs to another pool fails unless "allow-move" is set.
func handlePoolsItem(state *clusterState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		id := q.Get("poolid")

		state.mu.Lock()
		defer state.mu.Unlock()

		p := state.pools[id]

		switch r.Method {
		case http.MethodGet:
			if p == nil {
				writeData(w, []pools.Pool{})
				return
			}

			writeData(w, []pools.Pool{poolView(state, p, true)})

		case http.MethodPut:
			if p == nil {
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("pool '%s' does not exist", id), nil)
				return
			}

			if c, ok := q["comment"]; ok && len(c) > 0 {
				p.comment = c[0]
			}

			remove := q.Get("delete") == "1"
			allowMove := q.Get("allow-move") == "1"

			for v := range strings.SplitSeq(q.Get("vms"), ",") {
				if v == "" {
					continue
				}

				vmid, err := strconv.Atoi(v)
				if err != nil {
					writeError(w, http.StatusBadRequest, "parameter verification failed", map[string]string{"vms": "invalid vmid " + v})
					return
				}

				if remove {
					p.vmids = slices.DeleteFunc(p.vmids, func(id int) bool { return id == vmid })
					continue
				}

				if !guestExists(state, vmid) {
					writeError(w, http.StatusInternalServerError, fmt.Sprintf("no such VMID '%d'", vmid), nil)
					return
				}

				if other := poolOf(state, vmid); other != nil && other != p {
					if !allowMove {
						writeError(w, http.StatusInternalServerError, fmt.Sprintf("VM %d belongs already to pool '%s' and allow-move is not set", vmid, other.id), nil)
						return
					}

					other.vmids = slices.DeleteFunc(other.vmids, func(id int) bool { return id == vmid })
				}

				if !slices.Contains(p.vmids, vmid) {
					p.vmids = append(p.vmids, vmid)
				}
			}

			writeData(w, nil)

		case http.MethodDelete:
			if p == nil {
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("pool '%s' does not exist", id), nil)
				return
			}

			if len(p.vmids) > 0 {
				writeError(w, http.StatusInternalServerError, "pool is not empty", nil)
				return
			}

			delete(state.pools, id)
			writeData(w, nil)

		default:
			methodNotAllowed(w)
		}
	}
}
