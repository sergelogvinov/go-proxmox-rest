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
	"strings"

	"resty.dev/v3"
)

// maxDirectAttempts is the attempt number after which a pinned request stops
// going direct and falls back to the delegate balancer. It is a constant,
// not an option: any value above 1 means "retry the same dead node before
// trying anyone else", which only helps when the delegate pool has a single
// member, in which case falling back and retrying direct are the same
// request.
const maxDirectAttempts = 1

// nodeFromPath returns the node name from a node-scoped API path, i.e. the
// {node} in /nodes/{node}/... . It reports false for /nodes and /nodes/ (the
// cluster-wide node list) and for every non-node path. The path is matched
// as passed to Client.send, before basePath is prepended.
func nodeFromPath(p string) (string, bool) {
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return "", false
	}

	segments := strings.SplitN(p, "/", 3)
	if len(segments) < 2 || segments[0] != "nodes" {
		return "", false
	}

	node := segments[1]
	if node == "" || strings.ContainsAny(node, "%/") {
		return "", false
	}

	return node, true
}

// route is the per-request routing state. It is created by send, carried in
// the request context, and mutated by NextWithContext on each attempt. It is
// a pointer, deliberately: the context value cannot be replaced between
// retry attempts (resty holds one *Request), so a mutable cell is the only
// way for attempt n to know what attempt n-1 did.
type route struct {
	node     string
	attempts int
	pinned   []string
}

type routeCtxKey struct{}

// withRoute attaches r to ctx.
func withRoute(ctx context.Context, r *route) context.Context {
	return context.WithValue(ctx, routeCtxKey{}, r)
}

// routeFrom returns the route attached to ctx, or nil if none was attached —
// the normal, expected case for requests issued outside send (e.g.
// Client.ticket) or for paths nodeFromPath does not recognize.
func routeFrom(ctx context.Context) *route {
	r, _ := ctx.Value(routeCtxKey{}).(*route)

	return r
}

// RouteDecision reports how NodeBalancer routed one balancer attempt; see
// WithOnRoute.
type RouteDecision int

const (
	// RouteDirect means the request went straight to the node's own
	// endpoint.
	RouteDirect RouteDecision = iota
	// RouteUnknown means the node has no known endpoint, so the request
	// was delegated.
	RouteUnknown
	// RouteUnhealthy means the node's endpoint is known but currently
	// circuit-broken, so the request was delegated.
	RouteUnhealthy
	// RouteNotNodeScoped means the request path carried no node hint, so
	// the request was delegated as it always would have been.
	RouteNotNodeScoped
)

// NodeBalancer implements resty.LoadBalancer, routing node-scoped requests
// (see nodeFromPath) directly at the node that owns them and delegating
// everything else — and any direct attempt that cannot be served — to the
// wrapped delegate balancer. See docs/node-lb.md for the full design.
type NodeBalancer struct {
	delegate resty.LoadBalancer
	registry *registry
	health   *health
	onRoute  func(node, baseURL string, d RouteDecision)
}

var _ resty.LoadBalancer = (*NodeBalancer)(nil)

// newNodeBalancer builds a NodeBalancer wrapping delegate. poolURLs is the
// pool of base URLs behind delegate, used for the pool-name-matching
// endpoint source (§6.3) when cfg does not otherwise resolve a node.
func newNodeBalancer(delegate resty.LoadBalancer, cfg *nodeAffinityConfig, poolURLs []string) *NodeBalancer {
	return &NodeBalancer{
		delegate: delegate,
		registry: newRegistry(cfg, poolURLs),
		health:   newHealth(cfg.maxFailures, cfg.recovery),
		onRoute:  cfg.onRoute,
	}
}

// NextWithContext implements resty.LoadBalancer. See docs/node-lb.md §5 for
// the full decision rationale: falling back is the only behavior, and it
// is never silent — every miss is reported through onRoute.
func (b *NodeBalancer) NextWithContext(ctx context.Context) (string, error) {
	r := routeFrom(ctx)
	if r == nil || r.node == "" {
		b.report("", "", RouteNotNodeScoped)

		return b.delegate.NextWithContext(ctx)
	}

	r.attempts++
	if r.attempts > maxDirectAttempts {
		return b.delegate.NextWithContext(ctx)
	}

	ep, ok := b.registry.lookup(r.node)
	if !ok {
		b.report(r.node, "", RouteUnknown)

		return b.delegate.NextWithContext(ctx)
	}

	if !b.health.usable(ep.BaseURL) {
		b.report(r.node, ep.BaseURL, RouteUnhealthy)

		return b.delegate.NextWithContext(ctx)
	}

	r.pinned = append(r.pinned, ep.BaseURL)
	b.report(r.node, ep.BaseURL, RouteDirect)

	return ep.BaseURL, nil
}

// Feedback records the outcome against the affinity layer's own health
// state and unconditionally forwards it to the delegate, which may have its
// own health state over the same base URLs (e.g. a WeightedRoundRobin
// delegate).
func (b *NodeBalancer) Feedback(f *resty.RequestFeedback) {
	if f != nil {
		b.health.observe(f.BaseURL, f.Success)
	}

	b.delegate.Feedback(f)
}

// Close stops the registry's resolver-refresh goroutine and the health
// breaker's recovery ticker, and closes the delegate, which may own its own
// goroutines (e.g. WeightedRoundRobin's recovery ticker or a resolver's
// refresh loop). All three are closed unconditionally — an error from one
// must not skip the others, or a goroutine behind whichever is skipped
// would outlive Client.Close.
func (b *NodeBalancer) Close() error {
	return errors.Join(b.registry.Close(), b.health.Close(), b.delegate.Close())
}

func (b *NodeBalancer) report(node, baseURL string, d RouteDecision) {
	if b.onRoute != nil {
		b.onRoute(node, baseURL, d)
	}
}
