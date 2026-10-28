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
	"context"
	"errors"
	"net/http"
	"strings"
)

// faultStatus is the HTTP status a request fault answers with: the
// generic 500 real Proxmox returns for a failed API call.
const faultStatus = http.StatusInternalServerError

// fault is one injected failure, registered via Cluster.FailRequests.
type fault struct {
	message string

	// failTask makes the request succeed but its task end in error, with
	// the task's mutation still applied. See AsFailedTask.
	failTask bool
}

// FaultOption configures a fault registered via Cluster.FailRequests.
type FaultOption func(*fault)

// WithFaultMessage sets the error message the fault reports, in place of
// the default "fakeapi: injected fault".
func WithFaultMessage(message string) FaultOption {
	return func(f *fault) { f.message = message }
}

// AsFailedTask makes a fault on a task-backed route answer normally with a
// UPID whose task then ends in error. The task's mutation is still applied,
// modeling a Proxmox task that failed partway and left its work behind
// (a clone that created the guest, an upload that wrote the volume). On a
// route that starts no task, the request succeeds untouched.
func AsFailedTask() FaultOption {
	return func(f *fault) { f.failTask = true }
}

// FailRequests makes every request matching route fail until ClearFaults.
// route is a method and a path pattern exactly as this fake routes it,
// e.g. "POST /nodes/{node}/qemu/{vmid}/clone" or
// "PUT /nodes/{node}/qemu/{vmid}/config": it matches that route for every
// node and guest. By default a matching request gets a Proxmox-shaped 500
// and changes nothing; AsFailedTask changes that. Registering a route
// again replaces its fault.
func (cl *Cluster) FailRequests(route string, opts ...FaultOption) {
	f := &fault{message: "fakeapi: injected fault"}
	for _, opt := range opts {
		opt(f)
	}

	cl.state.mu.Lock()
	defer cl.state.mu.Unlock()

	cl.state.faults[route] = f
}

// ClearFaults removes every fault registered via FailRequests.
func (cl *Cluster) ClearFaults() {
	cl.state.mu.Lock()
	defer cl.state.mu.Unlock()

	clear(cl.state.faults)
}

// faultKey is the context key a task-failing fault travels under from
// withFaults to startTask.
type faultKey struct{}

// withFaults answers a request matching a registered fault before next
// runs, or for an AsFailedTask fault hands it to startTask via the
// request's context. pattern is the route's registration pattern, with or
// without a method prefix; a bare pattern matches each method separately.
func withFaults(state *clusterState, pattern string, next http.HandlerFunc) http.HandlerFunc {
	path := pattern
	if _, rest, ok := strings.Cut(pattern, " "); ok {
		path = rest
	}

	return func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		f := state.faults[r.Method+" "+path]
		state.mu.Unlock()

		switch {
		case f == nil:
			next(w, r)
		case f.failTask:
			next(w, r.WithContext(context.WithValue(r.Context(), faultKey{}, f)))
		default:
			writeError(w, faultStatus, f.message, nil)
		}
	}
}

// taskFault wraps apply so the task it backs ends in error after applying
// its mutation, when ctx carries an AsFailedTask fault.
func taskFault(ctx context.Context, apply func() error) func() error {
	f, ok := ctx.Value(faultKey{}).(*fault)
	if !ok {
		return apply
	}

	return func() error {
		if err := apply(); err != nil {
			return err
		}

		return errors.New(f.message)
	}
}
