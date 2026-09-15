// Package proxmox provides a type-safe Go client for the Proxmox VE REST API,
// built on top of resty with a fluent resource-chaining API.
package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
	"github.com/sergelogvinov/go-proxmox-rest/nodes"
	"github.com/sergelogvinov/go-proxmox-rest/pools"
	"github.com/sergelogvinov/go-proxmox-rest/storage"
	"resty.dev/v3"
)

const (
	defaultTimeout       = 5 * time.Minute
	defaultRetryCount    = 3
	defaultRetryWaitTime = 500 * time.Millisecond
	defaultRetryMaxWait  = 30 * time.Second

	// sessionTTL is how long the client considers a ticket valid before
	// renewing it. Proxmox tickets expire after 2 hours; renew earlier to
	// avoid a request failing mid-flight.
	sessionTTL = 1 * time.Hour

	// defaultBasePath is the Proxmox API path prefix used when a load
	// balancer (which carries no path information) is configured.
	defaultBasePath = "/api2/json"
)

// ClientConfig holds the configuration for a Client. All fields are private
// and can only be set through functional options (see Option).
type ClientConfig struct {
	BaseURL     string
	Token       string
	TokenSecret string
	Username    string
	Password    string
	UserAgent   string
	Proxy       string

	Insecure bool
	CACert   string

	timeout          time.Duration
	retryCount       int
	retryWaitTime    time.Duration
	retryMaxWaitTime time.Duration

	lb resty.LoadBalancer

	// basePath is the API path prefix (e.g. /api2/json) extracted from
	// BaseURL or set by LB options. It is prepended to every request path
	// when a load balancer is used, since LB endpoints carry no path.
	basePath string

	logger resty.Logger
}

// ToRESTConfig returns a copy of the config, safe to mutate without affecting
// the original.
func (c ClientConfig) ToRESTConfig() ClientConfig {
	return ClientConfig{
		BaseURL:          c.BaseURL,
		Token:            c.Token,
		TokenSecret:      c.TokenSecret,
		Username:         c.Username,
		Password:         c.Password,
		UserAgent:        c.UserAgent,
		Proxy:            c.Proxy,
		Insecure:         c.Insecure,
		CACert:           c.CACert,
		timeout:          c.timeout,
		retryCount:       c.retryCount,
		retryWaitTime:    c.retryWaitTime,
		retryMaxWaitTime: c.retryMaxWaitTime,
		lb:               c.lb,
		basePath:         c.basePath,
		logger:           c.logger,
	}
}

// Client is the Proxmox VE API client.
type Client struct {
	cfg ClientConfig
	rc  *resty.Client

	sessionMux       sync.Mutex
	session          *Session
	sessionExpiresAt time.Time
}

// Session holds the ticket-based authentication data returned by
// POST /access/ticket.
type Session struct {
	Username            string `json:"username,omitempty" url:"username,omitempty"`
	Ticket              string `json:"ticket,omitempty" url:"ticket,omitempty"`
	CSRFPreventionToken string `json:"CSRFPreventionToken,omitempty" url:"CSRFPreventionToken,omitempty"`
	Cap                 any    `json:"cap,omitempty" url:"cap,omitempty"`
	ClusterName         string `json:"clustername,omitempty" url:"clustername,omitempty"`
}

// New builds a Client from the given config, applying the options on a copy
// (last one wins). Either a base URL or a load balancer must be provided,
// otherwise the default https://127.0.0.1:8006/api2/json is used.
func New(cfg ClientConfig, opts ...Option) (*Client, error) {
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	if cfg.timeout <= 0 {
		cfg.timeout = defaultTimeout
	}
	if cfg.retryCount < 0 {
		cfg.retryCount = defaultRetryCount
	}
	if cfg.retryWaitTime <= 0 {
		cfg.retryWaitTime = defaultRetryWaitTime
	}
	if cfg.retryMaxWaitTime <= 0 {
		cfg.retryMaxWaitTime = defaultRetryMaxWait
	}

	transportSettings := &resty.TransportSettings{
		IdleConnTimeout:     120 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DialerTimeout:       10 * time.Second,
	}

	rc := resty.NewWithTransportSettings(transportSettings)
	rc.SetTimeout(cfg.timeout)
	rc.SetRetryCount(cfg.retryCount)
	rc.SetRetryWaitTime(cfg.retryWaitTime)
	rc.SetRetryMaxWaitTime(cfg.retryMaxWaitTime)
	rc.AddRetryConditions(retryCondition)

	if cfg.lb != nil {
		rc.SetLoadBalancer(cfg.lb)
		// LB endpoints carry no path; default to the standard Proxmox
		// API prefix unless the user set one via WithBasePath/WithURL.
		if cfg.basePath == "" {
			cfg.basePath = defaultBasePath
		}
	} else {
		rc.SetBaseURL(cfg.BaseURL)
	}

	if cfg.UserAgent != "" {
		rc.SetHeader("User-Agent", cfg.UserAgent)
	}

	if cfg.Proxy != "" {
		rc.SetProxy(cfg.Proxy)
	}

	if cfg.logger != nil {
		rc.SetLogger(cfg.logger)
	}

	if cfg.CACert != "" {
		rc.SetRootCertificates(cfg.CACert)
	}

	if cfg.Insecure {
		tlsCfg := &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		}
		rc.SetTLSClientConfig(tlsCfg)
	}

	c := &Client{cfg: cfg, rc: rc}

	return c, nil
}

