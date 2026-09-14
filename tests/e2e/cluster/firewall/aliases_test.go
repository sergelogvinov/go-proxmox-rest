//go:build e2e

// Package firewall_e2e exercises the full CRUD lifecycle of the cluster
// firewall aliases module against a live Proxmox VE cluster:
//
//	list → get(absent) → create → get → update → get → delete → list
package firewall_e2e

import (
	"slices"
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster/firewall"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestFirewallAliasesLifecycle runs the full CRUD lifecycle for a
// uniquely-named cluster-wide firewall alias.
func TestFirewallAliasesLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ac := client.Cluster().Firewall().Aliases()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)

	// Cleanup registry: delete the alias even if the test fails mid-way,
	// unless the caller asked to keep resources for debugging.
	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete alias", func() error {
				return ac.Delete(cleanupCtx, name, "")
			})
		})
	}

	// 1. list — baseline: our alias must not exist yet.
	listed, err := ac.List(ctx)
	e2e.RequireNoError(t, "list aliases", err)
	if containsFirewallAlias(listed, name) {
		t.Fatalf("list: alias %q already exists before create", name)
	}

	// 2. get (absent) — a non-existent alias must return an error.
	_, err = ac.Get(ctx, name)
	e2e.RequireError(t, "get absent alias", err)

	// 3. create — a uniquely-named alias with a CIDR and comment.
	comment := "e2e lifecycle"
	err = ac.Create(ctx, &firewall.AliasOptions{Name: name, CIDR: "10.0.0.0/24", Comment: &comment})
	e2e.RequireNoError(t, "create alias", err)

	// 4. get — verify the create.
	alias, err := ac.Get(ctx, name)
	e2e.RequireNoError(t, "get alias after create", err)
	if alias.Name != name {
		t.Errorf("get: Name = %q, want %q", alias.Name, name)
	}
	if alias.CIDR != "10.0.0.0/24" {
		t.Errorf("get: CIDR = %q, want %q", alias.CIDR, "10.0.0.0/24")
	}
	if alias.Comment != comment {
		t.Errorf("get: Comment = %q, want %q", alias.Comment, comment)
	}

	// 5. update — mutate the CIDR and comment. Proxmox requires CIDR to be
	// resent on every update, even unchanged.
	newComment := "e2e updated"
	err = ac.Update(ctx, name, &firewall.AliasOptions{CIDR: "10.0.1.0/24", Comment: &newComment})
	e2e.RequireNoError(t, "update alias", err)

	// 6. get — verify the update.
	alias, err = ac.Get(ctx, name)
	e2e.RequireNoError(t, "get alias after update", err)
	if alias.CIDR != "10.0.1.0/24" {
		t.Errorf("get after update: CIDR = %q, want %q", alias.CIDR, "10.0.1.0/24")
	}
	if alias.Comment != newComment {
		t.Errorf("get after update: Comment = %q, want %q", alias.Comment, newComment)
	}

	// 7. delete — remove the alias.
	err = ac.Delete(ctx, name, "")
	e2e.RequireNoError(t, "delete alias", err)

	// 8. list — verify the delete.
	listed, err = ac.List(ctx)
	e2e.RequireNoError(t, "list aliases after delete", err)
	if containsFirewallAlias(listed, name) {
		t.Errorf("list after delete: alias %q still present", name)
	}

	// get after delete must fail again.
	_, err = ac.Get(ctx, name)
	e2e.RequireError(t, "get alias after delete", err)
}

// TestFirewallAliasesRenameLifecycle verifies that AliasOptions.Rename
// changes an existing alias's name on Update.
func TestFirewallAliasesRenameLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ac := client.Cluster().Firewall().Aliases()
	ctx := t.Context()

	oldName := e2e.UniqueName(cfg.Prefix)
	newName := e2e.UniqueName(cfg.Prefix + "renamed-")

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			// Only one of the two names exists by the end of the test;
			// RetryCleanup treats the other's 404 as success.
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete alias (old name)", func() error {
				return ac.Delete(cleanupCtx, oldName, "")
			})
			e2e.RetryCleanup(t, "delete alias (new name)", func() error {
				return ac.Delete(cleanupCtx, newName, "")
			})
		})
	}

	err := ac.Create(ctx, &firewall.AliasOptions{Name: oldName, CIDR: "10.0.0.0/24"})
	e2e.RequireNoError(t, "create alias", err)

	err = ac.Update(ctx, oldName, &firewall.AliasOptions{CIDR: "10.0.0.0/24", Rename: &newName})
	e2e.RequireNoError(t, "rename alias", err)

	_, err = ac.Get(ctx, newName)
	e2e.RequireNoError(t, "get alias under new name", err)

	_, err = ac.Get(ctx, oldName)
	e2e.RequireError(t, "get alias under old name", err)
}

// TestFirewallAliasesValidation verifies client-side validation of nil and
// incomplete options, which never reach the server.
func TestFirewallAliasesValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ac := client.Cluster().Firewall().Aliases()
	ctx := t.Context()

	err := ac.Create(ctx, nil)
	e2e.RequireError(t, "create with nil options", err)

	err = ac.Create(ctx, &firewall.AliasOptions{CIDR: "10.0.0.0/24"})
	e2e.RequireError(t, "create with missing name", err)

	err = ac.Create(ctx, &firewall.AliasOptions{Name: "x"})
	e2e.RequireError(t, "create with missing cidr", err)

	err = ac.Update(ctx, "x", nil)
	e2e.RequireError(t, "update with nil options", err)

	err = ac.Update(ctx, "x", &firewall.AliasOptions{})
	e2e.RequireError(t, "update with missing cidr", err)
}

// containsFirewallAlias reports whether the slice contains an alias with
// the given name.
func containsFirewallAlias(list []firewall.Alias, name string) bool {
	return slices.ContainsFunc(list, func(a firewall.Alias) bool { return a.Name == name })
}
