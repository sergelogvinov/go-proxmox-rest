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

// TestFirewallGroupsLifecycle runs the full CRUD lifecycle for a
// uniquely-named, empty security group. Groups have no singular Get
// endpoint, so verification after each write is done via List:
//
//	list → create → list → update → list → delete → list
func TestFirewallGroupsLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	gr := client.Cluster().Firewall().Groups()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)

	// Cleanup registry: delete the group even if the test fails mid-way,
	// unless the caller asked to keep resources for debugging.
	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete firewall group", func() error {
				return gr.Delete(cleanupCtx, name)
			})
		})
	}

	// 1. list — baseline: our group must not exist yet.
	listed, err := gr.List(ctx)
	e2e.RequireNoError(t, "list firewall groups", err)
	if containsFirewallGroup(listed, name) {
		t.Fatalf("list: firewall group %q already exists before create", name)
	}

	// 2. create — a uniquely-named, empty security group.
	comment := "e2e lifecycle"
	err = gr.Create(ctx, &firewall.GroupOptions{Name: name, Comment: &comment})
	e2e.RequireNoError(t, "create firewall group", err)

	// 3. list — verify the create.
	listed, err = gr.List(ctx)
	e2e.RequireNoError(t, "list firewall groups after create", err)
	group, ok := findFirewallGroup(listed, name)
	if !ok {
		t.Fatalf("list after create: firewall group %q not found", name)
	}
	if group.Comment != comment {
		t.Errorf("list after create: Comment = %q, want %q", group.Comment, comment)
	}

	// 4. update — re-comment without renaming (opts.Name left empty, so
	// the group keeps its current name).
	newComment := "e2e updated"
	err = gr.Update(ctx, name, &firewall.GroupOptions{Comment: &newComment})
	e2e.RequireNoError(t, "update firewall group", err)

	// 5. list — verify the update; Name must be unchanged.
	listed, err = gr.List(ctx)
	e2e.RequireNoError(t, "list firewall groups after update", err)
	group, ok = findFirewallGroup(listed, name)
	if !ok {
		t.Fatalf("list after update: firewall group %q not found", name)
	}
	if group.Comment != newComment {
		t.Errorf("list after update: Comment = %q, want %q", group.Comment, newComment)
	}
	if group.Name != name {
		t.Errorf("list after update: Name = %q, want unchanged %q", group.Name, name)
	}

	// 6. delete — remove the (still empty) group.
	err = gr.Delete(ctx, name)
	e2e.RequireNoError(t, "delete firewall group", err)

	// 7. list — verify the delete.
	listed, err = gr.List(ctx)
	e2e.RequireNoError(t, "list firewall groups after delete", err)
	if containsFirewallGroup(listed, name) {
		t.Errorf("list after delete: firewall group %q still present", name)
	}
}

// TestFirewallGroupsRenameLifecycle verifies that Update can rename a
// security group by setting GroupOptions.Name to a new value.
func TestFirewallGroupsRenameLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	gr := client.Cluster().Firewall().Groups()
	ctx := t.Context()

	oldName := e2e.UniqueName(cfg.Prefix)
	newName := e2e.UniqueName(cfg.Prefix + "renamed-")

	// Cleanup registry: only one of oldName/newName will exist at any given
	// time; RetryCleanup treats the other's 404 as success.
	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete firewall group (old name)", func() error {
				return gr.Delete(cleanupCtx, oldName)
			})
			e2e.RetryCleanup(t, "delete firewall group (new name)", func() error {
				return gr.Delete(cleanupCtx, newName)
			})
		})
	}

	err := gr.Create(ctx, &firewall.GroupOptions{Name: oldName})
	e2e.RequireNoError(t, "create firewall group", err)

	err = gr.Update(ctx, oldName, &firewall.GroupOptions{Name: newName})
	e2e.RequireNoError(t, "rename firewall group", err)

	listed, err := gr.List(ctx)
	e2e.RequireNoError(t, "list firewall groups after rename", err)
	if containsFirewallGroup(listed, oldName) {
		t.Errorf("list after rename: old name %q still present", oldName)
	}
	if !containsFirewallGroup(listed, newName) {
		t.Errorf("list after rename: new name %q not found", newName)
	}
}

