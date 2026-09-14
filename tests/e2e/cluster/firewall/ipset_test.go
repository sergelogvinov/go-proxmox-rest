//go:build e2e

// Package firewall_e2e exercises the cluster firewall module (security
// groups, rules, aliases, IP sets and options) against a live Proxmox VE
// cluster.
package firewall_e2e

import (
	"slices"
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster/firewall"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestFirewallIPSetLifecycle runs the full CRUD lifecycle for a
// uniquely-named, empty IP set. IP sets have no singular Get endpoint, so
// verification after each write is done via List:
//
//	list → create → list → update → list → delete → list
func TestFirewallIPSetLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	is := client.Cluster().Firewall().IPSet()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)

	// Cleanup registry: delete the set even if the test fails mid-way,
	// unless the caller asked to keep resources for debugging. force=true
	// so cleanup works even if entries remain.
	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete firewall ipset", func() error {
				return is.Delete(cleanupCtx, name, true)
			})
		})
	}

	// 1. list — baseline: our set must not exist yet.
	listed, err := is.List(ctx)
	e2e.RequireNoError(t, "list firewall ipsets", err)
	if containsFirewallIPSet(listed, name) {
		t.Fatalf("list: firewall ipset %q already exists before create", name)
	}

	// 2. create — a uniquely-named, empty IP set.
	comment := "e2e lifecycle"
	err = is.Create(ctx, &firewall.IPSetOptions{Name: name, Comment: &comment})
	e2e.RequireNoError(t, "create firewall ipset", err)

	// 3. list — verify the create.
	listed, err = is.List(ctx)
	e2e.RequireNoError(t, "list firewall ipsets after create", err)
	set, ok := findFirewallIPSet(listed, name)
	if !ok {
		t.Fatalf("list after create: firewall ipset %q not found", name)
	}
	if set.Comment != comment {
		t.Errorf("list after create: Comment = %q, want %q", set.Comment, comment)
	}

	// 4. update — re-comment without renaming (opts.Name left empty, so
	// the set keeps its current name).
	newComment := "e2e updated"
	err = is.Update(ctx, name, &firewall.IPSetOptions{Comment: &newComment})
	e2e.RequireNoError(t, "update firewall ipset", err)

	// 5. list — verify the update; Name must be unchanged.
	listed, err = is.List(ctx)
	e2e.RequireNoError(t, "list firewall ipsets after update", err)
	set, ok = findFirewallIPSet(listed, name)
	if !ok {
		t.Fatalf("list after update: firewall ipset %q not found", name)
	}
	if set.Comment != newComment {
		t.Errorf("list after update: Comment = %q, want %q", set.Comment, newComment)
	}
	if set.Name != name {
		t.Errorf("list after update: Name = %q, want unchanged %q", set.Name, name)
	}

	// 6. delete — remove the (still empty) set; no force needed.
	err = is.Delete(ctx, name, false)
	e2e.RequireNoError(t, "delete firewall ipset", err)

	// 7. list — verify the delete.
	listed, err = is.List(ctx)
	e2e.RequireNoError(t, "list firewall ipsets after delete", err)
	if containsFirewallIPSet(listed, name) {
		t.Errorf("list after delete: firewall ipset %q still present", name)
	}
}

// TestFirewallIPSetRenameLifecycle verifies that Update can rename an IP
// set by setting IPSetOptions.Name to a new value.
func TestFirewallIPSetRenameLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	is := client.Cluster().Firewall().IPSet()
	ctx := t.Context()

	oldName := e2e.UniqueName(cfg.Prefix)
	newName := e2e.UniqueName(cfg.Prefix + "renamed-")

	// Cleanup registry: only one of oldName/newName will exist at any given
	// time; RetryCleanup treats the other's 404 as success.
	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete firewall ipset (old name)", func() error {
				return is.Delete(cleanupCtx, oldName, true)
			})
			e2e.RetryCleanup(t, "delete firewall ipset (new name)", func() error {
				return is.Delete(cleanupCtx, newName, true)
			})
		})
	}

	err := is.Create(ctx, &firewall.IPSetOptions{Name: oldName})
	e2e.RequireNoError(t, "create firewall ipset", err)

	err = is.Update(ctx, oldName, &firewall.IPSetOptions{Name: newName})
	e2e.RequireNoError(t, "rename firewall ipset", err)

	listed, err := is.List(ctx)
	e2e.RequireNoError(t, "list firewall ipsets after rename", err)
	if containsFirewallIPSet(listed, oldName) {
		t.Errorf("list after rename: old name %q still present", oldName)
	}
	if !containsFirewallIPSet(listed, newName) {
		t.Errorf("list after rename: new name %q not found", newName)
	}
}

