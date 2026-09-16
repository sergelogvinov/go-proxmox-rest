//go:build e2e

package qemu_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestQemuMigratePreconditionAbsent verifies that GET
// /nodes/{node}/qemu/{vmid}/migrate on a nonexistent guest returns an
// error.
func TestQemuMigratePreconditionAbsent(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu migrate tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes().Qemu().MigratePrecondition(ctx, cfg.Node, nonexistentVMID, "")
	e2e.RequireError(t, "migrate precondition of absent guest", err)
}

// TestQemuMigratePreconditionOpportunistic checks migration
// preconditions for a real guest, if cfg.Node happens to host one.
// Read-only and safe: it never starts a migration.
func TestQemuMigratePreconditionOpportunistic(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu migrate tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	resources, err := client.Cluster().Resources().Get(ctx, cluster.ResourceTypeVM)
	e2e.RequireNoError(t, "list cluster vm resources", err)

	var vmid int
	for _, r := range resources {
		if r.Node == cfg.Node && r.Type == "qemu" {
			vmid = r.VMID
			break
		}
	}
	if vmid == 0 {
		t.Skip("no QEMU guest found on PVE_E2E_NODE; skipping opportunistic precondition check")
	}

	// AllowedNodes/NotAllowedNodes are legitimately both empty on a
	// single-node cluster (nothing else to check migration targets
	// against), so this only confirms the response decodes without
	// error rather than asserting on cluster topology.
	_, err = client.Nodes().Qemu().MigratePrecondition(ctx, cfg.Node, vmid, "")
	e2e.RequireNoError(t, "migrate precondition of real guest", err)
}

// TestQemuMigrateValidation verifies client-side validation of nil
// options and a missing target, without making a request.
func TestQemuMigrateValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu migrate tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes().Qemu().Migrate(ctx, cfg.Node, nonexistentVMID, nil)
	e2e.RequireError(t, "migrate with nil options", err)

	_, err = client.Nodes().Qemu().Migrate(ctx, cfg.Node, nonexistentVMID, &qemu.MigrateOptions{})
	e2e.RequireError(t, "migrate without target", err)
}

// TestQemuMigrateInvalidTarget verifies that migrating to a nonexistent
// target node returns an error. Migrate is never exercised against a
// real target here: relocating a real guest is a genuinely disruptive,
// hard-to-reverse action this suite has no business taking against
// whatever cluster PVE_E2E_URL happens to point at.
func TestQemuMigrateInvalidTarget(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping qemu migrate tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes().Qemu().Migrate(ctx, cfg.Node, nonexistentVMID, &qemu.MigrateOptions{
		Target: "e2e-nonexistent-node",
	})
	e2e.RequireError(t, "migrate to nonexistent target node", err)
}
