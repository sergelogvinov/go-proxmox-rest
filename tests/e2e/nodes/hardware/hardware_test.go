//go:build e2e

// Package hardware_e2e exercises the read-only node hardware module
// against a live Proxmox VE cluster: local PCI and USB device inventories.
package hardware_e2e

import (
	"testing"

	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestHardwarePCI verifies GET /nodes/{node}/hardware/pci decodes without
// error and that every entry has a non-empty ID, then, if any device
// reports mdev capability, that its mdev type listing also decodes without
// error.
func TestHardwarePCI(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping hardware tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	pci := client.Nodes(cfg.Node).Hardware().PCI()
	ctx := t.Context()

	devices, err := pci.List(ctx, nil)
	e2e.RequireNoError(t, "list PCI devices", err)
	if len(devices) == 0 {
		t.Fatalf("list PCI devices: got no entries, want at least one device")
	}

	var mdevCapable string
	for _, d := range devices {
		if d.ID == "" {
			t.Errorf("list PCI devices: entry with empty ID: %+v", d)
		}
		if d.Mdev && mdevCapable == "" {
			mdevCapable = d.ID
		}
	}

	if mdevCapable != "" {
		_, err := pci.MdevTypes(ctx, mdevCapable)
		e2e.RequireNoError(t, "mdev types", err)
	}
}

// TestHardwareUSB verifies GET /nodes/{node}/hardware/usb decodes without
// error. The node may have no USB devices at all, so this doesn't assert
// on the list being non-empty.
func TestHardwareUSB(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}
	if cfg.Node == "" {
		t.Skip("PVE_E2E_NODE is not set; skipping hardware tests")
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	devices, err := client.Nodes(cfg.Node).Hardware().USB().List(ctx)
	e2e.RequireNoError(t, "list USB devices", err)

	for _, d := range devices {
		if d.Vendid == "" || d.Prodid == "" {
			t.Errorf("list USB devices: entry with empty Vendid/Prodid: %+v", d)
		}
	}
}
