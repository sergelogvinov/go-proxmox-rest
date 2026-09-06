// Package proxmox provides a type-safe Go client for the Proxmox VE REST API,
// built on top of resty with a fluent resource-chaining API.
package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

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
)

// ClientConfig holds the configuration for a Client. All fields are private
// and can only be set through functional options (see Option).
type ClientConfig struct {
	baseURL   string
	token     string
	secret    string
	username  string
	password  string
	userAgent string
	proxy     string

	insecure bool
	caCert   string

	timeout          time.Duration
	retryCount       int
	retryWaitTime    time.Duration
	retryMaxWaitTime time.Duration
	lb               resty.LoadBalancer

	logger resty.Logger
}

// ToRESTConfig returns a copy of the config, safe to mutate without affecting
// the original.
func (c ClientConfig) ToRESTConfig() ClientConfig {
	return ClientConfig{
		baseURL:          c.baseURL,
		token:            c.token,
		secret:           c.secret,
		username:         c.username,
		password:         c.password,
		userAgent:        c.userAgent,
		proxy:            c.proxy,
		insecure:         c.insecure,
		caCert:           c.caCert,
		timeout:          c.timeout,
		retryCount:       c.retryCount,
		retryWaitTime:    c.retryWaitTime,
		retryMaxWaitTime: c.retryMaxWaitTime,
		lb:               c.lb,
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
	Username            string `json:"username,omitempty"`
	Ticket              string `json:"ticket,omitempty"`
	CSRFPreventionToken string `json:"CSRFPreventionToken,omitempty"`
	Cap                 any    `json:"cap,omitempty"`
	ClusterName         string `json:"clustername,omitempty"`
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

	rc := resty.New()
	rc.SetTimeout(cfg.timeout)
	rc.SetRetryCount(cfg.retryCount)
	rc.SetRetryWaitTime(cfg.retryWaitTime)
	rc.SetRetryMaxWaitTime(cfg.retryMaxWaitTime)
	rc.AddRetryConditions(retryCondition)

	if cfg.lb != nil {
		rc.SetLoadBalancer(cfg.lb)
	} else {
		rc.SetBaseURL(cfg.baseURL)
	}

	if cfg.userAgent != "" {
		rc.SetHeader("User-Agent", cfg.userAgent)
	}

	if cfg.proxy != "" {
		rc.SetProxy(cfg.proxy)
	}

	if cfg.logger != nil {
		rc.SetLogger(cfg.logger)
	}

	if cfg.caCert != "" {
		rc.SetRootCertificates(cfg.caCert)
	}

	if cfg.insecure {
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
		baseURL:          c.cfg.baseURL,
		token:            c.cfg.token,
		secret:           c.cfg.secret,
		username:         c.cfg.username,
		password:         c.cfg.password,
		userAgent:        c.cfg.userAgent,
		proxy:            c.cfg.proxy,
		insecure:         c.cfg.insecure,
		caCert:           c.cfg.caCert,
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
	if c.cfg.token != "" || c.cfg.username == "" {
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

	return c.ticket(ctx, c.cfg.username, c.cfg.password)
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
		Post("/access/ticket")
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

// Session returns the current authenticated session, or nil if the client
// has not yet authenticated (or is using an API token).
func (c *Client) Session() *Session {
	c.sessionMux.Lock()
	defer c.sessionMux.Unlock()
	return c.session
}

// RefreshTicket renews the existing PVE auth ticket by re-POSTing to
// /access/ticket with the current ticket as the password. This is the
// documented renewal mechanism that does not require the original password.
func (c *Client) RefreshTicket(ctx context.Context) error {
	if c.cfg.token != "" {
		return fmt.Errorf("refresh ticket: client uses API token auth")
	}

	c.sessionMux.Lock()
	defer c.sessionMux.Unlock()

	if c.session == nil || c.session.Ticket == "" {
		return fmt.Errorf("refresh ticket: no session")
	}

	return c.ticket(ctx, c.session.Username, c.session.Ticket)
}

// do executes a typed request against the Proxmox API and decodes the
// envelope payload into out.
func (c *Client) do(ctx context.Context, method, path string, out any, params map[string]string) error {
	if err := c.ensureSession(ctx); err != nil {
		return err
	}

	req := c.rc.R().SetContext(ctx)
	if len(params) > 0 {
		req.SetQueryParams(params)
	}

	if c.cfg.token != "" {
		req.SetAuthScheme("PVEAPIToken")
		req.SetAuthToken(c.cfg.token + "=" + c.cfg.secret)
	} else {
		c.sessionMux.Lock()
		if c.session != nil {
			req.SetHeader("Cookie", fmt.Sprintf("PVEAuthCookie=%s", c.session.Ticket))
			req.SetHeader("CSRFPreventionToken", c.session.CSRFPreventionToken)
		}
		c.sessionMux.Unlock()
	}

	res, err := req.Execute(method, path)
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
