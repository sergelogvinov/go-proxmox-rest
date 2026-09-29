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

package proxmox

import "testing"

// TestWithCACert asserts a single path still works (back-compat with the
// pre-variadic signature) and that repeated calls accumulate bundles rather
// than replacing the previously configured ones.
func TestWithCACert(t *testing.T) {
	cfg := ClientConfig{}
	opts := []Option{WithCACert("/single.pem"), WithCACert("/a.pem", "/b.pem")}
	for _, opt := range opts {
		opt(&cfg)
	}

	want := []string{"/single.pem", "/a.pem", "/b.pem"}
	if len(cfg.CACerts) != len(want) {
		t.Fatalf("CACerts = %v, want %v", cfg.CACerts, want)
	}
	for i, p := range want {
		if cfg.CACerts[i] != p {
			t.Fatalf("CACerts = %v, want %v", cfg.CACerts, want)
		}
	}
}
