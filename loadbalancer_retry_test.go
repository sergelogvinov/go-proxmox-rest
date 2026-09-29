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
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestErrIsUnsent(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "plain error", err: errors.New("boom"), want: false},
		{
			name: "dial error (connection refused)",
			err:  &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			want: true,
		},
		{
			name: "dial error wrapped further",
			err:  fmt.Errorf("Get %q: %w", "http://x", &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}),
			want: true,
		},
		{
			name: "read error is not a dial failure",
			err:  &net.OpError{Op: "read", Net: "tcp", Err: errors.New("connection reset")},
			want: false,
		},
		{
			name: "TLS certificate verification failure",
			err:  &tls.CertificateVerificationError{Err: errors.New("x509: certificate signed by unknown authority")},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errIsUnsent(tt.err); got != tt.want {
				t.Fatalf("errIsUnsent(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestSafePOSTFailover exercises docs/node-lb.md §7.2/§11 phase 3: a POST
// against a pinned, dead node fails outright by default, but with
// WithSafePOSTFailover it fails over to the delegate — because a
// connection-refused error proves the request never reached the server.
func TestSafePOSTFailover(t *testing.T) {
	var gotPath string

	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":"UPID:pve1:00000001:qmcreate:"}`))
	}))
	defer healthy.Close()

	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := dead.URL
	dead.Close() // dead from the start: connection refused for every request

	t.Run("without WithSafePOSTFailover, POST fails outright", func(t *testing.T) {
		c, err := New(ClientConfig{},
			WithURL(healthy.URL),
			WithNodeAffinity(WithNodeEndpoints(map[string]string{"pve1": deadURL})),
		)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer c.Close()

		var out string
		err = c.Create(t.Context(), "/nodes/pve1/qemu", &out, map[string]string{"vmid": "100"})
		if err == nil {
			t.Fatalf("POST against a dead pinned node succeeded, want it to fail outright")
		}

		if apiErr, ok := errors.AsType[*APIError](err); ok {
			t.Fatalf("error = %v, want a bare transport error, not a structured API error", apiErr)
		}
	})

	t.Run("with WithSafePOSTFailover, POST falls back to the delegate", func(t *testing.T) {
		gotPath = ""

		c, err := New(ClientConfig{},
			WithURL(healthy.URL),
			WithRetryCount(1), // a retry is what turns the failure into a fallback
			WithRetryWaitTime(5*time.Millisecond),
			WithNodeAffinity(
				WithNodeEndpoints(map[string]string{"pve1": deadURL}),
				WithSafePOSTFailover(),
			),
		)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer c.Close()

		var out string
		if err := c.Create(t.Context(), "/nodes/pve1/qemu", &out, map[string]string{"vmid": "100"}); err != nil {
			t.Fatalf("Create() error = %v, want the delegate to serve it", err)
		}
		if out != "UPID:pve1:00000001:qmcreate:" {
			t.Fatalf("out = %q, want the delegate's response", out)
		}
		if gotPath != "/api2/json/nodes/pve1/qemu" {
			t.Fatalf("delegate saw path %q, want /api2/json/nodes/pve1/qemu", gotPath)
		}
	})

	t.Run("with WithSafePOSTFailover, a GET against the same dead node still fails outright without retries enabled", func(t *testing.T) {
		c, err := New(ClientConfig{},
			WithURL(healthy.URL),
			WithRetryCount(0),
			WithNodeAffinity(
				WithNodeEndpoints(map[string]string{"pve1": deadURL}),
				WithSafePOSTFailover(),
			),
		)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer c.Close()

		var out any
		if err := c.Get(t.Context(), "/nodes/pve1/status", &out, nil); err == nil {
			t.Fatalf("GET against a dead pinned node with retries disabled succeeded, want it to fail")
		}
	})
}
