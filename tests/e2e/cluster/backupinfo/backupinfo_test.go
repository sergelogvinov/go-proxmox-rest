//go:build e2e

// Package backupinfo_e2e exercises the cluster backup-info module against
// a live Proxmox VE cluster. It's a single read-only endpoint, so there's
// nothing to validate beyond a successful decode.
package backupinfo_e2e

import (
	"testing"

	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestBackupInfoNotBackedUp verifies GET /cluster/backup-info/not-backed-up
// decodes without error.
func TestBackupInfoNotBackedUp(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	guests, err := client.Cluster().BackupInfoNotBackedUp(ctx)
	e2e.RequireNoError(t, "list guests not backed up", err)

	for _, g := range guests {
		if g.VMID == 0 {
			t.Errorf("not-backed-up guest has zero VMID: %+v", g)
		}
		if g.Type != "qemu" && g.Type != "lxc" {
			t.Errorf("not-backed-up guest %d: Type = %q, want qemu or lxc", g.VMID, g.Type)
		}
	}
}
