//go:build e2e

package firewall_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc/firewall"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestFirewallRulesLifecycle runs the full CRUD lifecycle for a
// uniquely-commented firewall rule on a fresh fake guest's firewall
// config.
//
// Unlike ha.Rule, firewall.Rule has no caller-assigned identifier:
// Proxmox addresses a rule purely by its position (Pos) within the
// list, which it assigns automatically on Create and does not report
// back (Create's response is null data). So the rule is first created
// with a unique Comment, then located by re-listing and matching that
// comment to learn its Pos before Get/Update/Delete can be used.
func TestFirewallRulesLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc firewall tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	vmid := uniqueVMID()
	rr := client.Nodes().LXC().Firewall().Rules(cfg.Node, vmid)
	ctx := t.Context()

	comment := e2e.UniqueName(cfg.Prefix)

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete firewall rule", func() error {
				list, err := rr.List(cleanupCtx)
				if err != nil {
					return err
				}
				rule, ok := findFirewallRuleByComment(list, comment)
				if !ok {
					return nil
				}
				return rr.Delete(cleanupCtx, rule.Pos, "")
			})
		})
	}

	// 1. list — baseline: a fresh fake guest has no rules.
	listed, err := rr.List(ctx)
	e2e.RequireNoError(t, "list firewall rules", err)
	if _, ok := findFirewallRuleByComment(listed, comment); ok {
		t.Fatalf("list: firewall rule with comment %q already exists before create", comment)
	}

	// 2. create — a simple, non-destructive inbound accept rule.
	source := "10.0.0.0/8"
	proto := "tcp"
	dport := "8006"
	err = rr.Create(ctx, &firewall.RuleOptions{
		Type:    firewall.RuleTypeIn,
		Action:  string(firewall.PolicyAccept),
		Comment: &comment,
		Source:  &source,
		Proto:   &proto,
		DPort:   &dport,
	})
	e2e.RequireNoError(t, "create firewall rule", err)

	// 3. list — find the just-created rule by its unique comment and
	// capture its assigned position.
	listed, err = rr.List(ctx)
	e2e.RequireNoError(t, "list firewall rules after create", err)
	created, ok := findFirewallRuleByComment(listed, comment)
	if !ok {
		t.Fatalf("list after create: firewall rule with comment %q not found", comment)
	}
	pos := created.Pos

	// 4. get — verify the create.
	rule, err := rr.Get(ctx, pos)
	e2e.RequireNoError(t, "get firewall rule after create", err)
	if rule.Type != firewall.RuleTypeIn {
		t.Errorf("get: Type = %q, want %q", rule.Type, firewall.RuleTypeIn)
	}
	if rule.Action != string(firewall.PolicyAccept) {
		t.Errorf("get: Action = %q, want %q", rule.Action, firewall.PolicyAccept)
	}
	if rule.Comment != comment {
		t.Errorf("get: Comment = %q, want %q", rule.Comment, comment)
	}
	if rule.Source != source {
		t.Errorf("get: Source = %q, want %q", rule.Source, source)
	}
	if rule.Proto != proto {
		t.Errorf("get: Proto = %q, want %q", rule.Proto, proto)
	}
	if rule.DPort != dport {
		t.Errorf("get: DPort = %q, want %q", rule.DPort, dport)
	}

	// 5. update — mutate the comment and disable the rule. Type and
	// Action must be resent (Proxmox requires them on every update).
	newComment := comment + "-updated"
	enable := false
	err = rr.Update(ctx, pos, &firewall.RuleOptions{
		Type:    firewall.RuleTypeIn,
		Action:  string(firewall.PolicyAccept),
		Comment: &newComment,
		Enable:  &enable,
	})
	e2e.RequireNoError(t, "update firewall rule", err)

	// 6. get — verify the update; Source/Proto/DPort must be unchanged.
	rule, err = rr.Get(ctx, pos)
	e2e.RequireNoError(t, "get firewall rule after update", err)
	if rule.Comment != newComment {
		t.Errorf("get after update: Comment = %q, want %q", rule.Comment, newComment)
	}
	if rule.Enable != 0 {
		t.Errorf("get after update: Enable = %d, want 0 (disabled)", rule.Enable)
	}
	if rule.Source != source {
		t.Errorf("get after update: Source = %q, want to still be %q", rule.Source, source)
	}

	// 7. delete — remove the rule.
	err = rr.Delete(ctx, pos, "")
	e2e.RequireNoError(t, "delete firewall rule", err)

	// 8. list — verify the delete.
	listed, err = rr.List(ctx)
	e2e.RequireNoError(t, "list firewall rules after delete", err)
	if _, ok := findFirewallRuleByComment(listed, newComment); ok {
		t.Errorf("list after delete: firewall rule with comment %q still present", newComment)
	}
}

// TestFirewallRulesValidation verifies client-side validation of
// Create/Update options, none of which should reach the API.
func TestFirewallRulesValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc firewall tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	rr := client.Nodes().LXC().Firewall().Rules(cfg.Node, uniqueVMID())
	ctx := t.Context()

	err := rr.Create(ctx, nil)
	e2e.RequireError(t, "create with nil options", err)

	err = rr.Create(ctx, &firewall.RuleOptions{
		Action: string(firewall.PolicyAccept),
	})
	e2e.RequireError(t, "create with missing type", err)

	err = rr.Create(ctx, &firewall.RuleOptions{
		Type: firewall.RuleTypeIn,
	})
	e2e.RequireError(t, "create with missing action", err)

	err = rr.Update(ctx, 0, nil)
	e2e.RequireError(t, "update with nil options", err)

	err = rr.Update(ctx, 0, &firewall.RuleOptions{})
	e2e.RequireError(t, "update with missing type and action", err)
}

// findFirewallRuleByComment returns the first rule in list whose
// Comment matches comment.
func findFirewallRuleByComment(list []firewall.Rule, comment string) (firewall.Rule, bool) {
	for _, r := range list {
		if r.Comment == comment {
			return r, true
		}
	}

	return firewall.Rule{}, false
}
