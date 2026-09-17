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

	"github.com/sergelogvinov/go-proxmox-rest/cluster/ha"
)

// haGroupToWire converts g to the shape GET /cluster/ha/groups(/{group})
// serves.
func haGroupToWire(g *haGroupState) ha.Group {
	return ha.Group{
		Group:      g.id,
		Type:       "group",
		Nodes:      g.nodes,
		Comment:    g.comment,
		Nofailback: boolToInt(g.nofailback),
		Restricted: boolToInt(g.restricted),
	}
}

// handleHAGroupsCollection backs GET/POST /cluster/ha/groups — List and
// Create.
func handleHAGroupsCollection(state *clusterState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			state.mu.Lock()
			ids := make([]string, 0, len(state.haGroups))
			for id := range state.haGroups {
				ids = append(ids, id)
			}
			sort.Strings(ids)

			groups := make([]ha.Group, 0, len(ids))
			for _, id := range ids {
				groups = append(groups, haGroupToWire(state.haGroups[id]))
			}
			state.mu.Unlock()

			writeData(w, groups)

		case http.MethodPost:
			query := r.URL.Query()
			id := query.Get("group")
			if id == "" {
				writeError(w, http.StatusBadRequest, "parameter verification failed",
					map[string]string{"group": "group id is required"})
				return
			}

			g := &haGroupState{
				id:         id,
				nodes:      query.Get("nodes"),
				comment:    query.Get("comment"),
				nofailback: query.Get("nofailback") == "1",
				restricted: query.Get("restricted") == "1",
			}

			state.mu.Lock()
			state.haGroups[id] = g
			state.mu.Unlock()

			// Real Proxmox returns null data for this endpoint, same as
			// applying a config update elsewhere in this fake.
			writeData(w, nil)

		default:
			methodNotAllowed(w)
		}
	}
}

// handleHAGroupItem backs GET/PUT/DELETE /cluster/ha/groups/{group} — Get,
// Update, and Delete.
func handleHAGroupItem(state *clusterState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("group")

		state.mu.Lock()
		g, ok := state.haGroups[id]
		state.mu.Unlock()
		if !ok {
			notFound(w, "group")
			return
		}

		switch r.Method {
		case http.MethodGet:
			writeData(w, haGroupToWire(g))

		case http.MethodPut:
			query := r.URL.Query()

			state.mu.Lock()
			applyHAGroupUpdate(g, query)
			state.mu.Unlock()

			writeData(w, nil)

		case http.MethodDelete:
			state.mu.Lock()
			delete(state.haGroups, id)
			state.mu.Unlock()

			writeData(w, nil)

		default:
			methodNotAllowed(w)
		}
	}
}

// applyHAGroupUpdate merges query's parameters into g, honoring a
// comma-joined "delete" parameter the same way applyConfigUpdate does for
// guest config: reset those properties to their zero value instead of
// setting them. Called with the cluster state mutex held.
func applyHAGroupUpdate(g *haGroupState, query map[string][]string) {
	if del := query["delete"]; len(del) > 0 {
		for _, list := range del {
			for k := range strings.SplitSeq(list, ",") {
				switch strings.TrimSpace(k) {
				case "comment":
					g.comment = ""
				case "nofailback":
					g.nofailback = false
				case "restricted":
					g.restricted = false
				}
			}
		}
	}

	if v, ok := query["nodes"]; ok && len(v) > 0 {
		g.nodes = v[0]
	}
	if v, ok := query["comment"]; ok && len(v) > 0 {
		g.comment = v[0]
	}
	if v, ok := query["nofailback"]; ok && len(v) > 0 {
		g.nofailback = v[0] == "1"
	}
	if v, ok := query["restricted"]; ok && len(v) > 0 {
		g.restricted = v[0] == "1"
	}
}