// TestFirewallGroupRulesLifecycle runs the full CRUD lifecycle for a rule
// nested inside a security group. The group is created empty, a rule is
// added/verified/updated/removed, and only then is the (again empty) group
// deleted — Proxmox requires a group to be empty before it can be removed.
func TestFirewallGroupRulesLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	groups := client.Cluster().Firewall().Groups()
	ctx := t.Context()

	groupName := e2e.UniqueName(cfg.Prefix)
	gr := groups.Rules(groupName)

	var pos int
	havePos := false

	// Cleanup registry: remove the rule (if it still exists) before the
	// group, since Proxmox requires an empty group to delete it.
	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			if havePos {
				e2e.RetryCleanup(t, "delete group rule", func() error {
					return gr.Delete(cleanupCtx, pos, "")
				})
			}
			e2e.RetryCleanup(t, "delete firewall group", func() error {
				return groups.Delete(cleanupCtx, groupName)
			})
		})
	}

	err := groups.Create(ctx, &firewall.GroupOptions{Name: groupName})
	e2e.RequireNoError(t, "create firewall group", err)

	// 1. list — baseline: the new group has no rules yet.
	listed, err := gr.List(ctx)
	e2e.RequireNoError(t, "list group rules", err)
	if len(listed) != 0 {
		t.Fatalf("list: group %q already has %d rule(s) before create", groupName, len(listed))
	}

	// 2. create — a uniquely-commented inbound accept rule for SSH.
	comment := e2e.UniqueName(cfg.Prefix + "rule-")
	err = gr.Create(ctx, &firewall.RuleOptions{
		Type:    firewall.RuleTypeIn,
		Action:  string(firewall.PolicyAccept),
		Comment: &comment,
		Proto:   groupPtrString("tcp"),
		DPort:   groupPtrString("22"),
	})
	e2e.RequireNoError(t, "create group rule", err)

	// 3. list — find the new rule by its unique comment (Proxmox does not
	// report the position a created rule was assigned).
	listed, err = gr.List(ctx)
	e2e.RequireNoError(t, "list group rules after create", err)
	rule, ok := findFirewallGroupRuleByComment(listed, comment)
	if !ok {
		t.Fatalf("list after create: rule with comment %q not found", comment)
	}
	pos = rule.Pos
	havePos = true

	// 4. get — verify the create.
	got, err := gr.Get(ctx, pos)
	e2e.RequireNoError(t, "get group rule after create", err)
	if got.Type != firewall.RuleTypeIn {
		t.Errorf("get: Type = %q, want %q", got.Type, firewall.RuleTypeIn)
	}
	if got.Action != string(firewall.PolicyAccept) {
		t.Errorf("get: Action = %q, want %q", got.Action, firewall.PolicyAccept)
	}
	if got.Comment != comment {
		t.Errorf("get: Comment = %q, want %q", got.Comment, comment)
	}

	// 5. update — mutate the comment; Type/Action must be resent.
	newComment := "e2e updated"
	err = gr.Update(ctx, pos, &firewall.RuleOptions{
		Type:    firewall.RuleTypeIn,
		Action:  string(firewall.PolicyAccept),
		Comment: &newComment,
	})
	e2e.RequireNoError(t, "update group rule", err)

	// 6. get — verify the update.
	got, err = gr.Get(ctx, pos)
	e2e.RequireNoError(t, "get group rule after update", err)
	if got.Comment != newComment {
		t.Errorf("get after update: Comment = %q, want %q", got.Comment, newComment)
	}

	// 7. delete — remove the rule.
	err = gr.Delete(ctx, pos, "")
	e2e.RequireNoError(t, "delete group rule", err)
	havePos = false

	// 8. delete the now-empty group.
	err = groups.Delete(ctx, groupName)
	e2e.RequireNoError(t, "delete firewall group", err)
}

// TestFirewallGroupsValidation verifies client-side validation of
// Create/Update options, none of which should reach the API.
func TestFirewallGroupsValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	gr := client.Cluster().Firewall().Groups()
	ctx := t.Context()

	err := gr.Create(ctx, nil)
	e2e.RequireError(t, "create with nil options", err)

	err = gr.Create(ctx, &firewall.GroupOptions{})
	e2e.RequireError(t, "create with missing name", err)

	err = gr.Update(ctx, "x", nil)
	e2e.RequireError(t, "update with nil options", err)
}

// containsFirewallGroup reports whether the slice contains a security
// group with the given name.
func containsFirewallGroup(list []firewall.Group, name string) bool {
	return slices.ContainsFunc(list, func(g firewall.Group) bool { return g.Name == name })
}

// findFirewallGroup returns the security group with the given name, if
// present.
func findFirewallGroup(list []firewall.Group, name string) (firewall.Group, bool) {
	i := slices.IndexFunc(list, func(g firewall.Group) bool { return g.Name == name })
	if i < 0 {
		return firewall.Group{}, false
	}

	return list[i], true
}

// findFirewallGroupRuleByComment returns the rule with the given comment,
// if present. Used to locate a just-created rule, since Proxmox does not
// report the position a created rule was assigned.
func findFirewallGroupRuleByComment(list []firewall.Rule, comment string) (firewall.Rule, bool) {
	i := slices.IndexFunc(list, func(r firewall.Rule) bool { return r.Comment == comment })
	if i < 0 {
		return firewall.Rule{}, false
	}

	return list[i], true
}

// groupPtrString returns a pointer to the given string.
func groupPtrString(s string) *string {
	return &s
}
