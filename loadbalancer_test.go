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
	"errors"
	"sync"
	"testing"
	"time"

	"resty.dev/v3"
)

func TestNodeFromPath(t *testing.T) {
	tests := []struct {
		path     string
		wantNode string
		wantOK   bool
	}{
		{path: "/nodes/pve1/status", wantNode: "pve1", wantOK: true},
		{path: "/nodes", wantOK: false},
		{path: "/nodes/", wantOK: false},
		{path: "/nodes/pve1", wantNode: "pve1", wantOK: true},
		{path: "/cluster/status", wantOK: false},
		{path: "/version", wantOK: false},
		{path: "/nodes/pve%2f1/x", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			node, ok := nodeFromPath(tt.path)
			if ok != tt.wantOK || node != tt.wantNode {
				t.Fatalf("nodeFromPath(%q) = (%q, %v), want (%q, %v)", tt.path, node, ok, tt.wantNode, tt.wantOK)
			}
		})
	}
}

func TestRegistryPrecedence(t *testing.T) {
	cfg := &nodeAffinityConfig{
		endpoints: map[string]string{"pve1": "https://10.0.0.1:8006"},
		template:  "https://{node}.template.example.com:8006",
	}
	r := newRegistry(cfg, []string{"https://pve1.pool.example.com:8006", "https://pve2.pool.example.com:8006"})

	// Static map wins over template and pool for pve1.
	ep, ok := r.lookup("pve1")
	if !ok || ep.BaseURL != "https://10.0.0.1:8006" {
		t.Fatalf("lookup(pve1) = (%+v, %v), want static map entry", ep, ok)
	}

	// Template wins over pool for pve2, since a template is configured.
	ep, ok = r.lookup("pve2")
	if !ok || ep.BaseURL != "https://pve2.template.example.com:8006" {
		t.Fatalf("lookup(pve2) = (%+v, %v), want template entry", ep, ok)
	}

	// Pool wins when neither static map nor template is configured.
	poolOnly := newRegistry(&nodeAffinityConfig{}, []string{"https://pve3.pool.example.com:8006"})
	ep, ok = poolOnly.lookup("pve3")
	if !ok || ep.BaseURL != "https://pve3.pool.example.com:8006" {
		t.Fatalf("lookup(pve3) = (%+v, %v), want pool entry", ep, ok)
	}

	if _, ok := poolOnly.lookup("unknown"); ok {
		t.Fatalf("lookup(unknown) = ok, want a miss")
	}

	// Template wins over resolver, since a template unconditionally matches
	// every node name — including one the resolver also has an entry for.
	res := &stubResolver{eps: []NodeEndpoint{
		{Node: "pve2", BaseURL: "https://resolved.example.com:8006"},
	}}
	resolverReg := newRegistry(&nodeAffinityConfig{
		template: "https://{node}.template.example.com:8006",
		resolver: res,
	}, nil)
	defer resolverReg.Close()

	ep, ok = resolverReg.lookup("pve2")
	if !ok || ep.BaseURL != "https://pve2.template.example.com:8006" {
		t.Fatalf("lookup(pve2) = (%+v, %v), want the template entry (higher precedence than resolver)", ep, ok)
	}

	// Resolver still wins over pool matching when no template is configured.
	res2 := &stubResolver{eps: []NodeEndpoint{{Node: "pve3", BaseURL: "https://resolved.example.com:8006"}}}
	resolverOverPool := newRegistry(&nodeAffinityConfig{resolver: res2}, []string{"https://pve3.pool.example.com:8006"})
	defer resolverOverPool.Close()

	waitForLookup(t, resolverOverPool, "pve3", "https://resolved.example.com:8006")
}

// stubResolver is a NodeResolver whose response and error are settable at
// runtime, and which counts calls, for exercising registry's refresh loop.
type stubResolver struct {
	mu    sync.Mutex
	eps   []NodeEndpoint
	err   error
	calls int
}

func (s *stubResolver) Nodes(_ context.Context) ([]NodeEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++
	if s.err != nil {
		return nil, s.err
	}

	return s.eps, nil
}

func (s *stubResolver) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.calls
}

func (s *stubResolver) setErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.err = err
}

