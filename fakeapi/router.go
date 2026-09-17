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
	"strconv"
	"strings"
)

// newRouter builds the fake cluster's HTTP frontend. Routes that name a
// single HTTP method are registered with that prefix (Go 1.22+
// ServeMux); routes serving more than one method at the same path (guest
// config, storage content) are registered bare and dispatch on r.Method
// themselves. Everything not registered here 404s via ServeMux's default
// behavior — see docs/fakeapi.md §7: unimplemented surface fails loudly
// instead of silently returning zero values.
func newRouter(state *clusterState) http.Handler {
	mux := http.NewServeMux()

	reg := func(pattern string, h http.HandlerFunc) {
		mux.HandleFunc(pattern, requireAuth(state, h))
	}

	reg("POST /access/ticket", handleTicket(state))

	reg("GET /cluster/status", handleClusterStatus(state))
	reg("GET /cluster/resources", handleClusterResources(state))

	reg("/cluster/ha/groups", handleHAGroupsCollection(state))
	reg("/cluster/ha/groups/{group}", handleHAGroupItem(state))

	reg("GET /nodes", handleNodesList(state))
	reg("GET /nodes/{node}/status", withNode(state, handleNodeStatus))

	reg("/nodes/{node}/qemu/{vmid}/config", withNode(state, handleQemuConfig))
	reg("GET /nodes/{node}/qemu/{vmid}/status/current", withNode(state, handleQemuStatusCurrent))
	reg("POST /nodes/{node}/qemu/{vmid}/status/{action}", withNode(state, handleQemuAction))

	reg("/nodes/{node}/lxc/{vmid}/config", withNode(state, handleLXCConfig))
	reg("GET /nodes/{node}/lxc/{vmid}/status/current", withNode(state, handleLXCStatusCurrent))
	reg("POST /nodes/{node}/lxc/{vmid}/status/{action}", withNode(state, handleLXCAction))

	reg("GET /nodes/{node}/storage", withNode(state, handleStorageList))
	reg("GET /nodes/{node}/storage/{storage}/status", withNode(state, handleStorageStatus))
	reg("/nodes/{node}/storage/{storage}/content", withNode(state, handleStorageContentCollection))
	reg("/nodes/{node}/storage/{storage}/content/{volume...}", withNode(state, handleStorageContentItem))

	reg("GET /nodes/{node}/tasks/{upid}/status", withNode(state, handleTaskStatus))
	reg("DELETE /nodes/{node}/tasks/{upid}", withNode(state, handleTaskStop))

	return mux
}

// requireAuth accepts any request carrying a non-empty PVEAPIToken
// Authorization header (the fake does not check the token/secret
// themselves — Cluster.Client always sends a fixed fake one) or a
// PVEAuthCookie matching a ticket previously issued by POST
// /access/ticket, and rejects everything else with a 401 — so
// password/ticket auth is also testable, per docs/fakeapi.md §7's
// /access/ticket row.
func requireAuth(state *clusterState, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "PVEAPIToken") {
			next(w, r)
			return
		}

		if cookie, err := r.Cookie("PVEAuthCookie"); err == nil {
			state.mu.Lock()
			_, ok := state.tickets[cookie.Value]
			state.mu.Unlock()

			if ok {
				next(w, r)
				return
			}
		}

		writeError(w, http.StatusUnauthorized, "authentication failure", nil)
	}
}

// withNode resolves the {node} path segment, 404s for a name never
// passed to WithNodes, and — for a node currently failing per
// Cluster.FailNode — short-circuits per its FailureMode before next ever
// runs. Every /nodes/{node}/... handler in this package is wrapped with
// it.
func withNode(state *clusterState, next func(w http.ResponseWriter, r *http.Request, n *nodeState)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("node")

		state.mu.Lock()
		n, ok := state.nodes[name]
		var failing bool
		var mode FailureMode
		if ok {
			failing, mode = n.failing, n.failMode
		}
		state.mu.Unlock()

		if !ok {
			notFound(w, "node")
			return
		}

		if failing {
			if mode == FailureConnRefused {
				hijackConnRefused(w)
				return
			}
			unreachable(w)
			return
		}

		next(w, r, n)
	}
}

// pathVMID parses the {vmid} path segment, writing a 400 and reporting
// failure if it isn't a valid integer.
func pathVMID(w http.ResponseWriter, r *http.Request) (int, bool) {
	vmid, err := strconv.Atoi(r.PathValue("vmid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "parameter verification failed", map[string]string{"vmid": "not a number"})
		return 0, false
	}

	return vmid, true
}
