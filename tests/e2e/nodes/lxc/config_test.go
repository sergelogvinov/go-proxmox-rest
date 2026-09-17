//go:build e2e

package lxc_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestLXCConfigAbsent verifies that GET
// /nodes/{node}/lxc/{vmid}/config on a nonexistent container returns an
// error.
func TestLXCConfigAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc config tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes(cfg.Node).LXC().Config(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "config of absent container", err)
}

// TestLXCConfigOpportunistic decodes a real container's configuration,
// if cfg.Node happens to host one, verifying the numerically-indexed
// families (net0, mp0, ...) decode without error against production
// data rather than only synthetic fixtures. It never writes anything —
// Config is read-only.
func TestLXCConfigOpportunistic(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc config tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	resources, err := client.Cluster().Resources().Get(ctx, cluster.ResourceTypeVM)
	e2e.RequireNoError(t, "list cluster vm resources", err)

	var vmid int
	for _, r := range resources {
		if r.Node == cfg.Node && r.Type == "lxc" {
			vmid = r.VMID
			break
		}
	}
	if vmid == 0 {
		t.Skip("no LXC container found on PVE_E2E_NODE; skipping opportunistic config check")
	}

	config, err := client.Nodes(cfg.Node).LXC().Config(ctx, vmid, nil)
	e2e.RequireNoError(t, "get real container config", err)
	if config.Digest == "" {
		t.Errorf("config: Digest is empty for a real container")
	}
	if config.RootFS == "" {
		t.Errorf("config: RootFS is empty for a real container")
	}
}

// TestLXCUpdateConfigAbsent verifies that UpdateConfig returns an error
// against a nonexistent container.
func TestLXCUpdateConfigAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping lxc config tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	lc := client.Nodes(cfg.Node).LXC()
	ctx := t.Context()

	err := lc.UpdateConfig(ctx, nonexistentVMID, nil)
	e2e.RequireError(t, "update config with nil options", err)

	err = lc.UpdateConfig(ctx, nonexistentVMID, &lxc.Config{Description: "e2e"})
	e2e.RequireError(t, "update config of absent container", err)
}
