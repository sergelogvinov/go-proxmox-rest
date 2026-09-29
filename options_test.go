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

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBasePathResolution asserts the final request URL under the option
// combinations from docs/node-lb.md §8.1/§9: a load balancer's own base
// URLs carry no path, so basePath is the only source of the API prefix
// once one is installed, and the balancer options must not clobber a
// caller's explicit WithBasePath (the §9 regression).
func TestBasePathResolution(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":null}`))
	}))
	defer srv.Close()

	tests := []struct {
		name     string
		opts     func(url string) []Option
		wantPath string
	}{
		{
			name: "WithURL with path, no balancer",
			opts: func(url string) []Option {
				return []Option{WithURL(url + "/api2/json")}
			},
			wantPath: "/api2/json/version",
		},
		{
			name: "WithURL with path, plus WithNodeAffinity",
			opts: func(url string) []Option {
				return []Option{WithURL(url + "/api2/json"), WithNodeAffinity()}
			},
			wantPath: "/api2/json/version",
		},
		{
			name: "WithURL without path, plus WithNodeAffinity falls back to default",
			opts: func(url string) []Option {
				return []Option{WithURL(url), WithNodeAffinity()}
			},
			wantPath: "/api2/json/version",
		},
		{
			name: "WithBasePath then WithRoundRobin keeps the custom prefix",
			opts: func(url string) []Option {
				return []Option{WithBasePath("/pve/api2/json"), WithRoundRobin(url)}
			},
			wantPath: "/pve/api2/json/version",
		},
		{
			name: "WithBasePath then a path-carrying WithURL: last one wins",
			opts: func(url string) []Option {
				return []Option{WithBasePath("/custom"), WithURL(url + "/api2/json")}
			},
			wantPath: "/api2/json/version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(ClientConfig{}, tt.opts(srv.URL)...)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			defer c.Close()

			var out any
			if err := c.Get(context.Background(), "/version", &out, nil); err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if gotPath != tt.wantPath {
				t.Fatalf("request path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

// TestLBURLsClearedByOpaqueBalancer guards against a stale pool: pool
// matching (§6.3 of docs/node-lb.md) is documented as yielding nothing for
// WithLoadBalancer and WithSRVWeightedRoundRobin, since a custom balancer is
// opaque and SRV targets are resolved inside resty. That is only true if
// lbURLs left over from an earlier balancer option (applied first, then
// overridden) is actually cleared — otherwise node affinity's default
// endpoint source would keep matching node names against a pool that no
// longer has anything to do with the active balancer.
//
// WithSRVWeightedRoundRobin does the same clearing (see its source), but
// isn't exercised here: unlike WithLoadBalancer it performs a real
// net.LookupSRV at option-apply time, so it can't be driven to its success
// path in a hermetic unit test without injecting a fake resolver.
func TestLBURLsClearedByOpaqueBalancer(t *testing.T) {
	cfg := ClientConfig{}
	WithRoundRobin("https://pve1.example.com:8006")(&cfg)
	WithLoadBalancer(&stubDelegate{nextURL: "https://vip.example.com:8006"})(&cfg)

	if cfg.lbURLs != nil {
		t.Fatalf("lbURLs = %v, want nil after WithLoadBalancer overrides WithRoundRobin", cfg.lbURLs)
	}
}

// TestWithURLClearsEarlierBalancer guards against the mirror image of
// TestLBURLsClearedByOpaqueBalancer: WithURL is the one option that does not
// itself set cfg.lb, so without explicitly clearing it, calling WithURL
// after a balancer option would leave the earlier balancer active in cfg.lb
// while cfg.lbURLs (and node-affinity's pool-matching source) pointed at the
// single URL WithURL was just given — a mismatch between what actually
// serves traffic and what pool matching believes the pool is.
func TestWithURLClearsEarlierBalancer(t *testing.T) {
	cfg := ClientConfig{}
	WithRoundRobin("https://pve1.example.com:8006", "https://pve2.example.com:8006")(&cfg)
	WithURL("https://pve-vip.example.com:8006")(&cfg)

	if cfg.lb != nil {
		t.Fatalf("lb = %v, want nil after WithURL overrides WithRoundRobin", cfg.lb)
	}
	if len(cfg.lbURLs) != 1 || cfg.lbURLs[0] != "https://pve-vip.example.com:8006" {
		t.Fatalf("lbURLs = %v, want [https://pve-vip.example.com:8006]", cfg.lbURLs)
	}
}
