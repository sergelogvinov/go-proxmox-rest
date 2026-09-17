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
	"encoding/json"
	"net/http"
)

// wireEnvelope mirrors the root package's envelope type: every response
// this fake writes uses this exact shape, so proxmox.decodeInto/
// params.Decode and the *proxmox.APIError predicates run against it
// unmodified — see docs/fakeapi.md §3.
type wireEnvelope struct {
	Data    any               `json:"data"`
	Errors  map[string]string `json:"errors,omitempty"`
	Message string            `json:"message,omitempty"`
}

// writeData writes a 200 response with v as the envelope's "data".
func writeData(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(wireEnvelope{Data: v}) //nolint:errchkjson // v is always a JSON-marshalable domain type or map/string built by this package
}

// writeError writes a Proxmox-shaped error envelope with the given status.
func writeError(w http.ResponseWriter, status int, message string, errs map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(wireEnvelope{Errors: errs, Message: message}) //nolint:errchkjson // Errors/Message are plain strings/maps, always marshalable
}

// notFound writes the 404 shape proxmox.IsNotFound checks for.
func notFound(w http.ResponseWriter, what string) {
	writeError(w, http.StatusNotFound, what+" does not exist", nil)
}

// methodNotAllowed writes a 405 for a method this fake doesn't implement
// on an otherwise-known path, matching docs/fakeapi.md §7's "everything
// else 404s" philosophy for unimplemented surface, applied at the
// per-method level here since the path itself is recognized.
func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed", nil)
}

// unreachable writes the 596 shape a FailureUnreachable node responds
// with — the status real Proxmox's inter-node API proxy returns when it
// cannot reach a cluster member. proxmox.IsUnexpected reports true for
// any 5xx, 596 included.
func unreachable(w http.ResponseWriter) {
	writeError(w, 596, "no route to host (596)", nil)
}

// hijackConnRefused closes the underlying TCP connection outright instead
// of writing an HTTP response, simulating FailureConnRefused: the client
// sees a transport error, not a *proxmox.APIError.
func hijackConnRefused(w http.ResponseWriter) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	conn, _, err := hj.Hijack()
	if err != nil {
		return
	}
	_ = conn.Close()
}
