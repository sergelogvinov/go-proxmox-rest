package proxmox

import (
	"time"

	"resty.dev/v3"
)

// Option mutates a ClientConfig. Options are the only way to populate the
// private config fields; the last applied option wins.
type Option func(*ClientConfig)

// WithBaseURL sets the API endpoint, e.g. https://10.0.0.1:8006/api2/json.
// A bare host is normalized to https://<host>:8006/api2/json.
func WithBaseURL(u string) Option {
	return func(c *ClientConfig) { c.baseURL = u }
}

// WithTokenAuth enables API token authentication.
// tokenID has the form user@realm!tokenid.
func WithTokenAuth(tokenID, secret string) Option {
	return func(c *ClientConfig) {
		c.token = tokenID
		c.secret = secret
	}
}

// WithPasswordAuth enables ticket/password authentication. The realm is
// included in the username (e.g., "root@pam").
func WithPasswordAuth(username, password string) Option {
	return func(c *ClientConfig) {
		c.username = username
		c.password = password
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
	return func(c *ClientConfig) { c.userAgent = ua }
}

// WithInsecure controls whether TLS certificate verification is skipped.
func WithInsecure(skip bool) Option {
	return func(c *ClientConfig) { c.insecure = skip }
}

// WithCACert sets the path to a PEM CA bundle to trust.
func WithCACert(path string) Option {
	return func(c *ClientConfig) { c.caCert = path }
}

// WithProxy sets an optional proxy URL.
func WithProxy(p string) Option {
	return func(c *ClientConfig) { c.proxy = p }
}

// WithLogger sets the resty logger.
func WithLogger(l resty.Logger) Option {
	return func(c *ClientConfig) { c.logger = l }
}

// WithRoundRobin distributes requests across the given API endpoints
// round-robin style.
func WithRoundRobin(urls ...string) Option {
	return func(c *ClientConfig) {
		if lb, err := resty.NewRoundRobin(urls...); err == nil {
			c.lb = lb
		}
	}
}

// WithWeightedRoundRobin distributes requests across the given hosts
// proportionally to their weights.
func WithWeightedRoundRobin(hosts ...*resty.Host) Option {
	return func(c *ClientConfig) {
		if lb, err := resty.NewWeightedRoundRobin(0, hosts...); err == nil {
			c.lb = lb
		}
	}
}

// WithSRVWeightedRoundRobin resolves SRV records and load balances across
// the discovered endpoints.
func WithSRVWeightedRoundRobin(service, proto, domain, scheme string) Option {
	return func(c *ClientConfig) {
		if lb, err := resty.NewSRVWeightedRoundRobin(service, proto, domain, scheme); err == nil {
			c.lb = lb
		}
	}
}

// WithLoadBalancer sets a custom resty.LoadBalancer implementation,
// overriding the single base URL.
func WithLoadBalancer(lb resty.LoadBalancer) Option {
	return func(c *ClientConfig) { c.lb = lb }
}
