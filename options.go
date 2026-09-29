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
	"net/url"
	"strings"
	"time"

	"resty.dev/v3"
)

// Option mutates a ClientConfig. Options are the only way to populate the
// private config fields; the last applied option wins.
type Option func(*ClientConfig)

// WithURL sets the API endpoint, e.g. https://10.0.0.1:8006/api2/json
// If the URL contains a path component, it is saved as the base path and
// prepended to request paths when a load balancer is used.
func WithURL(u string) Option {
	return func(c *ClientConfig) {
		c.BaseURL = u
		c.lb = nil
		c.lbURLs = []string{u}
		if parsed, err := url.Parse(u); err == nil && parsed.Path != "" && parsed.Path != "/" {
			c.basePath = strings.TrimSuffix(parsed.Path, "/")
		}
	}
}

// WithBasePath sets the API path prefix (e.g. /api2/json) explicitly,
// overriding any path extracted from the URL. It is used when a load
// balancer is configured, since LB endpoints carry no path information.
func WithBasePath(p string) Option {
	return func(c *ClientConfig) { c.basePath = strings.TrimSuffix(p, "/") }
}

// WithTokenAuth enables API token authentication.
// tokenID has the form user@realm!tokenid.
func WithTokenAuth(tokenID, secret string) Option {
	return func(c *ClientConfig) {
		c.Token = tokenID
		c.TokenSecret = secret
	}
}

// WithPasswordAuth enables ticket/password authentication. The realm is
// included in the username (e.g., "root@pam").
func WithPasswordAuth(username, password string) Option {
	return func(c *ClientConfig) {
		c.Username = username
		c.Password = password
	}
}

// WithTimeout sets the per-request timeout (default 5m).
func WithTimeout(d time.Duration) Option {
	return func(c *ClientConfig) { c.timeout = d }
}

// WithRetryCount sets the number of auto-retries (default 3).
func WithRetryCount(n int) Option {
	return func(c *ClientConfig) { c.retryCount = n }
}

// WithRetryWaitTime sets the initial backoff wait per retry (default 500ms).
func WithRetryWaitTime(d time.Duration) Option {
	return func(c *ClientConfig) { c.retryWaitTime = d }
}

// WithRetryMaxWaitTime sets the cap on backoff wait (default 30s).
func WithRetryMaxWaitTime(d time.Duration) Option {
	return func(c *ClientConfig) { c.retryMaxWaitTime = d }
}

// WithUserAgent sets a custom User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *ClientConfig) { c.UserAgent = ua }
}

// WithInsecure controls whether TLS certificate verification is skipped.
func WithInsecure(skip bool) Option {
	return func(c *ClientConfig) { c.Insecure = skip }
}

// WithCACert sets the path to a PEM CA bundle to trust.
func WithCACert(path string) Option {
	return func(c *ClientConfig) { c.CACert = path }
}

// WithProxy sets an optional proxy URL.
func WithProxy(p string) Option {
	return func(c *ClientConfig) { c.Proxy = p }
}

// WithLogger sets the resty logger.
func WithLogger(l resty.Logger) Option {
	return func(c *ClientConfig) { c.logger = l }
}

// WithRoundRobin distributes requests across the given API endpoints
// round-robin style. The endpoints ignore the API path prefix; New defaults
// it to /api2/json. To redefine the default API path, use WithBasePath.
func WithRoundRobin(urls ...string) Option {
	return func(c *ClientConfig) {
		if lb, err := resty.NewRoundRobin(urls...); err == nil {
			c.lb = lb
			c.lbURLs = urls
		}
	}
}

// WithWeightedRoundRobin distributes requests across the given hosts
// proportionally to their weights. Since hosts carry no path, New defaults
// requests to the standard API path (/api2/json).
// To redefine the default API path, use WithBasePath
func WithWeightedRoundRobin(hosts ...*resty.Host) Option {
	return func(c *ClientConfig) {
		if lb, err := resty.NewWeightedRoundRobin(0, hosts...); err == nil {
			c.lb = lb

			urls := make([]string, len(hosts))
			for i, h := range hosts {
				urls[i] = h.BaseURL
			}
			c.lbURLs = urls
		}
	}
}

// WithSRVWeightedRoundRobin resolves SRV records and load balances across
// the discovered endpoints. Since SRV records carry no path, New defaults
// requests to the standard API path (/api2/json).
// To redefine the default API path, use WithBasePath
func WithSRVWeightedRoundRobin(service, proto, domain, scheme string) Option {
	return func(c *ClientConfig) {
		if lb, err := resty.NewSRVWeightedRoundRobin(service, proto, domain, scheme); err == nil {
			c.lb = lb
			c.lbURLs = nil
		}
	}
}

// WithLoadBalancer sets a custom resty.LoadBalancer implementation,
// overriding the single base URL.
func WithLoadBalancer(lb resty.LoadBalancer) Option {
	return func(c *ClientConfig) {
		c.lb = lb
		c.lbURLs = nil
	}
}

// nodeAffinityConfig holds WithNodeAffinity's sub-option state.
type nodeAffinityConfig struct {
	endpoints       map[string]string
	template        string
	resolver        NodeResolver
	resolverRefresh time.Duration

	maxFailures int
	recovery    time.Duration

	onRoute          func(node, baseURL string, d RouteDecision)
	safePOSTFailover bool
}

