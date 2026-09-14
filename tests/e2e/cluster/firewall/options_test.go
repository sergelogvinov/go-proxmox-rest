//go:build e2e

// Package firewall_e2e exercises the cluster-wide firewall options
// (singleton config, GET/PUT /cluster/firewall/options) and refs
// (GET /cluster/firewall/refs) against a live Proxmox VE cluster.
package firewall_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster/firewall"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestFirewallOptionsGetUpdate verifies GET/PUT /cluster/firewall/options.
//
// Options is a cluster-wide singleton — there is exactly one, so this test
// does not create/delete anything. Because a real cluster's firewall policy
// lives here, the update is kept deliberately conservative: it only ever
// touches LogRatelimit (a cosmetic, non-disruptive rate-limiting knob) and
// leaves Enable/Ebtables/PolicyIn/PolicyOut/PolicyForward untouched by not
// sending them at all — Proxmox's PUT only applies the fields present in
// the request, so omitted fields are left exactly as they were. The
// original LogRatelimit is captured up front and restored in t.Cleanup.
func TestFirewallOptionsGetUpdate(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	fw := client.Cluster().Firewall()

	before, err := fw.Options().Get(ctx)
	e2e.RequireNoError(t, "options get (before)", err)

	originalLogRatelimit := before.LogRatelimit

	t.Cleanup(func() {
		if t.Failed() && !cfg.CleanupOnFailure {
			return
		}

		var restore firewall.Options
		if originalLogRatelimit == "" {
			// It was unset before the test: reset it to the default
			// instead of writing back an empty string (a pointer to ""
			// would explicitly set LogRatelimit to empty, which is not
			// the same as "unset" from Proxmox's point of view).
			restore.Delete = []string{"log_ratelimit"}
		} else {
			restore.LogRatelimit = originalLogRatelimit
		}

		cleanupCtx, cancel := e2e.CleanupContext()
		defer cancel()

		e2e.RetryCleanup(t, "restore log_ratelimit", func() error {
			return fw.Options().Update(cleanupCtx, &restore)
		})
	})

	// Pick a value that is guaranteed to differ from the original so the
	// change is actually observable.
	newRatelimit := "enable=1,rate=1/second,burst=5"
	if newRatelimit == originalLogRatelimit {
		newRatelimit = "enable=1,rate=1/second,burst=6"
	}

	err = fw.Options().Update(ctx, &firewall.Options{LogRatelimit: newRatelimit})
	e2e.RequireNoError(t, "options update", err)

	after, err := fw.Options().Get(ctx)
	e2e.RequireNoError(t, "options get (after)", err)

	if after.LogRatelimit != newRatelimit {
		t.Errorf("options: LogRatelimit = %q, want %q", after.LogRatelimit, newRatelimit)
	}

	// Every field we did not send must be unchanged.
	if after.PolicyIn != before.PolicyIn {
		t.Errorf("options: PolicyIn = %q, want unchanged %q", after.PolicyIn, before.PolicyIn)
	}
	if after.PolicyOut != before.PolicyOut {
		t.Errorf("options: PolicyOut = %q, want unchanged %q", after.PolicyOut, before.PolicyOut)
	}
	if after.PolicyForward != before.PolicyForward {
		t.Errorf("options: PolicyForward = %q, want unchanged %q", after.PolicyForward, before.PolicyForward)
	}
	if after.Enable != before.Enable {
		t.Errorf("options: Enable = %d, want unchanged %d", after.Enable, before.Enable)
	}
	if after.Ebtables != before.Ebtables {
		t.Errorf("options: Ebtables = %v, want unchanged %v", after.Ebtables, before.Ebtables)
	}
}

// TestFirewallOptionsValidation verifies that Update rejects a nil options
// argument client-side, without making any request.
func TestFirewallOptionsValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	err := client.Cluster().Firewall().Options().Update(ctx, nil)
	e2e.RequireError(t, "options update (nil)", err)
}

// TestFirewallRefs verifies GET /cluster/firewall/refs, unfiltered and
// filtered by type. It is read-only and asserts only that returned entries
// carry the requested Type; an empty result is acceptable on a fresh
// cluster with no aliases/IP sets defined.
func TestFirewallRefs(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	fw := client.Cluster().Firewall()

	_, err := fw.Refs(ctx, "")
	e2e.RequireNoError(t, "refs (unfiltered)", err)

	aliases, err := fw.Refs(ctx, firewall.RefTypeAlias)
	e2e.RequireNoError(t, "refs (type=alias)", err)
	for _, r := range aliases {
		if r.Type != "alias" {
			t.Errorf("refs(type=alias): Type = %q, want %q", r.Type, "alias")
		}
	}

	ipsets, err := fw.Refs(ctx, firewall.RefTypeIPSet)
	e2e.RequireNoError(t, "refs (type=ipset)", err)
	for _, r := range ipsets {
		if r.Type != "ipset" {
			t.Errorf("refs(type=ipset): Type = %q, want %q", r.Type, "ipset")
		}
	}
}
