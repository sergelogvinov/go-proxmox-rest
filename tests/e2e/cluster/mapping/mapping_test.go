//go:build e2e

// Package mapping_e2e exercises the cluster hardware mapping module
// (PCI, USB, directory) against a live Proxmox VE cluster. Every mapping
// entry is scoped to PVE_E2E_NODE with a syntactically-valid but
// non-existent device/path: Create/Update only validate the property-string
// shape, not that the hardware actually exists (that check only runs when
// List is called with a check-node), so this is safe to run without real
// PCI/USB devices.
package mapping_e2e

import (
	"fmt"
	"slices"
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster/mapping"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestMappingDirLifecycle runs the full CRUD lifecycle for a
// uniquely-named directory mapping:
//
//	list → get(absent) → create → get → update → get → delete → list
func TestMappingDirLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping directory mapping tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	dc := client.Cluster().Mapping().Dir()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete dir mapping", func() error {
				return dc.Delete(cleanupCtx, name)
			})
		})
	}

	// 1. list — baseline: our mapping must not exist yet.
	listed, err := dc.List(ctx, "")
	e2e.RequireNoError(t, "list dir mappings", err)
	if containsDir(listed, name) {
		t.Fatalf("list: dir mapping %q already exists before create", name)
	}

	// 2. get (absent) — a non-existent mapping must return an error.
	_, err = dc.Get(ctx, name)
	e2e.RequireError(t, "get absent dir mapping", err)

	// 3. create — a uniquely-named mapping with one node entry.
	comment := "e2e lifecycle"
	mapEntry := fmt.Sprintf("node=%s,path=/tmp", cfg.Node)
	err = dc.Create(ctx, &mapping.DirOptions{
		ID:          name,
		Description: &comment,
		Map:         []string{mapEntry},
	})
	e2e.RequireNoError(t, "create dir mapping", err)

	// 4. get — verify the create.
	m, err := dc.Get(ctx, name)
	e2e.RequireNoError(t, "get dir mapping after create", err)
	if m.ID != name {
		t.Errorf("get: ID = %q, want %q", m.ID, name)
	}
	if m.Description != comment {
		t.Errorf("get: Description = %q, want %q", m.Description, comment)
	}
	if len(m.Map) != 1 || m.Map[0] != mapEntry {
		t.Errorf("get: Map = %v, want [%q]", m.Map, mapEntry)
	}

	// 5. update — mutate the description. Map is required on every
	// Update (Proxmox does not treat it as optional there), so it must
	// be resent even though it is unchanged.
	newComment := "e2e updated"
	err = dc.Update(ctx, name, &mapping.DirOptions{Description: &newComment, Map: []string{mapEntry}})
	e2e.RequireNoError(t, "update dir mapping", err)

	// 6. get — verify the update.
	m, err = dc.Get(ctx, name)
	e2e.RequireNoError(t, "get dir mapping after update", err)
	if m.Description != newComment {
		t.Errorf("get after update: Description = %q, want %q", m.Description, newComment)
	}
	if len(m.Map) != 1 || m.Map[0] != mapEntry {
		t.Errorf("get after update: Map = %v, want unchanged [%q]", m.Map, mapEntry)
	}

	// 7. delete — remove the mapping.
	err = dc.Delete(ctx, name)
	e2e.RequireNoError(t, "delete dir mapping", err)

	// 8. list — verify the delete.
	listed, err = dc.List(ctx, "")
	e2e.RequireNoError(t, "list dir mappings after delete", err)
	if containsDir(listed, name) {
		t.Errorf("list after delete: dir mapping %q still present", name)
	}

	_, err = dc.Get(ctx, name)
	e2e.RequireError(t, "get dir mapping after delete", err)
}

