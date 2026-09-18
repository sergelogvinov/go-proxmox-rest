//go:build e2e

package cluster_e2e

import (
	"testing"

	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestHAStatus verifies GET /cluster/ha/status/current and
// GET /cluster/ha/status/manager_status decode without error.
//
// Update (arm-ha/disarm-ha) is deliberately not exercised here: it releases
// every fencing watchdog cluster-wide, which is destructive to the live
// cluster rather than to a uniquely-named resource the suite owns — see
// docs/e2e.md's Non-Goals. TestHAStatusUpdateValidation below only checks
// the client-side argument validation.
func TestHAStatus(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	hs := client.Cluster().HA().Status()
	ctx := t.Context()

	current, err := hs.Current(ctx)
	e2e.RequireNoError(t, "ha status current", err)
	if len(current) == 0 {
		t.Fatalf("current: got no entries, want at least a quorum entry")
	}
	for _, entry := range current {
		if entry.ID == "" || entry.Node == "" || entry.Type == "" {
			t.Errorf("current: entry with empty ID/Node/Type: %+v", entry)
		}
	}

	// manager_status returns an empty object until HA has actually
	// elected a master, so only assert it decodes without error.
	_, err = hs.ManagerStatus(ctx)
	e2e.RequireNoError(t, "ha status manager_status", err)
}

// TestHAStatusUpdateValidation verifies client-side validation of Update's
// arguments; it must reject a disarm request with no resource mode before
// ever reaching the API. It deliberately never calls Update with valid
// arguments — see TestHAStatus's doc comment.
func TestHAStatusUpdateValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	hs := client.Cluster().HA().Status()
	ctx := t.Context()

	err := hs.Update(ctx, false, "")
	e2e.RequireError(t, "disarm with missing resource mode", err)
}