// ToRESTConfig returns a copy of the client's configuration suitable for use
// with the REST client. Modifications to the returned config do not affect
// the original client.
func (c *Client) ToRESTConfig() ClientConfig {
	return ClientConfig{
		BaseURL:          c.cfg.BaseURL,
		Token:            c.cfg.Token,
		TokenSecret:      c.cfg.TokenSecret,
		Username:         c.cfg.Username,
		Password:         c.cfg.Password,
		UserAgent:        c.cfg.UserAgent,
		Proxy:            c.cfg.Proxy,
		Insecure:         c.cfg.Insecure,
		CACert:           c.cfg.CACert,
		timeout:          c.cfg.timeout,
		retryCount:       c.cfg.retryCount,
		retryWaitTime:    c.cfg.retryWaitTime,
		retryMaxWaitTime: c.cfg.retryMaxWaitTime,
		lb:               c.cfg.lb,
		logger:           c.cfg.logger,
	}
}

// ensureSession performs ticket/password authentication lazily on the first
// request and renews the ticket before it expires. Proxmox tickets are valid
// for 2 hours; the client renews after sessionTTL to stay ahead of expiry.
// It is a no-op for token-authenticated clients.
func (c *Client) ensureSession(ctx context.Context) error {
	if c.cfg.Token != "" || c.cfg.Username == "" {
		return nil
	}

	c.sessionMux.Lock()
	defer c.sessionMux.Unlock()

	if c.session != nil && time.Now().Before(c.sessionExpiresAt) {
		return nil
	}

	// If a session exists but expired, try renewing the ticket first: the
	// Proxmox API accepts the previous ticket as the password on
	// POST /access/ticket, which avoids needing the original password and
	// keeps OTP/2FA-authenticated sessions alive.
	if c.session != nil && c.session.Ticket != "" {
		if err := c.ticket(ctx, c.session.Username, c.session.Ticket); err == nil {
			return nil
		}
	}

	return c.ticket(ctx, c.cfg.Username, c.cfg.Password)
}

// ticket exchanges credentials for a PVEAuthCookie via POST /access/ticket
// and stores the session on the client. The caller must hold sessionMux.
func (c *Client) ticket(ctx context.Context, username, password string) error {
	res, err := c.rc.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"username": username,
			"password": password,
		}).
		Post(c.path("/access/ticket"))
	if err != nil {
		return fmt.Errorf("ticket request: %w", err)
	}
	if res.StatusCode() != http.StatusOK {
		return newAPIError(res)
	}

	s := &Session{}
	if err := decodeInto(res.Bytes(), s); err != nil {
		return err
	}

	// The ticket arrives both in the JSON payload and as the PVEAuthCookie
	// set-cookie header; resty stores cookies in the client's cookie jar.
	if s.Ticket == "" {
		return fmt.Errorf("ticket: no ticket returned")
	}

	c.session = s
	c.sessionExpiresAt = time.Now().Add(sessionTTL)

	return nil
}

// do executes a typed request against the Proxmox API and decodes the
// envelope payload into out.
func (c *Client) do(ctx context.Context, method, path string, out any, params map[string]string) error {
	req := c.rc.R().SetContext(ctx)
	if len(params) > 0 {
		req.SetQueryParams(params)
	}

	return c.send(ctx, method, path, out, req)
}

// doValues is do's counterpart for query parameters that need more than one
// value under the same key — genuine Proxmox "array"-typed parameters,
// which the API expects as repeated same-named fields rather than a single
// comma-joined value (e.g. cluster/mapping's "map" property, whose entries
// are themselves comma-bearing property strings that a comma-join would
// corrupt).
func (c *Client) doValues(ctx context.Context, method, path string, out any, params url.Values) error {
	req := c.rc.R().SetContext(ctx)
	if len(params) > 0 {
		req.SetQueryParamsFromValues(params)
	}

	return c.send(ctx, method, path, out, req)
}

