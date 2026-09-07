//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest"
)

// E2EConfig holds the environment-driven configuration for the e2e suite.
type E2EConfig struct {
	// URL is the base API URL, e.g. https://pve:8006/api2/json
	URL string
	// TokenID / TokenSecret enable API token auth when set.
	TokenID     string
	TokenSecret string
	// Username / Password enable ticket auth when set.
	Username string
	Password string
	// Insecure skips TLS verification when true.
	Insecure bool
	// CACert is an optional path to a PEM CA bundle.
	CACert string
	// Node scopes node-dependent tests (unused by pools, kept for other modules).
	Node string
	// Prefix is prepended to all created resource names.
	Prefix string
	// Parallel enables t.Parallel() in module tests.
	Parallel bool
	// CleanupOnFailure deletes created resources even when a test fails.
	CleanupOnFailure bool
}

// LoadE2EConfig reads the PVE_E2E_* environment variables and applies
// defaults. It returns nil when the suite is not configured (no URL),
// which callers translate into a skip.
func LoadE2EConfig() *E2EConfig {
	cfg := &E2EConfig{
		URL:              os.Getenv("PVE_E2E_URL"),
		TokenID:          os.Getenv("PVE_E2E_TOKEN_ID"),
		TokenSecret:      os.Getenv("PVE_E2E_TOKEN_SECRET"),
		Username:         os.Getenv("PVE_E2E_USERNAME"),
		Password:         os.Getenv("PVE_E2E_PASSWORD"),
		Insecure:         os.Getenv("PVE_E2E_INSECURE") == "1",
		CACert:           os.Getenv("PVE_E2E_CA_CERT"),
		Node:             os.Getenv("PVE_E2E_NODE"),
		Prefix:           os.Getenv("PVE_E2E_PREFIX"),
		Parallel:         os.Getenv("PVE_E2E_PARALLEL") == "true",
		CleanupOnFailure: os.Getenv("PVE_E2E_CLEANUP_ON_FAILURE") != "false",
	}

	if cfg.URL == "" {
		return nil
	}

	if cfg.Prefix == "" {
		cfg.Prefix = "e2e-"
	}

	return cfg
}

// HasTokenAuth reports whether API token credentials are configured.
func (c *E2EConfig) HasTokenAuth() bool {
	return c.TokenID != "" && c.TokenSecret != ""
}

// HasPasswordAuth reports whether password/ticket credentials are configured.
func (c *E2EConfig) HasPasswordAuth() bool {
	return c.Username != "" && c.Password != ""
}

// NewE2EClient builds a proxmox.Client from the given config, performs a
// cheap smoke request to fail fast on bad credentials, and registers a
// cleanup hook that closes the client.
func NewE2EClient(t *testing.T, cfg *E2EConfig) *proxmox.Client {
	t.Helper()

	if cfg == nil {
		t.Skip("PVE_E2E_URL is not set; skipping e2e suite")
	}

	opts := []proxmox.Option{proxmox.WithURL(cfg.URL)}

	switch {
	case cfg.HasTokenAuth():
		opts = append(opts, proxmox.WithTokenAuth(cfg.TokenID, cfg.TokenSecret))
	case cfg.HasPasswordAuth():
		opts = append(opts, proxmox.WithPasswordAuth(cfg.Username, cfg.Password))
	default:
		t.Skip("neither PVE_E2E_TOKEN_ID nor PVE_E2E_USERNAME/PVE_E2E_PASSWORD is set; skipping e2e suite")
	}

	if cfg.Insecure {
		opts = append(opts, proxmox.WithInsecure(true))
	}
	if cfg.CACert != "" {
		opts = append(opts, proxmox.WithCACert(cfg.CACert))
	}

	client, err := proxmox.New(proxmox.ClientConfig{}, opts...)
	if err != nil {
		t.Fatalf("failed to build proxmox client: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
	})

	// Smoke check: fail fast with a clear message on connection or
	// credential problems instead of a confusing per-test failure.
	if _, err := client.Version(t.Context()); err != nil {
		t.Fatalf("e2e smoke check failed (check PVE_E2E_URL and credentials): %v", err)
	}

	return client
}

// MustConfig returns the loaded e2e config or skips the test when the
// suite is not configured.
func MustConfig(t *testing.T) *E2EConfig {
	t.Helper()

	cfg := LoadE2EConfig()
	if cfg == nil {
		t.Skip("PVE_E2E_URL is not set; skipping e2e suite")
	}

	return cfg
}

// String returns a human-readable summary of the config (without secrets).
func (c *E2EConfig) String() string {
	auth := "none"
	switch {
	case c.HasTokenAuth():
		auth = fmt.Sprintf("token(%s)", c.TokenID)
	case c.HasPasswordAuth():
		auth = fmt.Sprintf("password(%s)", c.Username)
	}

	return fmt.Sprintf("url=%s auth=%s insecure=%v prefix=%s", c.URL, auth, c.Insecure, c.Prefix)
}
