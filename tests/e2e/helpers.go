//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sergelogvinov/go-proxmox-rest"
)

// maxUniqueNameLen is the longest name UniqueName returns, matching the
// tightest name-length limit among Proxmox resources exercised by the e2e
// suite (e.g. IP set/alias names).
const maxUniqueNameLen = 16

// UniqueName returns a unique resource name prefixed with the configured
// prefix, e.g. "e2e-7890123456", capped at maxUniqueNameLen characters. It
// is safe for parallel and repeated runs: when the timestamp suffix must be
// truncated to fit, the least significant (fastest-changing) digits are
// kept so short-lived name collisions stay unlikely.
func UniqueName(prefix string) string {
	if prefix == "" {
		prefix = "e2e-"
	}

	if len(prefix) >= maxUniqueNameLen {
		return prefix[:maxUniqueNameLen]
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	if room := maxUniqueNameLen - len(prefix); len(suffix) > room {
		suffix = suffix[len(suffix)-room:]
	}

	return prefix + suffix
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

// cleanupTimeout bounds how long a single t.Cleanup-registered API call
// (delete/restore) may take.
const cleanupTimeout = 30 * time.Second

// cleanupRetries / cleanupRetryDelay bound how many times, and how far
// apart, RetryCleanup retries a failed call before giving up.
const (
	cleanupRetries    = 3
	cleanupRetryDelay = 2 * time.Second
)

// CleanupContext returns a fresh, independent context (bounded by
// cleanupTimeout) and its cancel func, for use inside a t.Cleanup callback.
//
// t.Context() must never be reused there: per testing.T.Context's doc
// comment, it is already canceled by the time Cleanup functions run, so an
// API call made with it fails immediately with "context canceled" —
// silently leaving the resource the test created behind. Call this
// instead, with `defer cancel()`, at the top of every cleanup closure.
func CleanupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cleanupTimeout)
}

// alreadyGone reports whether err indicates that a cleanup call's target no
// longer exists, so RetryCleanup should stop without retrying or reporting
// a failure.
//
// Most Proxmox delete endpoints signal this with a proper 404
// (proxmox.IsNotFound). A few firewall endpoints instead respond 500 with a
// "no such <resource> '<name>'" message — e.g.
// DELETE .../firewall/groups/{group} on a group already removed by the
// test's own happy path. Matching that message shape is scoped to cleanup
// idempotency only; it does not change proxmox.IsNotFound's general,
// status-code-based semantics used elsewhere.
func alreadyGone(err error) bool {
	if proxmox.IsNotFound(err) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "no such")
}

// RetryCleanup calls fn up to cleanupRetries times, waiting
// cleanupRetryDelay between attempts, for use inside a t.Cleanup closure
// that deletes or restores a resource.
//
// An already-gone target (see alreadyGone) is treated as success without
// retrying: most cleanup calls race a redundant delete the test body's own
// happy path already issued, so that there is expected, not a failure to
// recover from. Any other error is retried; if it still fails after the
// last attempt, it is reported via t.Errorf so a genuinely leaked resource
// is never silently swallowed.
func RetryCleanup(t *testing.T, what string, fn func() error) {
	t.Helper()

	var err error

	for attempt := 1; attempt <= cleanupRetries; attempt++ {
		err = fn()
		if err == nil || alreadyGone(err) {
			return
		}
		if attempt < cleanupRetries {
			time.Sleep(cleanupRetryDelay)
		}
	}

	t.Errorf("cleanup: %s: %v", what, err)
}
