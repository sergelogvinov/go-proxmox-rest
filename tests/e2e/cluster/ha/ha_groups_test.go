//go:build e2e

package cluster_e2e

import (
	"slices"
	"strings"
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster/ha"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestHAGroupsLifecycle runs the full CRUD lifecycle for a uniquely-named HA
// group:
//
//	list → get(absent) → create → get → update → get → delete → list
func TestHAGroupsLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping HA group tests")
	}

	ctx := t.Context()

	client := e2e.NewE2EClient(t, cfg)

	hg := client.Cluster().HA().Groups()

	// 1. list — baseline: our group must not exist yet. Proxmox VE 9 migrates
	// HA groups to HA rules once the whole cluster is upgraded, after which
	// every groups endpoint fails; probe before registering cleanup.
	listed, err := hg.List(ctx)
	if err != nil && strings.Contains(err.Error(), "migrated to rules") {
		t.Skip("HA groups have been migrated to rules; skipping HA group tests")
	}
	e2e.RequireNoError(t, "list ha groups", err)

	name := e2e.UniqueName(cfg.Prefix)

	// Cleanup registry: delete the group even if the test fails mid-way,
	// unless the caller asked to keep resources for debugging.
	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete ha group", func() error {
				return hg.Delete(cleanupCtx, name)
			})
		})
	}

	if containsHAGroup(listed, name) {
		t.Fatalf("list: ha group %q already exists before create", name)
	}

	// 2. get (absent) — a non-existent group must return an error.
	_, err = hg.Get(ctx, name)
	e2e.RequireError(t, "get absent ha group", err)

	// 3. create — a uniquely-named group bound to the configured node.
	comment := "e2e lifecycle"
	nodes := cfg.Node
	err = hg.Create(ctx, &ha.GroupOptions{
		ID:      name,
		Nodes:   &nodes,
		Comment: &comment,
	})
	e2e.RequireNoError(t, "create ha group", err)

	// 4. get — verify the create.
	group, err := hg.Get(ctx, name)
	e2e.RequireNoError(t, "get ha group after create", err)
	if group.Group != name {
		t.Errorf("get: Group = %q, want %q", group.Group, name)
	}
	if group.Nodes != nodes {
		t.Errorf("get: Nodes = %q, want %q", group.Nodes, nodes)
	}
	if group.Comment != comment {
		t.Errorf("get: Comment = %q, want %q", group.Comment, comment)
	}

	// 5. update — mutate the comment and enable restricted/nofailback.
	newComment := "e2e updated"
	trueVal := true
	err = hg.Update(ctx, name, &ha.GroupOptions{
		Comment:    &newComment,
		Restricted: &trueVal,
		Nofailback: &trueVal,
	})
	e2e.RequireNoError(t, "update ha group", err)

	// 6. get — verify the update; Nodes must be unchanged.
	group, err = hg.Get(ctx, name)
	e2e.RequireNoError(t, "get ha group after update", err)
	if group.Comment != newComment {
		t.Errorf("get after update: Comment = %q, want %q", group.Comment, newComment)
	}
	if group.Restricted == 0 {
		t.Errorf("get after update: Restricted = %d, want non-zero", group.Restricted)
	}
	if group.Nofailback == 0 {
		t.Errorf("get after update: Nofailback = %d, want non-zero", group.Nofailback)
	}
	if group.Nodes != nodes {
		t.Errorf("get after update: Nodes = %q, want unchanged %q", group.Nodes, nodes)
	}

	// 7. delete — remove the group.
	err = hg.Delete(ctx, name)
	e2e.RequireNoError(t, "delete ha group", err)

	// 8. list — verify the delete.
	listed, err = hg.List(ctx)
	e2e.RequireNoError(t, "list ha groups after delete", err)
	if containsHAGroup(listed, name) {
		t.Errorf("list after delete: ha group %q still present", name)
	}

	// get after delete must fail again.
	_, err = hg.Get(ctx, name)
	e2e.RequireError(t, "get ha group after delete", err)
}

// TestHAGroupsValidation verifies client-side validation of Create/Update
// options, none of which should reach the API.
func TestHAGroupsValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	hg := client.Cluster().HA().Groups()
	ctx := t.Context()

	err := hg.Create(ctx, nil)
	e2e.RequireError(t, "create with nil options", err)

	nodes := "does-not-matter"
	err = hg.Create(ctx, &ha.GroupOptions{Nodes: &nodes})
	e2e.RequireError(t, "create with missing id", err)

	err = hg.Create(ctx, &ha.GroupOptions{ID: "does-not-matter"})
	e2e.RequireError(t, "create with missing nodes", err)

	err = hg.Update(ctx, "does-not-matter", nil)
	e2e.RequireError(t, "update with nil options", err)
}

// containsHAGroup reports whether the slice contains an HA group with the
// given ID.
func containsHAGroup(list []ha.Group, id string) bool {
	return slices.ContainsFunc(list, func(g ha.Group) bool { return g.Group == id })
}
