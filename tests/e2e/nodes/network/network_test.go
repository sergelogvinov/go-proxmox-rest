//go:build e2e

// Package network_e2e exercises the per-node network module against a live
// Proxmox VE cluster.
//
// List/Get are exercised read-only. The lifecycle test creates, updates and
// deletes a uniquely-named "alias" interface (a virtual lo:<label>
// sub-interface, e.g. "lo:12345") — Create/Update/Delete only ever touch
// Proxmox's pending /etc/network/interfaces.new file, never the live
// network state, so this is safe without a Reload. Reload itself (which
// applies pending changes via "ifreload -a" and can affect real
// connectivity) is deliberately never called here.
package network_e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/network"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestNetworkList verifies GET /nodes/{node}/network decodes without error,
// both unfiltered and filtered by type.
func TestNetworkList(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping network tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	nc := client.Nodes().Network()
	ctx := t.Context()

	all, err := nc.List(ctx, cfg.Node, "")
	e2e.RequireNoError(t, "list interfaces (unfiltered)", err)
	for _, ifc := range all {
		if ifc.Iface == "" || ifc.Type == "" {
			t.Errorf("list interfaces: entry with empty Iface/Type: %+v", ifc)
		}
	}

	// The node may have zero bridges configured, so only assert that
	// whatever comes back is actually a bridge entry.
	bridges, err := nc.List(ctx, cfg.Node, network.TypeBridge)
	e2e.RequireNoError(t, "list interfaces (type=bridge)", err)
	for _, ifc := range bridges {
		if ifc.Type != network.TypeBridge {
			t.Errorf("list interfaces(type=bridge): Type = %q, want %q", ifc.Type, network.TypeBridge)
		}
	}
}

// TestNetworkLifecycle runs the full CRUD lifecycle for a uniquely-named
// "alias" interface bound to the loopback device:
//
//	get(absent) → create → get → update → get → delete → get(absent)
func TestNetworkLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping network tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	nc := client.Nodes().Network()
	ctx := t.Context()

	// Proxmox's pve-iface format requires an alias suffix to be purely
	// numeric ("^[a-z][a-z0-9_]{1,20}([:\.]\d+)?\z" in PVE::JSONSchema),
	// so this can't reuse e2e.UniqueName's alphanumeric suffix.
	iface := fmt.Sprintf("lo:%d", time.Now().UnixNano()%100000)

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete network interface", func() error {
				return nc.Delete(cleanupCtx, cfg.Node, iface)
			})
		})
	}

	// 1. get (absent) — a non-existent interface must return an error.
	_, err := nc.Get(ctx, cfg.Node, iface)
	e2e.RequireError(t, "get absent interface", err)

	// 2. create.
	comment := "go-proxmox-rest " + e2e.UniqueName(cfg.Prefix)
	err = nc.Create(ctx, cfg.Node, iface, &network.InterfaceOptions{
		Type:     network.TypeAlias,
		Comments: &comment,
	})
	e2e.RequireNoError(t, "create network interface", err)

	// 3. get — verify the pending config was applied. Proxmox stores
	// Comments as "#"-prefixed comment lines and always appends a
	// trailing "\n" when reassembling them
	// (PVE::Network::Interfaces), so a single-line comment round-trips
	// with one trailing newline.
	got, err := nc.Get(ctx, cfg.Node, iface)
	e2e.RequireNoError(t, "get network interface", err)
	if got.Type != network.TypeAlias {
		t.Errorf("get: Type = %q, want %q", got.Type, network.TypeAlias)
	}
	wantComment := comment + "\n"
	if got.Comments != wantComment {
		t.Errorf("get: Comments = %q, want %q", got.Comments, wantComment)
	}

	// 4. update.
	updatedComment := comment + "-updated"
	err = nc.Update(ctx, cfg.Node, iface, &network.InterfaceOptions{
		Type:     network.TypeAlias,
		Comments: &updatedComment,
	})
	e2e.RequireNoError(t, "update network interface", err)

	// 5. get — verify the update was applied.
	got, err = nc.Get(ctx, cfg.Node, iface)
	e2e.RequireNoError(t, "get network interface after update", err)
	wantUpdatedComment := updatedComment + "\n"
	if got.Comments != wantUpdatedComment {
		t.Errorf("get after update: Comments = %q, want %q", got.Comments, wantUpdatedComment)
	}

	// 6. delete.
	err = nc.Delete(ctx, cfg.Node, iface)
	e2e.RequireNoError(t, "delete network interface", err)

	// 7. get (absent) — must be gone.
	_, err = nc.Get(ctx, cfg.Node, iface)
	e2e.RequireError(t, "get deleted interface", err)
}
