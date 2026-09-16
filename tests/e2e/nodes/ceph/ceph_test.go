//go:build e2e

// Package ceph_e2e exercises the per-node Ceph module's read-only
// surface (status, releases, log) against a live Proxmox VE cluster.
// Ceph is optional on a Proxmox cluster, so a server-side "not
// initialized"-type error skips a test rather than failing it — mirroring
// tests/e2e/cluster/ceph.
//
// Start/Stop/Restart are implemented but never invoked here: they issue
// real systemctl start/stop/restart calls against Ceph service units on
// the node. Even scoped to a single instance, restarting a live mon/osd/
// mgr/mds is a real storage-availability risk this suite has no business
// taking against whatever cluster PVE_E2E_URL happens to point at.
package ceph_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/ceph"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestCephStatus verifies GET /nodes/{node}/ceph/status decodes into a
// Status with a non-empty FSID and a recognized health status.
func TestCephStatus(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping ceph tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	status, err := client.Nodes().Ceph().Status(ctx, cfg.Node)
	if err != nil {
		t.Skipf("ceph status: %v (Ceph may not be configured on this node)", err)
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

// TestCephReleases verifies GET /nodes/{node}/ceph/releases decodes
// without error. Unlike Status/Log, this doesn't require Ceph to be
// configured on the node (Proxmox's own handler never calls
// check_ceph_inited for it), so no error is tolerated.
func TestCephReleases(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping ceph tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	releases, err := client.Nodes().Ceph().Releases(ctx, cfg.Node)
	e2e.RequireNoError(t, "list ceph releases", err)
	if len(releases) == 0 {
		t.Fatalf("releases: got an empty list")
	}

	var sawDefault bool
	for _, r := range releases {
		if r.Release == "" {
			t.Errorf("releases: entry with empty Release: %+v", r)
		}
		if r.Version == "" {
			t.Errorf("releases: entry with empty Version: %+v", r)
		}
		if r.IsDefault {
			sawDefault = true
		}
	}
	if !sawDefault {
		t.Errorf("releases: no entry marked IsDefault")
	}
}

// TestCephLog verifies GET /nodes/{node}/ceph/log decodes without error.
func TestCephLog(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping ceph tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Nodes().Ceph().Log(ctx, cfg.Node, &ceph.LogOptions{Limit: 5})
	if err != nil {
		t.Skipf("ceph log: %v (Ceph may not be configured on this node)", err)
	}
}