// waitForLookup polls r.lookup(node) until it returns wantBaseURL or a
// deadline passes, for asserting on the async resolver refresh goroutine
// without a fixed sleep.
func waitForLookup(t *testing.T, r *registry, node, wantBaseURL string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ep, ok := r.lookup(node); ok && ep.BaseURL == wantBaseURL {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("lookup(%q) never returned %q within the deadline", node, wantBaseURL)
}

func TestRegistryResolverRefreshFailureKeepsPreviousMap(t *testing.T) {
	res := &stubResolver{eps: []NodeEndpoint{{Node: "pve1", BaseURL: "https://10.0.0.1:8006"}}}

	r := newRegistry(&nodeAffinityConfig{resolver: res, resolverRefresh: 10 * time.Millisecond}, nil)
	defer r.Close()

	waitForLookup(t, r, "pve1", "https://10.0.0.1:8006")

	res.setErr(errors.New("resolver unavailable"))
	time.Sleep(50 * time.Millisecond) // let a handful of failing refreshes happen

	ep, ok := r.lookup("pve1")
	if !ok || ep.BaseURL != "https://10.0.0.1:8006" {
		t.Fatalf("lookup(pve1) = (%+v, %v), want the previous map preserved across a failed refresh", ep, ok)
	}
}

func TestRegistryResolverCloseStopsRefresh(t *testing.T) {
	res := &stubResolver{eps: []NodeEndpoint{{Node: "pve1", BaseURL: "https://10.0.0.1:8006"}}}

	r := newRegistry(&nodeAffinityConfig{resolver: res, resolverRefresh: 10 * time.Millisecond}, nil)
	waitForLookup(t, r, "pve1", "https://10.0.0.1:8006")

	if err := r.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("second Close() error = %v, want nil (idempotent)", err)
	}

	callsAtClose := res.callCount()
	time.Sleep(50 * time.Millisecond)

	if got := res.callCount(); got != callsAtClose {
		t.Fatalf("resolver called %d more times after Close, want 0", got-callsAtClose)
	}
}

func TestPoolMatch(t *testing.T) {
	tests := []struct {
		name     string
		urls     []string
		node     string
		wantOK   bool
		wantBase string
	}{
		{
			name:     "fqdn first label",
			urls:     []string{"https://pve1.example.com:8006"},
			node:     "pve1",
			wantOK:   true,
			wantBase: "https://pve1.example.com:8006",
		},
		{
			name:   "bare IP never matches",
			urls:   []string{"https://10.0.0.1:8006"},
			node:   "10.0.0.1",
			wantOK: false,
		},
		{
			name:     "case-insensitive",
			urls:     []string{"https://PVE1.example.com:8006"},
			node:     "pve1",
			wantOK:   true,
			wantBase: "https://PVE1.example.com:8006",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRegistry(&nodeAffinityConfig{}, tt.urls)
			ep, ok := r.lookup(tt.node)
			if ok != tt.wantOK {
				t.Fatalf("lookup(%q) ok = %v, want %v", tt.node, ok, tt.wantOK)
			}
			if ok && ep.BaseURL != tt.wantBase {
				t.Fatalf("lookup(%q).BaseURL = %q, want %q", tt.node, ep.BaseURL, tt.wantBase)
			}
		})
	}
}

func TestHealth(t *testing.T) {
	const baseURL = "https://pve1.example.com:8006"

	h := newHealth(3, time.Hour)
	defer h.Close()

	if !h.usable(baseURL) {
		t.Fatalf("fresh endpoint should be usable")
	}

	h.observe(baseURL, false)
	h.observe(baseURL, false)
	if !h.usable(baseURL) {
		t.Fatalf("endpoint should still be usable below MaxFailures")
	}

	h.observe(baseURL, false)
	if h.usable(baseURL) {
		t.Fatalf("endpoint should be unusable at MaxFailures")
	}

	h.observe(baseURL, true)
	if !h.usable(baseURL) {
		t.Fatalf("a success should reset the breaker")
	}
}

