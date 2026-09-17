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
	"time"
)

// handleTicket backs POST /access/ticket: it issues a session for any
// non-empty username (the fake does not check passwords), which
// requireAuth then validates on subsequent ticket-auth requests.
func handleTicket(state *clusterState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username == "" {
			writeError(w, http.StatusBadRequest, "parameter verification failed",
				map[string]string{"username": "property is missing and it is not optional"})
			return
		}

		now := time.Now().Unix()
		ticket := fmt.Sprintf("PVE:%s:%08X::fakeapi", username, now)
		csrf := fmt.Sprintf("%08X:fakeapi", now)

		state.mu.Lock()
		if state.tickets == nil {
			state.tickets = map[string]string{}
		}
		state.tickets[ticket] = username
		state.mu.Unlock()

		http.SetCookie(w, &http.Cookie{Name: "PVEAuthCookie", Value: ticket, Path: "/"})
		writeData(w, map[string]string{
			"username":            username,
			"ticket":              ticket,
			"CSRFPreventionToken": csrf,
		})
	}
}
