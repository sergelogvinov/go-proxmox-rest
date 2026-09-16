//go:build e2e

// Package firewall_e2e exercises the per-guest LXC firewall module
// (options, rules, aliases, IP sets, refs, log) against a live Proxmox
// VE cluster.
//
// Unlike every other nodes/lxc subresource, Proxmox's firewall config
// endpoints never check that the guest itself exists — PVE::Firewall's
// load_vmfw_conf just reads (or lazily creates) a "<vmid>.fw" file on
// disk, independent of whether "<vmid>.conf" exists. So it's safe to
// run the full CRUD lifecycle here against a syntactically valid but
// guaranteed-nonexistent VMID: there's no real guest whose firewall
// could be disrupted, only a throwaway config file that's created (and,
// via t.Cleanup, torn down again). Each test picks its own fresh fake
// VMID (uniqueVMID) rather than sharing one, so parallel subtests don't
// race on the same config file.
package firewall_e2e

import (
	"testing"
	"time"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc/firewall"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// uniqueVMID returns a syntactically valid VMID that's never allocated
// by this suite, distinct enough between rapid successive calls (e.g.
// parallel subtests) to avoid colliding on the same firewall config
// file.
func uniqueVMID() int {
	return 999000000 + int(time.Now().UnixNano()%900000)
}

// TestFirewallOptionsGetUpdate verifies GET/PUT
// /nodes/{node}/lxc/{vmid}/firewall/options against a fresh fake
// guest's (empty, default) firewall options.
func TestFirewallOptionsGetUpdate(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc firewall tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()
	vmid := uniqueVMID()

	oc := client.Nodes().LXC().Firewall().Options(cfg.Node, vmid)

	before, err := oc.Get(ctx)
	e2e.RequireNoError(t, "options get (before)", err)
	if before.Enable != 0 {
		t.Errorf("options (before): Enable = %d, want 0 on a fresh fake guest", before.Enable)
	}

	err = oc.Update(ctx, &firewall.Options{Enable: 1, MacFilter: true})
	e2e.RequireNoError(t, "options update", err)

	after, err := oc.Get(ctx)
	e2e.RequireNoError(t, "options get (after)", err)
	if after.Enable != 1 {
		t.Errorf("options (after): Enable = %d, want 1", after.Enable)
	}
	if !after.MacFilter {
		t.Errorf("options (after): MacFilter = false, want true")
	}
}

// TestFirewallOptionsValidation verifies that Update rejects a nil
// options argument client-side, without making any request.
func TestFirewallOptionsValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc firewall tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	err := client.Nodes().LXC().Firewall().Options(cfg.Node, uniqueVMID()).Update(ctx, nil)
	e2e.RequireError(t, "options update (nil)", err)
}

// TestFirewallRefs verifies GET
// /nodes/{node}/lxc/{vmid}/firewall/refs, unfiltered and filtered by
// type, against a fresh fake guest. Read-only; an empty result is
// expected since no aliases/IP sets are defined anywhere relevant to
// it.
func TestFirewallRefs(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc firewall tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	fw := client.Nodes().LXC().Firewall()
	ctx := t.Context()
	vmid := uniqueVMID()

	_, err := fw.Refs(ctx, cfg.Node, vmid, "")
	e2e.RequireNoError(t, "refs (unfiltered)", err)

	aliases, err := fw.Refs(ctx, cfg.Node, vmid, firewall.RefTypeAlias)
	e2e.RequireNoError(t, "refs (type=alias)", err)
	for _, r := range aliases {
		if r.Type != "alias" {
			t.Errorf("refs(type=alias): Type = %q, want %q", r.Type, "alias")
		}
	}

	ipsets, err := fw.Refs(ctx, cfg.Node, vmid, firewall.RefTypeIPSet)
	e2e.RequireNoError(t, "refs (type=ipset)", err)
	for _, r := range ipsets {
		if r.Type != "ipset" {
			t.Errorf("refs(type=ipset): Type = %q, want %q", r.Type, "ipset")
		}
	}
}

// TestFirewallLog verifies GET /nodes/{node}/lxc/{vmid}/firewall/log
// decodes without error against a fresh fake guest. Read-only; an
// empty result is expected since nothing has ever logged against this
// VMID.
func TestFirewallLog(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc firewall tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes().LXC().Firewall().Log(ctx, cfg.Node, uniqueVMID(), &firewall.LogOptions{Limit: 5})
	e2e.RequireNoError(t, "firewall log", err)
}
