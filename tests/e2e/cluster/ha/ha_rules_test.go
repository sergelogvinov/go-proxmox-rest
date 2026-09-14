//go:build e2e

package cluster_e2e

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/sergelogvinov/go-proxmox-rest/cluster/ha"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestHARulesLifecycle runs the full CRUD lifecycle for a uniquely-named
// node-affinity HA rule:
//
//	list → get(absent) → create → get → update → get → delete → list
//
// The rule references a synthetic (non-existent) VMID: HA rules are plain
// configuration entries, so Proxmox does not require the referenced guest
// to exist.
func TestHARulesLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping HA rule tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	hr := client.Cluster().HA().Rules()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)
	resource := uniqueVMResource()

	// Cleanup registry: delete the rule even if the test fails mid-way,
	// unless the caller asked to keep resources for debugging.
	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			_ = hr.Delete(ctx, name)
		})
	}

	// 1. list — baseline: our rule must not exist yet.
	listed, err := hr.List(ctx, "", "")
	e2e.RequireNoError(t, "list ha rules", err)
	if containsHARule(listed, name) {
		t.Fatalf("list: ha rule %q already exists before create", name)
	}

	// 2. get (absent) — a non-existent rule must return an error.
	_, err = hr.Get(ctx, name)
	e2e.RequireError(t, "get absent ha rule", err)

	// 3. create — a uniquely-named node-affinity rule bound to the
	// configured node.
	comment := "e2e lifecycle"
	nodes := cfg.Node
	rule, err := hr.Create(ctx, &ha.RuleOptions{
		ID:        name,
		Type:      ha.RuleTypeNodeAffinity,
		Resources: []string{resource},
		Nodes:     &nodes,
		Comment:   &comment,
	})
	e2e.RequireNoError(t, "create ha rule", err)
	if rule.Rule != name {
		t.Errorf("create: Rule = %q, want %q", rule.Rule, name)
	}

	// 4. get — verify the create.
	rule, err = hr.Get(ctx, name)
	e2e.RequireNoError(t, "get ha rule after create", err)
	if rule.Rule != name {
		t.Errorf("get: Rule = %q, want %q", rule.Rule, name)
	}
	if rule.Type != string(ha.RuleTypeNodeAffinity) {
		t.Errorf("get: Type = %q, want %q", rule.Type, ha.RuleTypeNodeAffinity)
	}
	if !slices.Contains(rule.Resources, resource) {
		t.Errorf("get: Resources = %v, want to contain %q", rule.Resources, resource)
	}
	if rule.Nodes != nodes {
		t.Errorf("get: Nodes = %q, want %q", rule.Nodes, nodes)
	}
	if rule.Comment != comment {
		t.Errorf("get: Comment = %q, want %q", rule.Comment, comment)
	}

	// 5. update — mutate the comment and disable the rule. Type must be
	// resent (Proxmox requires it on every update).
	newComment := "e2e updated"
	disabled := true
	_, err = hr.Update(ctx, name, &ha.RuleOptions{
		Type:    ha.RuleTypeNodeAffinity,
		Comment: &newComment,
		Disable: &disabled,
	})
	e2e.RequireNoError(t, "update ha rule", err)

	// 6. get — verify the update; Resources/Nodes must be unchanged.
	rule, err = hr.Get(ctx, name)
	e2e.RequireNoError(t, "get ha rule after update", err)
	if rule.Comment != newComment {
		t.Errorf("get after update: Comment = %q, want %q", rule.Comment, newComment)
	}
	if rule.Disable == 0 {
		t.Errorf("get after update: Disable = %d, want non-zero", rule.Disable)
	}
	if !slices.Contains(rule.Resources, resource) {
		t.Errorf("get after update: Resources = %v, want to still contain %q", rule.Resources, resource)
	}

	// 7. delete — remove the rule.
	err = hr.Delete(ctx, name)
	e2e.RequireNoError(t, "delete ha rule", err)

	// 8. list — verify the delete.
	listed, err = hr.List(ctx, "", "")
	e2e.RequireNoError(t, "list ha rules after delete", err)
	if containsHARule(listed, name) {
		t.Errorf("list after delete: ha rule %q still present", name)
	}

	// get after delete must fail again.
	_, err = hr.Get(ctx, name)
	e2e.RequireError(t, "get ha rule after delete", err)
}

