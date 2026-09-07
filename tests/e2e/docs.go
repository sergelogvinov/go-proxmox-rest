//go:build e2e

// Package e2e provides the shared connection helper for the end-to-end
// test suite. It is gated behind the `e2e` build tag and skipped
// entirely when PVE_E2E_URL is not set.
package e2e