func TestHealthRecovery(t *testing.T) {
	const baseURL = "https://pve1.example.com:8006"

	h := newHealth(1, 20*time.Millisecond)
	defer h.Close()

	h.observe(baseURL, false)
	if h.usable(baseURL) {
		t.Fatalf("endpoint should be unusable after MaxFailures=1")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if h.usable(baseURL) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("endpoint did not recover within the deadline")
}

// stubDelegate is a resty.LoadBalancer that records every call, for
// asserting NextWithContext's decision matrix.
type stubDelegate struct {
	nextCalls int
	feedback  []*resty.RequestFeedback
	closed    bool
	nextURL   string
	nextErr   error
}

func (s *stubDelegate) NextWithContext(_ context.Context) (string, error) {
	s.nextCalls++
	if s.nextErr != nil {
		return "", s.nextErr
	}
	return s.nextURL, nil
}

func (s *stubDelegate) Feedback(f *resty.RequestFeedback) { s.feedback = append(s.feedback, f) }

func (s *stubDelegate) Close() error {
	s.closed = true
	return nil
}

func TestNodeBalancerNextWithContext(t *testing.T) {
	const (
		pve1URL     = "https://10.0.0.1:8006"
		delegateURL = "https://vip.example.com:8006"
	)

	newBalancer := func() (*NodeBalancer, *stubDelegate) {
		delegate := &stubDelegate{nextURL: delegateURL}
		cfg := &nodeAffinityConfig{
			endpoints:   map[string]string{"pve1": pve1URL},
			maxFailures: 1,
			recovery:    time.Hour,
		}
		b := newNodeBalancer(delegate, cfg, nil)
		return b, delegate
	}

	t.Run("RouteNotNodeScoped", func(t *testing.T) {
		b, delegate := newBalancer()
		defer b.Close()

		got, err := b.NextWithContext(context.Background())
		if err != nil {
			t.Fatalf("NextWithContext() error = %v", err)
		}
		if got != delegateURL {
			t.Fatalf("NextWithContext() = %q, want delegate URL", got)
		}
		if delegate.nextCalls != 1 {
			t.Fatalf("delegate called %d times, want 1", delegate.nextCalls)
		}
	})

	t.Run("RouteUnknown", func(t *testing.T) {
		b, delegate := newBalancer()
		defer b.Close()

		ctx := withRoute(context.Background(), &route{node: "pve-unknown"})
		got, err := b.NextWithContext(ctx)
		if err != nil {
			t.Fatalf("NextWithContext() error = %v", err)
		}
		if got != delegateURL {
			t.Fatalf("NextWithContext() = %q, want delegate URL", got)
		}
		if delegate.nextCalls != 1 {
			t.Fatalf("delegate called %d times, want 1", delegate.nextCalls)
		}
	})

	t.Run("RouteDirect", func(t *testing.T) {
		b, delegate := newBalancer()
		defer b.Close()

		ctx := withRoute(context.Background(), &route{node: "pve1"})
		got, err := b.NextWithContext(ctx)
		if err != nil {
			t.Fatalf("NextWithContext() error = %v", err)
		}
		if got != pve1URL {
			t.Fatalf("NextWithContext() = %q, want %q", got, pve1URL)
		}
		if delegate.nextCalls != 0 {
			t.Fatalf("delegate called %d times, want 0 for a direct route", delegate.nextCalls)
		}
	})

	t.Run("RouteUnhealthy", func(t *testing.T) {
		b, delegate := newBalancer()
		defer b.Close()

		b.health.observe(pve1URL, false) // maxFailures: 1, so this opens the breaker

		ctx := withRoute(context.Background(), &route{node: "pve1"})
		got, err := b.NextWithContext(ctx)
		if err != nil {
			t.Fatalf("NextWithContext() error = %v", err)
		}
		if got != delegateURL {
			t.Fatalf("NextWithContext() = %q, want delegate URL", got)
		}
		if delegate.nextCalls != 1 {
			t.Fatalf("delegate called %d times, want 1", delegate.nextCalls)
		}
	})

	t.Run("retry failover after maxDirectAttempts", func(t *testing.T) {
		b, delegate := newBalancer()
		defer b.Close()

		r := &route{node: "pve1"}
		ctx := withRoute(context.Background(), r)

		got, err := b.NextWithContext(ctx)
		if err != nil {
			t.Fatalf("attempt 1: NextWithContext() error = %v", err)
		}
		if got != pve1URL {
			t.Fatalf("attempt 1: NextWithContext() = %q, want %q", got, pve1URL)
		}

		got, err = b.NextWithContext(ctx)
		if err != nil {
			t.Fatalf("attempt 2: NextWithContext() error = %v", err)
		}
		if got != delegateURL {
			t.Fatalf("attempt 2: NextWithContext() = %q, want delegate URL", got)
		}
		if delegate.nextCalls != 1 {
			t.Fatalf("delegate called %d times, want 1", delegate.nextCalls)
		}
	})

	t.Run("Feedback forwards to delegate and updates health", func(t *testing.T) {
		b, delegate := newBalancer()
		defer b.Close()

		b.Feedback(&resty.RequestFeedback{BaseURL: pve1URL, Success: false})
		if len(delegate.feedback) != 1 {
			t.Fatalf("delegate received %d feedback calls, want 1", len(delegate.feedback))
		}
		if b.health.usable(pve1URL) {
			t.Fatalf("health should have opened the breaker for %q", pve1URL)
		}
	})

	t.Run("Close closes health and delegate", func(t *testing.T) {
		b, delegate := newBalancer()
		if err := b.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if !delegate.closed {
			t.Fatalf("delegate was not closed")
		}
	})
}

func TestNodeBalancerCloseReturnsDelegateError(t *testing.T) {
	wantErr := errors.New("delegate close failed")
	delegate := &closeErrDelegate{err: wantErr}

	b := newNodeBalancer(delegate, &nodeAffinityConfig{}, nil)
	if err := b.Close(); !errors.Is(err, wantErr) {
		t.Fatalf("Close() error = %v, want %v", err, wantErr)
	}
}

type closeErrDelegate struct {
	stubDelegate

	err error
}

func (d *closeErrDelegate) Close() error { return d.err }
