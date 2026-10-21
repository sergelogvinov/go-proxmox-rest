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

package ha

import (
	"context"
	"maps"
	"testing"
)

// recordingGetter captures the last request made through it.
type recordingGetter struct {
	method string
	path   string
	params map[string]string
}

func (g *recordingGetter) record(method, path string, params map[string]string) error {
	g.method, g.path, g.params = method, path, params

	return nil
}

func (g *recordingGetter) Get(_ context.Context, path string, _ any, params map[string]string) error {
	return g.record("GET", path, params)
}

func (g *recordingGetter) Create(_ context.Context, path string, _ any, params map[string]string) error {
	return g.record("POST", path, params)
}

func (g *recordingGetter) Update(_ context.Context, path string, _ any, params map[string]string) error {
	return g.record("PUT", path, params)
}

func (g *recordingGetter) Delete(_ context.Context, path string, _ any, params map[string]string) error {
	return g.record("DELETE", path, params)
}

func TestResourcesRequests(t *testing.T) {
	ctx := t.Context()
	comment := "e2e"
	failback := false

	tests := []struct {
		name   string
		call   func(r *resourcesResource) error
		method string
		path   string
		params map[string]string
	}{
		{
			name:   "list unfiltered",
			call:   func(r *resourcesResource) error { _, err := r.List(ctx, ""); return err },
			method: "GET",
			path:   "/cluster/ha/resources",
		},
		{
			name:   "list by type",
			call:   func(r *resourcesResource) error { _, err := r.List(ctx, ResourceTypeCT); return err },
			method: "GET",
			path:   "/cluster/ha/resources",
			params: map[string]string{"type": "ct"},
		},
		{
			name: "create",
			call: func(r *resourcesResource) error {
				return r.Create(ctx, &ResourceOptions{SID: "vm:100", State: ResourceStateStopped, Comment: &comment})
			},
			method: "POST",
			path:   "/cluster/ha/resources",
			params: map[string]string{"sid": "vm:100", "state": "stopped", "comment": "e2e"},
		},
		{
			name: "update clears failback",
			call: func(r *resourcesResource) error {
				return r.Update(ctx, "vm:100", &ResourceOptions{Failback: &failback})
			},
			method: "PUT",
			path:   "/cluster/ha/resources/vm:100",
			params: map[string]string{"failback": "0"},
		},
		{
			name:   "delete keeps rules",
			call:   func(r *resourcesResource) error { return r.Delete(ctx, "vm:100", false) },
			method: "DELETE",
			path:   "/cluster/ha/resources/vm:100",
			params: map[string]string{"purge": "0"},
		},
		{
			name:   "delete purges rules",
			call:   func(r *resourcesResource) error { return r.Delete(ctx, "vm:100", true) },
			method: "DELETE",
			path:   "/cluster/ha/resources/vm:100",
			params: map[string]string{"purge": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &recordingGetter{}
			if err := tt.call(New(g).Resources()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if g.method != tt.method || g.path != tt.path {
				t.Errorf("request = %s %s, want %s %s", g.method, g.path, tt.method, tt.path)
			}
			if len(g.params) != 0 || len(tt.params) != 0 {
				if !maps.Equal(g.params, tt.params) {
					t.Errorf("params = %v, want %v", g.params, tt.params)
				}
			}
		})
	}
}

func TestResourcesValidation(t *testing.T) {
	r := New(&recordingGetter{}).Resources()
	ctx := t.Context()

	if err := r.Create(ctx, nil); err == nil {
		t.Error("create with nil options: expected error")
	}
	if err := r.Create(ctx, &ResourceOptions{State: ResourceStateStarted}); err == nil {
		t.Error("create with missing sid: expected error")
	}
	if err := r.Update(ctx, "vm:100", nil); err == nil {
		t.Error("update with nil options: expected error")
	}
}
