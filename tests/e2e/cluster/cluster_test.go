//go:build e2e

// Package cluster_e2e exercises the read-only cluster module against a live
// Proxmox VE cluster: status, resources (unfiltered and type-filtered),
// recent tasks, and the next free VMID.
package cluster_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestClusterStatus verifies GET /cluster/status returns at least the local
// node entry.
func TestClusterStatus(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	status, err := client.Cluster().Status(ctx)
	e2e.RequireNoError(t, "cluster status", err)

	if len(status.Nodes) == 0 {
		t.Errorf("status: Nodes is empty, want at least the local node")
	}
}

// TestClusterResources verifies GET /cluster/resources returns entries and
// that Type narrows the result to the requested resource type.
func TestClusterResources(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	cc := client.Cluster()
	ctx := t.Context()

	// Unfiltered: at least the local node must be present, and every entry
	// must decode with a non-empty ID/Type.
	all, err := cc.Resources().Get(ctx, "")
	e2e.RequireNoError(t, "resources (unfiltered)", err)
	if len(all) == 0 {
		t.Fatalf("resources: got no entries, want at least the local node")
	}
	for _, r := range all {
		if r.ID == "" || r.Type == "" {
			t.Errorf("resources: entry with empty ID/Type: %+v", r)
		}
	}

	// Filtered by node: every cluster has at least one node, so this must
	// be non-empty and every entry's Type must be "node".
	nodes, err := cc.Resources().Get(ctx, cluster.ResourceTypeNode)
	e2e.RequireNoError(t, "resources (type=node)", err)
	if len(nodes) == 0 {
		t.Fatalf("resources(type=node): got no entries, want at least one node")
	}
	for _, r := range nodes {
		if r.Type != "node" {
			t.Errorf("resources(type=node): Type = %q, want %q", r.Type, "node")
		}
	}

	// Filtered by vm: the cluster may have zero guests, so only assert that
	// whatever comes back is actually a guest entry.
	vms, err := cc.Resources().Get(ctx, cluster.ResourceTypeVM)
	e2e.RequireNoError(t, "resources (type=vm)", err)
	for _, r := range vms {
		if r.Type != "qemu" && r.Type != "lxc" {
			t.Errorf("resources(type=vm): Type = %q, want %q or %q", r.Type, "qemu", "lxc")
		}
	}
}

// TestClusterTasks verifies GET /cluster/tasks decodes without error and
// that every entry has a non-empty UPID/Node/Type. The cluster may have no
// recent tasks at all, so this doesn't assert on the list being non-empty.
func TestClusterTasks(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	tasks, err := client.Cluster().Tasks(ctx)
	e2e.RequireNoError(t, "cluster tasks", err)

	for _, task := range tasks {
		if task.UPID == "" || task.Node == "" || task.Type == "" {
			t.Errorf("tasks: entry with empty UPID/Node/Type: %+v", task)
		}
	}
}

// TestClusterNextID verifies GET /cluster/nextid returns a usable VMID both
// unfiltered and when asserting a specific, currently-free id is available.
func TestClusterNextID(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	cc := client.Cluster()
	ctx := t.Context()

	next, err := cc.NextID(ctx, 0)
	e2e.RequireNoError(t, "nextid (unfiltered)", err)
	if next < 100 {
		t.Errorf("nextid: got %d, want >= 100", next)
	}

	// Asserting that the id nextid itself just returned is still free
	// must succeed and return the same id.
	same, err := cc.NextID(ctx, next)
	e2e.RequireNoError(t, "nextid (assert free)", err)
	if same != next {
		t.Errorf("nextid (assert free): got %d, want %d", same, next)
	}
}