// TestHARulesListTypeFilter verifies that the type filter narrows List to
// the requested rule type.
func TestHARulesListTypeFilter(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping HA rule tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	hr := client.Cluster().HA().Rules()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)
	resource := uniqueVMResource()
	nodes := cfg.Node

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			_ = hr.Delete(ctx, name)
		})
	}

	_, err := hr.Create(ctx, &ha.RuleOptions{
		ID:        name,
		Type:      ha.RuleTypeNodeAffinity,
		Resources: []string{resource},
		Nodes:     &nodes,
	})
	e2e.RequireNoError(t, "create ha rule", err)

	// The filter must include our node-affinity rule.
	nodeAffinity, err := hr.List(ctx, ha.RuleTypeNodeAffinity, "")
	e2e.RequireNoError(t, "list ha rules filtered by type=node-affinity", err)
	if !containsHARule(nodeAffinity, name) {
		t.Errorf("list type=node-affinity: rule %q missing from result", name)
	}
	for _, r := range nodeAffinity {
		if r.Type != string(ha.RuleTypeNodeAffinity) {
			t.Errorf("list type=node-affinity: rule %q has Type %q, want %q", r.Rule, r.Type, ha.RuleTypeNodeAffinity)
		}
	}

	// A filter for the other type must not include our rule.
	resourceAffinity, err := hr.List(ctx, ha.RuleTypeResourceAffinity, "")
	e2e.RequireNoError(t, "list ha rules filtered by type=resource-affinity", err)
	if containsHARule(resourceAffinity, name) {
		t.Errorf("list type=resource-affinity: node-affinity rule %q unexpectedly present", name)
	}

	err = hr.Delete(ctx, name)
	e2e.RequireNoError(t, "delete ha rule", err)
}

// TestHARulesValidation verifies client-side validation of Create/Update
// options, none of which should reach the API.
func TestHARulesValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	hr := client.Cluster().HA().Rules()
	ctx := t.Context()

	_, err := hr.Create(ctx, nil)
	e2e.RequireError(t, "create with nil options", err)

	_, err = hr.Create(ctx, &ha.RuleOptions{
		Type:      ha.RuleTypeNodeAffinity,
		Resources: []string{"vm:100"},
	})
	e2e.RequireError(t, "create with missing id", err)

	_, err = hr.Create(ctx, &ha.RuleOptions{
		ID:        "does-not-matter",
		Resources: []string{"vm:100"},
	})
	e2e.RequireError(t, "create with missing type", err)

	_, err = hr.Create(ctx, &ha.RuleOptions{
		ID:   "does-not-matter",
		Type: ha.RuleTypeNodeAffinity,
	})
	e2e.RequireError(t, "create with missing resources", err)

	_, err = hr.Update(ctx, "does-not-matter", nil)
	e2e.RequireError(t, "update with nil options", err)

	_, err = hr.Update(ctx, "does-not-matter", &ha.RuleOptions{})
	e2e.RequireError(t, "update with missing type", err)
}

// uniqueVMResource returns a syntactically valid, unlikely-to-collide HA
// resource ID ("vm:<digits>") for a guest that does not need to exist.
func uniqueVMResource() string {
	return fmt.Sprintf("vm:%d", 100000+time.Now().UnixNano()%800000)
}

// containsHARule reports whether the slice contains an HA rule with the
// given ID.
func containsHARule(list []ha.Rule, id string) bool {
	return slices.ContainsFunc(list, func(r ha.Rule) bool { return r.Rule == id })
}