// TestFirewallIPSetEntriesLifecycle runs the full CRUD lifecycle for a
// single CIDR member of an IP set. Unlike the set index itself, entries
// have a real Get, addressed by CIDR:
//
//	list → get(absent) → create → get → list → update → get → delete → list → get(absent)
func TestFirewallIPSetEntriesLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	is := client.Cluster().Firewall().IPSet()
	ctx := t.Context()

	setName := e2e.UniqueName(cfg.Prefix)

	// Cleanup registry: delete the parent set even if the test fails
	// mid-way; force=true handles any leftover entries.
	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete firewall ipset", func() error {
				return is.Delete(cleanupCtx, setName, true)
			})
		})
	}

	err := is.Create(ctx, &firewall.IPSetOptions{Name: setName})
	e2e.RequireNoError(t, "create firewall ipset", err)

	entries := is.Entries(setName)
	const cidr = "10.0.0.0/24"

	// 1. list — baseline: the new set has no members yet.
	listed, err := entries.List(ctx)
	e2e.RequireNoError(t, "list ipset entries", err)
	if len(listed) != 0 {
		t.Fatalf("list: ipset %q already has %d entr(ies) before create", setName, len(listed))
	}

	// 2. get (absent) — a non-existent member must return an error.
	_, err = entries.Get(ctx, cidr)
	e2e.RequireError(t, "get absent ipset entry", err)

	// 3. create — a single CIDR member.
	comment := "e2e lifecycle"
	err = entries.Create(ctx, &firewall.IPSetEntryOptions{CIDR: cidr, Comment: &comment})
	e2e.RequireNoError(t, "create ipset entry", err)

	// 4. get — verify the create.
	entry, err := entries.Get(ctx, cidr)
	e2e.RequireNoError(t, "get ipset entry after create", err)
	if entry.CIDR != cidr {
		t.Errorf("get: CIDR = %q, want %q", entry.CIDR, cidr)
	}
	if entry.Comment != comment {
		t.Errorf("get: Comment = %q, want %q", entry.Comment, comment)
	}

	// 5. list — verify the create is reflected in the list too.
	listed, err = entries.List(ctx)
	e2e.RequireNoError(t, "list ipset entries after create", err)
	if !containsFirewallIPSetEntry(listed, cidr) {
		t.Errorf("list after create: ipset entry %q not found", cidr)
	}

	// 6. update — mutate the comment and set NoMatch; CIDR must be resent.
	newComment := "e2e updated"
	err = entries.Update(ctx, cidr, &firewall.IPSetEntryOptions{
		CIDR:    cidr,
		Comment: &newComment,
		NoMatch: ipsetPtrBool(true),
	})
	e2e.RequireNoError(t, "update ipset entry", err)

	// 7. get — verify the update.
	entry, err = entries.Get(ctx, cidr)
	e2e.RequireNoError(t, "get ipset entry after update", err)
	if entry.Comment != newComment {
		t.Errorf("get after update: Comment = %q, want %q", entry.Comment, newComment)
	}
	if !entry.NoMatch {
		t.Errorf("get after update: NoMatch = %v, want true", entry.NoMatch)
	}

	// 8. delete — remove the member.
	err = entries.Delete(ctx, cidr, "")
	e2e.RequireNoError(t, "delete ipset entry", err)

	// 9. list — verify the delete.
	listed, err = entries.List(ctx)
	e2e.RequireNoError(t, "list ipset entries after delete", err)
	if containsFirewallIPSetEntry(listed, cidr) {
		t.Errorf("list after delete: ipset entry %q still present", cidr)
	}

	// get after delete must fail again.
	_, err = entries.Get(ctx, cidr)
	e2e.RequireError(t, "get ipset entry after delete", err)
}

// TestFirewallIPSetValidation verifies client-side validation of
// Create/Update options, none of which should reach the API.
func TestFirewallIPSetValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	is := client.Cluster().Firewall().IPSet()
	ctx := t.Context()

	err := is.Create(ctx, nil)
	e2e.RequireError(t, "create ipset with nil options", err)

	err = is.Create(ctx, &firewall.IPSetOptions{})
	e2e.RequireError(t, "create ipset with missing name", err)

	err = is.Entries("whatever").Create(ctx, nil)
	e2e.RequireError(t, "create ipset entry with nil options", err)

	err = is.Entries("whatever").Create(ctx, &firewall.IPSetEntryOptions{})
	e2e.RequireError(t, "create ipset entry with missing cidr", err)

	err = is.Entries("whatever").Update(ctx, "10.0.0.0/24", &firewall.IPSetEntryOptions{})
	e2e.RequireError(t, "update ipset entry with missing cidr", err)
}

// containsFirewallIPSet reports whether the slice contains an IP set with
// the given name.
func containsFirewallIPSet(list []firewall.IPSet, name string) bool {
	return slices.ContainsFunc(list, func(s firewall.IPSet) bool { return s.Name == name })
}

// findFirewallIPSet returns the IP set with the given name, if present.
func findFirewallIPSet(list []firewall.IPSet, name string) (firewall.IPSet, bool) {
	i := slices.IndexFunc(list, func(s firewall.IPSet) bool { return s.Name == name })
	if i < 0 {
		return firewall.IPSet{}, false
	}

	return list[i], true
}

// containsFirewallIPSetEntry reports whether the slice contains a member
// with the given CIDR.
func containsFirewallIPSetEntry(list []firewall.IPSetEntry, cidr string) bool {
	return slices.ContainsFunc(list, func(e firewall.IPSetEntry) bool { return e.CIDR == cidr })
}

// ipsetPtrBool returns a pointer to the given bool.
func ipsetPtrBool(b bool) *bool {
	return &b
}
