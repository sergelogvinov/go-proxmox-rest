//go:build e2e

package e2e

import (
	"fmt"
	"testing"
	"time"
)

// UniqueName returns a unique resource name prefixed with the configured
// prefix, e.g. "e2e-1731234567890123456". It is safe for parallel and
// repeated runs.
func UniqueName(prefix string) string {
	if prefix == "" {
		prefix = "e2e-"
	}

	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
}

// RequireNoError fails the test immediately when err is non-nil.
func RequireNoError(t *testing.T, what string, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: unexpected error: %v", what, err)
	}
}

// RequireError fails the test immediately when err is nil.
func RequireError(t *testing.T, what string, err error) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s: expected an error, got nil", what)
	}
}

// RequireNotFound fails the test when err is nil or is not a 404.
func RequireNotFound(t *testing.T, what string, err error) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s: expected a not-found error, got nil", what)
	}
}
