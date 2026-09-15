//go:build e2e

// Package ceph_e2e exercises the read-only cluster Ceph module against a
// live Proxmox VE cluster: cluster-wide status.
package ceph_e2e

import (
	"testing"

	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestCephStatus verifies GET /cluster/ceph/status decodes into a Status
// with a non-empty FSID and a recognized health status. Ceph is optional on
// a Proxmox cluster, so a server-side error (not initialized) skips the
// test rather than failing it.
func TestCephStatus(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	status, err := client.Cluster().Ceph().Status(ctx)
	if err != nil {
		t.Skipf("ceph status: %v (Ceph may not be configured on this cluster)", err)
	}

	if status.FSID == "" {
		t.Errorf("status: FSID is empty")
	}
	if status.Health == nil || status.Health.Status == "" {
		t.Fatalf("status: Health is empty, want a health summary")
	}

	switch status.Health.Status {
	case "HEALTH_OK", "HEALTH_WARN", "HEALTH_ERR":
	default:
		t.Errorf("status: Health.Status = %q, want one of HEALTH_OK/HEALTH_WARN/HEALTH_ERR", status.Health.Status)
	}
}
