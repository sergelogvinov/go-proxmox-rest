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
	"slices"
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

// haGroupsMigratedMessage is the status-line reason phrase a real Proxmox
// cluster sends back from GET /cluster/ha/groups once its HA groups have
// been migrated to HA rules. See WithHAGroupsMigrated and
// writeStatusLineError.
const haGroupsMigratedMessage = "cannot index groups: ha groups have been migrated to rules"

// handleHAGroupsCollection backs GET/POST /cluster/ha/groups — List and
// Create.
func handleHAGroupsCollection(state *clusterState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			state.mu.Lock()
			if state.haGroupsMigrated {
				state.mu.Unlock()
				writeStatusLineError(w, http.StatusInternalServerError, haGroupsMigratedMessage)

				return
			}

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

// haRuleToWire converts r to the shape GET /cluster/ha/rules(/{rule})
// serves.
func haRuleToWire(r *haRuleState) ha.Rule {
	return ha.Rule{
		Rule:      r.id,
		Type:      r.typ,
		Resources: r.resources,
		Nodes:     r.nodes,
		Affinity:  r.affinity,
		Strict:    boolToInt(r.strict),
		Disable:   boolToInt(r.disable),
		Comment:   r.comment,
	}
}

// handleHARulesCollection backs GET/POST /cluster/ha/rules — List and
// Create.
func handleHARulesCollection(state *clusterState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			query := r.URL.Query()
			ruleType := query.Get("type")
			resource := query.Get("resource")

			state.mu.Lock()
			ids := make([]string, 0, len(state.haRules))
			for id := range state.haRules {
				ids = append(ids, id)
			}
			sort.Strings(ids)

			rules := make([]ha.Rule, 0, len(ids))
			for _, id := range ids {
				rs := state.haRules[id]
				if ruleType != "" && rs.typ != ruleType {
					continue
				}
				if resource != "" && !slices.Contains(rs.resources, resource) {
					continue
				}

				rules = append(rules, haRuleToWire(rs))
			}
			state.mu.Unlock()

			writeData(w, rules)

		case http.MethodPost:
			query := r.URL.Query()
			id := query.Get("rule")
			ruleType := query.Get("type")

			if id == "" || ruleType == "" || query.Get("resources") == "" {
				writeError(w, http.StatusBadRequest, "parameter verification failed",
					map[string]string{"rule": "rule id, type and resources are required"})

				return
			}

			rs := &haRuleState{
				id:        id,
				typ:       ruleType,
				resources: strings.Split(query.Get("resources"), ","),
				nodes:     query.Get("nodes"),
				affinity:  query.Get("affinity"),
				strict:    query.Get("strict") == "1",
				disable:   query.Get("disable") == "1",
				comment:   query.Get("comment"),
			}

			state.mu.Lock()
			state.haRules[id] = rs
			state.mu.Unlock()

			// Real Proxmox returns null data for this endpoint, same as
			// applying a config update elsewhere in this fake.
			writeData(w, nil)

		default:
			methodNotAllowed(w)
		}
	}
}

// handleHARuleItem backs GET/PUT/DELETE /cluster/ha/rules/{rule} — Get,
// Update, and Delete.
func handleHARuleItem(state *clusterState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("rule")

		state.mu.Lock()
		rs, ok := state.haRules[id]
		state.mu.Unlock()
		if !ok {
			notFound(w, "rule")
			return
		}

		switch r.Method {
		case http.MethodGet:
			writeData(w, haRuleToWire(rs))

		case http.MethodPut:
			query := r.URL.Query()

			state.mu.Lock()
			applyHARuleUpdate(rs, query)
			state.mu.Unlock()

			writeData(w, nil)

		case http.MethodDelete:
			state.mu.Lock()
			delete(state.haRules, id)
			state.mu.Unlock()

			writeData(w, nil)

		default:
			methodNotAllowed(w)
		}
	}
}

// applyHARuleUpdate merges query's parameters into rs, honoring a
// comma-joined "delete" parameter the same way applyHAGroupUpdate does.
// Called with the cluster state mutex held.
func applyHARuleUpdate(rs *haRuleState, query map[string][]string) {
	if del := query["delete"]; len(del) > 0 {
		for _, list := range del {
			for k := range strings.SplitSeq(list, ",") {
				switch strings.TrimSpace(k) {
				case "nodes":
					rs.nodes = ""
				case "affinity":
					rs.affinity = ""
				case "strict":
					rs.strict = false
				case "disable":
					rs.disable = false
				case "comment":
					rs.comment = ""
				}
			}
		}
	}

	if v, ok := query["resources"]; ok && len(v) > 0 {
		rs.resources = strings.Split(v[0], ",")
	}
	if v, ok := query["nodes"]; ok && len(v) > 0 {
		rs.nodes = v[0]
	}
	if v, ok := query["affinity"]; ok && len(v) > 0 {
		rs.affinity = v[0]
	}
	if v, ok := query["strict"]; ok && len(v) > 0 {
		rs.strict = v[0] == "1"
	}
	if v, ok := query["disable"]; ok && len(v) > 0 {
		rs.disable = v[0] == "1"
	}
	if v, ok := query["comment"]; ok && len(v) > 0 {
		rs.comment = v[0]
	}
}
