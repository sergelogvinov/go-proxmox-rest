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
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// firewallSkipKeys are rule/options request parameters that address or
// guard the write rather than name a stored field.
var firewallSkipKeys = map[string]bool{
	"pos":    true,
	"moveto": true,
	"digest": true,
	"delete": true,
}

// applyFirewallUpdate merges query into fields the way pve-firewall's
// copy_rule_data and delete_rule_properties do: an empty or "-" value and
// every key named in "delete" remove the field. Called with the state
// mutex held.
func applyFirewallUpdate(fields map[string]string, query url.Values) {
	if del := query.Get("delete"); del != "" {
		for k := range strings.SplitSeq(del, ",") {
			delete(fields, strings.TrimSpace(k))
		}
	}

	for k, vals := range query {
		if firewallSkipKeys[k] || len(vals) == 0 {
			continue
		}

		if vals[0] == "" || vals[0] == "-" {
			delete(fields, k)
			continue
		}

		fields[k] = vals[0]
	}
}

// firewallGuest resolves the {vmid} of a guest firewall route, writing the
// error response and reporting failure if the guest doesn't exist.
func firewallGuest(w http.ResponseWriter, r *http.Request, n *nodeState) (*vmState, bool) {
	vmid, ok := pathVMID(w, r)
	if !ok {
		return nil, false
	}

	n.cs.mu.Lock()
	vm := n.guests[vmid]
	n.cs.mu.Unlock()

	if vm == nil {
		notFound(w, "vm")
		return nil, false
	}

	return vm, true
}

// ruleView renders a stored rule at position pos.
func ruleView(rule map[string]string, pos int) map[string]any {
	out := make(map[string]any, len(rule)+1)
	for k, v := range rule {
		out[k] = v
	}

	out["pos"] = pos

	return out
}

// handleFirewallOptions backs GET/PUT
// /nodes/{node}/qemu/{vmid}/firewall/options.
func handleFirewallOptions(w http.ResponseWriter, r *http.Request, n *nodeState) {
	vm, ok := firewallGuest(w, r, n)
	if !ok {
		return
	}

	n.cs.mu.Lock()
	defer n.cs.mu.Unlock()

	if vm.fwOptions == nil {
		vm.fwOptions = map[string]string{}
	}

	switch r.Method {
	case http.MethodGet:
		writeData(w, cloneStringMap(vm.fwOptions))
	case http.MethodPut:
		applyFirewallUpdate(vm.fwOptions, r.URL.Query())
		writeData(w, nil)
	default:
		methodNotAllowed(w)
	}
}

// handleFirewallRules backs GET/POST /nodes/{node}/qemu/{vmid}/firewall/rules.
// Like pve-firewall's create_rule, a new rule is always inserted at
// position 0, shifting existing rules down (a "pos" parameter is ignored),
// and is disabled unless "enable" is given.
func handleFirewallRules(w http.ResponseWriter, r *http.Request, n *nodeState) {
	vm, ok := firewallGuest(w, r, n)
	if !ok {
		return
	}

	n.cs.mu.Lock()
	defer n.cs.mu.Unlock()

	switch r.Method {
	case http.MethodGet:
		list := make([]map[string]any, len(vm.fwRules))
		for i, rule := range vm.fwRules {
			list[i] = ruleView(rule, i)
		}

		writeData(w, list)

	case http.MethodPost:
		q := r.URL.Query()
		if q.Get("type") == "" || q.Get("action") == "" {
			writeError(w, http.StatusBadRequest, "parameter verification failed", map[string]string{"type": "type and action are required"})
			return
		}

		rule := map[string]string{}
		applyFirewallUpdate(rule, q)

		if _, ok := rule["enable"]; !ok {
			rule["enable"] = "0"
		}

		vm.fwRules = slices.Insert(vm.fwRules, 0, rule)
		writeData(w, nil)

	default:
		methodNotAllowed(w)
	}
}

// handleFirewallRule backs GET/PUT/DELETE
// /nodes/{node}/qemu/{vmid}/firewall/rules/{pos}. A PUT with "moveto"
// only repositions the rule, using pve-firewall's update_rule algorithm;
// otherwise it updates the rule's fields. A DELETE removes the rule and
// renumbers every later one.
func handleFirewallRule(w http.ResponseWriter, r *http.Request, n *nodeState) {
	vm, ok := firewallGuest(w, r, n)
	if !ok {
		return
	}

	pos, err := strconv.Atoi(r.PathValue("pos"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "parameter verification failed", map[string]string{"pos": "not a number"})
		return
	}

	n.cs.mu.Lock()
	defer n.cs.mu.Unlock()

	if pos < 0 || pos >= len(vm.fwRules) {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("no rule at position %d", pos), nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeData(w, ruleView(vm.fwRules[pos], pos))

	case http.MethodPut:
		q := r.URL.Query()

		if mv := q.Get("moveto"); mv != "" {
			moveto, err := strconv.Atoi(mv)
			if err != nil {
				writeError(w, http.StatusBadRequest, "parameter verification failed", map[string]string{"moveto": "not a number"})
				return
			}

			if moveto != pos {
				rule := vm.fwRules[pos]
				moved := make([]map[string]string, 0, len(vm.fwRules))

				for i, other := range vm.fwRules {
					if i == pos {
						continue
					}

					if i == moveto {
						moved = append(moved, rule)
					}

					moved = append(moved, other)
				}

				if moveto >= len(vm.fwRules) {
					moved = append(moved, rule)
				}

				vm.fwRules = moved
			}

			writeData(w, nil)

			return
		}

		applyFirewallUpdate(vm.fwRules[pos], q)
		writeData(w, nil)

	case http.MethodDelete:
		vm.fwRules = slices.Delete(vm.fwRules, pos, pos+1)
		writeData(w, nil)

	default:
		methodNotAllowed(w)
	}
}