// TestMappingPCICreateDelete verifies that a PCI mapping with a
// multi-key property-string "map" entry round-trips correctly — this is
// the key regression check for the repeated-query-parameter encoding
// (see mapping.Getter's doc comment).
func TestMappingPCICreateDelete(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping PCI mapping tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	pc := client.Cluster().Mapping().PCI()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete pci mapping", func() error {
				return pc.Delete(cleanupCtx, name)
			})
		})
	}

	mapEntry := fmt.Sprintf("node=%s,path=0000:01:00.0,id=1234:5678", cfg.Node)
	err := pc.Create(ctx, &mapping.PCIOptions{
		ID:  name,
		Map: []string{mapEntry},
	})
	e2e.RequireNoError(t, "create pci mapping", err)

	m, err := pc.Get(ctx, name)
	e2e.RequireNoError(t, "get pci mapping after create", err)
	if m.ID != name {
		t.Errorf("get: ID = %q, want %q", m.ID, name)
	}
	if len(m.Map) != 1 || m.Map[0] != mapEntry {
		t.Errorf("get: Map = %v, want [%q]", m.Map, mapEntry)
	}

	err = pc.Delete(ctx, name)
	e2e.RequireNoError(t, "delete pci mapping", err)

	_, err = pc.Get(ctx, name)
	e2e.RequireError(t, "get pci mapping after delete", err)
}

// TestMappingUSBCreateDelete verifies the USB mapping's Create/Get/Delete
// round trip.
func TestMappingUSBCreateDelete(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping USB mapping tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	uc := client.Cluster().Mapping().USB()
	ctx := t.Context()

	name := e2e.UniqueName(cfg.Prefix)

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete usb mapping", func() error {
				return uc.Delete(cleanupCtx, name)
			})
		})
	}

	mapEntry := fmt.Sprintf("node=%s,id=1234:5678", cfg.Node)
	err := uc.Create(ctx, &mapping.USBOptions{
		ID:  name,
		Map: []string{mapEntry},
	})
	e2e.RequireNoError(t, "create usb mapping", err)

	m, err := uc.Get(ctx, name)
	e2e.RequireNoError(t, "get usb mapping after create", err)
	if m.ID != name {
		t.Errorf("get: ID = %q, want %q", m.ID, name)
	}
	if len(m.Map) != 1 || m.Map[0] != mapEntry {
		t.Errorf("get: Map = %v, want [%q]", m.Map, mapEntry)
	}

	err = uc.Delete(ctx, name)
	e2e.RequireNoError(t, "delete usb mapping", err)

	_, err = uc.Get(ctx, name)
	e2e.RequireError(t, "get usb mapping after delete", err)
}

// TestMappingValidation verifies client-side validation of Create options,
// none of which should reach the API.
func TestMappingValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	m := client.Cluster().Mapping()
	ctx := t.Context()

	err := m.PCI().Create(ctx, nil)
	e2e.RequireError(t, "pci create with nil options", err)
	err = m.PCI().Create(ctx, &mapping.PCIOptions{})
	e2e.RequireError(t, "pci create with missing id", err)
	err = m.PCI().Update(ctx, "does-not-matter", nil)
	e2e.RequireError(t, "pci update with nil options", err)
	err = m.PCI().Update(ctx, "does-not-matter", &mapping.PCIOptions{})
	e2e.RequireError(t, "pci update with missing map", err)

	err = m.USB().Create(ctx, nil)
	e2e.RequireError(t, "usb create with nil options", err)
	err = m.USB().Create(ctx, &mapping.USBOptions{})
	e2e.RequireError(t, "usb create with missing id", err)
	err = m.USB().Update(ctx, "does-not-matter", nil)
	e2e.RequireError(t, "usb update with nil options", err)
	err = m.USB().Update(ctx, "does-not-matter", &mapping.USBOptions{})
	e2e.RequireError(t, "usb update with missing map", err)

	err = m.Dir().Create(ctx, nil)
	e2e.RequireError(t, "dir create with nil options", err)
	err = m.Dir().Create(ctx, &mapping.DirOptions{})
	e2e.RequireError(t, "dir create with missing id", err)
	err = m.Dir().Update(ctx, "does-not-matter", nil)
	e2e.RequireError(t, "dir update with nil options", err)
	err = m.Dir().Update(ctx, "does-not-matter", &mapping.DirOptions{})
	e2e.RequireError(t, "dir update with missing map", err)
}

// containsDir reports whether the slice contains a directory mapping with
// the given ID.
func containsDir(list []mapping.Dir, id string) bool {
	return slices.ContainsFunc(list, func(d mapping.Dir) bool { return d.ID == id })
}
