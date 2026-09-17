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
// round-robin style. The endpoints ignore the API path prefix
// To redefine the default API path, use WithBasePath
func WithRoundRobin(urls ...string) Option {
	return func(c *ClientConfig) {
		if lb, err := resty.NewRoundRobin(urls...); err == nil {
			c.lb = lb
			c.basePath = defaultBasePath
		}
	}
}

// WithWeightedRoundRobin distributes requests across the given hosts
// proportionally to their weights. Since hosts carry no path, requests are
// prefixed with the default API path (/api2/json)
// To redefine the default API path, use WithBasePath
func WithWeightedRoundRobin(hosts ...*resty.Host) Option {
	return func(c *ClientConfig) {
		if lb, err := resty.NewWeightedRoundRobin(0, hosts...); err == nil {
			c.lb = lb
			c.basePath = defaultBasePath
		}
	}
}

// WithSRVWeightedRoundRobin resolves SRV records and load balances across
// the discovered endpoints. Since SRV records carry no path, requests are
// prefixed with the default API path (/api2/json)
// To redefine the default API path, use WithBasePath
func WithSRVWeightedRoundRobin(service, proto, domain, scheme string) Option {
	return func(c *ClientConfig) {
		if lb, err := resty.NewSRVWeightedRoundRobin(service, proto, domain, scheme); err == nil {
			c.lb = lb
			c.basePath = defaultBasePath
		}
	}
}

// WithLoadBalancer sets a custom resty.LoadBalancer implementation,
// overriding the single base URL.
func WithLoadBalancer(lb resty.LoadBalancer) Option {
	return func(c *ClientConfig) { c.lb = lb }
}