// NodeAffinityOption mutates a nodeAffinityConfig. See WithNodeAffinity.
type NodeAffinityOption func(*nodeAffinityConfig)

// WithNodeAffinity routes /nodes/{node}/... requests directly at the node
// that owns them, falling back to the configured load-balancing algorithm
// when the node's endpoint is unknown or unreachable. See docs/node-lb.md
// for the full design.
//
// Any configured balancer — WithRoundRobin, WithWeightedRoundRobin,
// WithSRVWeightedRoundRobin, WithLoadBalancer — gets this automatically,
// using zero-config pool matching (§6.3) as its endpoint source; calling
// WithNodeAffinity is only needed to configure an explicit source
// (WithNodeEndpoints, WithNodeEndpointTemplate), tune health
// (WithNodeMaxFailures, WithNodeRecovery), install WithOnRoute, or to
// enable the feature at all for a client with no balancer, i.e. WithURL by
// itself (apply WithNodeAffinity after it).
//
// Falling back never changes what a call returns, only which host serves
// it — except for the error it returns. When the target node is down, the
// first request against it fails with a bare transport error rather than
// the structured Proxmox error a relay through another node would have
// produced; subsequent requests fall back once the node is known-bad. See
// docs/node-lb.md §5 and §7.2.
//
// TLS verification needs the endpoint host to appear in that node's
// certificate SANs, which PVE's generated certificates normally carry for
// both the node name/FQDN and its local IP addresses. A cluster fronted by
// a VIP/ingress certificate that differs from the nodes' PVE-CA-signed
// certificates needs both CAs trusted; see WithCACert.
func WithNodeAffinity(opts ...NodeAffinityOption) Option {
	return func(c *ClientConfig) {
		cfg := &nodeAffinityConfig{}
		for _, opt := range opts {
			if opt != nil {
				opt(cfg)
			}
		}

		c.nodeAffinity = cfg
	}
}

// WithNodeEndpoints sets an explicit node name -> base URL map, the
// highest-precedence endpoint source. No I/O, no surprises — the
// recommended source for production.
func WithNodeEndpoints(m map[string]string) NodeAffinityOption {
	return func(c *nodeAffinityConfig) { c.endpoints = m }
}

// WithNodeEndpointTemplate sets a URL template with a {node} placeholder,
// e.g. "https://{node}.pve.example.com:8006", for clusters where node names
// resolve in DNS. Lower precedence than WithNodeEndpoints. Unlike the other
// sources it cannot tell a real node from a typo — it produces an endpoint
// for every name it is handed.
func WithNodeEndpointTemplate(tmpl string) NodeAffinityOption {
	return func(c *nodeAffinityConfig) { c.template = tmpl }
}

// WithNodeResolver sets a dynamic endpoint source — e.g. one backed by a
// Kubernetes informer — for callers with their own trustworthy mapping of
// nodes to endpoints. It is refreshed on its own goroutine, started when
// the client is built and stopped by Client.Close, every refresh interval
// (default 60s when refresh is non-positive). A failed refresh keeps the
// previous map and logs, rather than clearing it. Lower precedence than
// WithNodeEndpoints and WithNodeEndpointTemplate, higher than pool
// matching. r must be safe for concurrent use, and must not route its own
// traffic through the client it is configuring.
func WithNodeResolver(r NodeResolver, refresh time.Duration) NodeAffinityOption {
	return func(c *nodeAffinityConfig) {
		c.resolver = r
		c.resolverRefresh = refresh
	}
}

// WithSafePOSTFailover allows a POST against a pinned node to fail over to
// the delegate balancer, but only when the transport error proves the
// original request never reached the server — a connection refused, a DNS
// failure, or a TLS handshake failure — never once bytes are on the wire.
// Off by default (docs/node-lb.md §7.2): Proxmox POSTs create VMs, start
// migrations and fire tasks, so a retry that isn't provably safe could
// double-execute one. Without this, the first POST after a node dies fails
// outright with a transport error and the circuit breaker (§7.1) handles
// every request after it; this trades that one guaranteed failure for an
// automatic fallback, for callers who have judged the failure mode
// acceptable.
func WithSafePOSTFailover() NodeAffinityOption {
	return func(c *nodeAffinityConfig) { c.safePOSTFailover = true }
}

// WithNodeMaxFailures sets the number of consecutive failed requests to a
// node's endpoint before its circuit breaker opens (default 5). See
// docs/node-lb.md §7.1.
func WithNodeMaxFailures(n int) NodeAffinityOption {
	return func(c *nodeAffinityConfig) { c.maxFailures = n }
}

// WithNodeRecovery sets how long a node's endpoint stays circuit-broken
// before the breaker closes it again (default 120s). There is no half-open
// probe: the first real request routed after recovery is the probe.
func WithNodeRecovery(d time.Duration) NodeAffinityOption {
	return func(c *nodeAffinityConfig) { c.recovery = d }
}

// WithOnRoute sets a callback invoked for every balancer decision (direct or
// delegated), so a caller can turn routing into metrics. The default is a
// no-op, keeping the hot path free. fn must be safe for concurrent use.
func WithOnRoute(fn func(node, baseURL string, d RouteDecision)) NodeAffinityOption {
	return func(c *nodeAffinityConfig) { c.onRoute = fn }
}