// send finishes preparing req (session + auth), executes it, and decodes
// the envelope payload into out. Shared by do and doValues, which differ
// only in how they populate req's query parameters.
func (c *Client) send(ctx context.Context, method, path string, out any, req *resty.Request) error {
	if err := c.ensureSession(ctx); err != nil {
		return err
	}

	if c.cfg.Token != "" {
		req.SetAuthScheme("PVEAPIToken")
		req.SetAuthToken(c.cfg.Token + "=" + c.cfg.TokenSecret)
	} else {
		c.sessionMux.Lock()
		if c.session != nil {
			req.SetHeader("Cookie", fmt.Sprintf("PVEAuthCookie=%s", c.session.Ticket))
			req.SetHeader("CSRFPreventionToken", c.session.CSRFPreventionToken)
		}
		c.sessionMux.Unlock()
	}

	res, err := req.Execute(method, c.path(path))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("%s %s: %w", method, path, err)
	}

	if res.StatusCode() >= http.StatusBadRequest {
		return newAPIError(res)
	}

	return decodeInto(res.Bytes(), out)
}

// path returns the full request path: the configured base path (e.g.
// /api2/json) joined with the resource path. When no load balancer is used,
// resty already applies the BaseURL path, so the resource path is returned
// as-is to avoid duplication.
func (c *Client) path(p string) string {
	if c.cfg.lb == nil || c.cfg.basePath == "" {
		return p
	}
	return c.cfg.basePath + "/" + strings.TrimPrefix(p, "/")
}

// Get performs a GET request and decodes the response envelope into out.
func (c *Client) Get(ctx context.Context, path string, out any, params map[string]string) error {
	return c.do(ctx, http.MethodGet, path, out, params)
}

// Create performs a POST request with form-encoded params.
func (c *Client) Create(ctx context.Context, path string, out any, params map[string]string) error {
	return c.do(ctx, http.MethodPost, path, out, params)
}

// Update performs a PUT request with form-encoded params.
func (c *Client) Update(ctx context.Context, path string, out any, params map[string]string) error {
	return c.do(ctx, http.MethodPut, path, out, params)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string, out any, params map[string]string) error {
	return c.do(ctx, http.MethodDelete, path, out, params)
}

// Patch performs a PATCH request with form-encoded params.
func (c *Client) Patch(ctx context.Context, path string, out any, params map[string]string) error {
	return c.do(ctx, http.MethodPatch, path, out, params)
}

// CreateValues performs a POST request with url.Values params, which
// (unlike Create's map[string]string) support multiple values under the
// same key. Used only by endpoints with a genuine Proxmox "array"-typed
// parameter (see doValues).
func (c *Client) CreateValues(ctx context.Context, path string, out any, params url.Values) error {
	return c.doValues(ctx, http.MethodPost, path, out, params)
}

// UpdateValues performs a PUT request with url.Values params, which (unlike
// Update's map[string]string) support multiple values under the same key.
// Used only by endpoints with a genuine Proxmox "array"-typed parameter
// (see doValues).
func (c *Client) UpdateValues(ctx context.Context, path string, out any, params url.Values) error {
	return c.doValues(ctx, http.MethodPut, path, out, params)
}

// Cluster returns a client for the cluster API section (/cluster).
func (c *Client) Cluster() *cluster.Client {
	return cluster.New(c)
}

// Nodes returns a client for the per-node API section (/nodes/{node}).
func (c *Client) Nodes() *nodes.Client {
	return nodes.New(c)
}

// Pools returns a client for the pools API section (/pools).
func (c *Client) Pools() *pools.Client {
	return pools.New(c)
}

// Storage returns a client for the storage API section (/storage).
func (c *Client) Storage() *storage.Client {
	return storage.New(c)
}

// Close releases the underlying resources held by the client.
func (c *Client) Close() error {
	c.rc.Close()
	return nil
}

// retryCondition retries only on explicit Proxmox "SlowDown" responses,
// 5xx server errors and connectivity errors.
func retryCondition(res *resty.Response, err error) bool {
	if err != nil {
		return true
	}
	if res == nil {
		return false
	}

	switch res.StatusCode() {
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusTooManyRequests:
		return true
	}

	var e struct {
		Errors map[string]string `json:"errors"`
	}
	if json.Unmarshal(res.Bytes(), &e) == nil {
		for _, msg := range e.Errors {
			if strings.Contains(msg, "SlowDown") {
				return true
			}
		}
	}
	return false
}

// envelope is the Proxmox response wrapper: { "data": ..., "errors": ... }.
type envelope struct {
	Data   json.RawMessage `json:"data"`
	Errors json.RawMessage `json:"errors"`
}

// decodeInto unwraps the { "data": ... } envelope and decodes the payload
// into out. A null data yields the zero value of out. Unknown fields are
// ignored so the API can grow without breaking this client. Decoding uses
// params.Decode rather than plain json.Unmarshal so that []string fields
// Proxmox sends as a comma-joined string (e.g. storage.Storage.Content)
// decode correctly; every other field behaves exactly as encoding/json
// would.
func decodeInto[T any](b []byte, out T) error {
	var env envelope
	if err := json.Unmarshal(b, &env); err != nil {
		return err
	}

	return params.Decode(env.Data, out)
}
