//go:build e2e

// Package qemu_e2e exercises the cluster QEMU module against a live
// Proxmox VE cluster: read-only CPU flags, and the full CRUD lifecycle of
// custom CPU model definitions:
//
//	list → get(absent) → create → get → update → get → delete → list
package qemu_e2e

import (
	"slices"
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster/qemu"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestCPUFlags verifies GET /cluster/qemu/cpu-flags decodes into a
// non-empty list of named flags for the default (host) architecture.
func TestCPUFlags(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	flags, err := client.Cluster().Qemu().CPUFlags(ctx, "", "")
	e2e.RequireNoError(t, "cpu flags", err)

	for _, f := range flags {
		if f.Name == "" {
			t.Errorf("cpu flags: entry with empty Name: %+v", f)
		}
	}
}

// TestCustomCPUModelsLifecycle runs the full CRUD lifecycle for a
// uniquely-named custom CPU model.
func TestCustomCPUModelsLifecycle(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	cc := client.Cluster().Qemu().CustomCPUModels()
	ctx := t.Context()

	// pve-configid names must start with a letter; the default prefix
	// (e2e-) already satisfies that.
	name := e2e.UniqueName(cfg.Prefix)
	cputype := "custom-" + name

	if cfg.CleanupOnFailure {
		t.Cleanup(func() {
			cleanupCtx, cancel := e2e.CleanupContext()
			defer cancel()

			e2e.RetryCleanup(t, "delete custom cpu model", func() error {
				return cc.Delete(cleanupCtx, name)
			})
		})
	}

	// 1. list — baseline: our model must not exist yet.
	listed, err := cc.List(ctx)
	e2e.RequireNoError(t, "list custom cpu models", err)
	if containsCPUModel(listed, cputype) {
		t.Fatalf("list: custom cpu model %q already exists before create", cputype)
	}

	// 2. get (absent) — a non-existent model must return an error.
	_, err = cc.Get(ctx, name)
	e2e.RequireError(t, "get absent custom cpu model", err)

	// 3. create — a uniquely-named model reporting as kvm64.
	reportedModel := "kvm64"
	err = cc.Create(ctx, &qemu.CPUModelOptions{
		CPUType:       name,
		ReportedModel: &reportedModel,
	})
	e2e.RequireNoError(t, "create custom cpu model", err)

	// 4. get — verify the create. The "custom-" prefix on cputype is
	// optional and added automatically.
	m, err := cc.Get(ctx, name)
	e2e.RequireNoError(t, "get custom cpu model after create", err)
	if m.CPUType != cputype {
		t.Errorf("get: CPUType = %q, want %q", m.CPUType, cputype)
	}
	if m.ReportedModel != reportedModel {
		t.Errorf("get: ReportedModel = %q, want %q", m.ReportedModel, reportedModel)
	}

	// 5. update — mutate flags; ReportedModel stays the same.
	flags := "+aes;-hypervisor"
	err = cc.Update(ctx, name, &qemu.CPUModelOptions{Flags: &flags})
	e2e.RequireNoError(t, "update custom cpu model", err)

	// 6. get — verify the update; ReportedModel must be unchanged.
	m, err = cc.Get(ctx, name)
	e2e.RequireNoError(t, "get custom cpu model after update", err)
	if m.Flags != flags {
		t.Errorf("get after update: Flags = %q, want %q", m.Flags, flags)
	}
	if m.ReportedModel != reportedModel {
		t.Errorf("get after update: ReportedModel = %q, want unchanged %q", m.ReportedModel, reportedModel)
	}

	// 7. delete — remove the model.
	err = cc.Delete(ctx, name)
	e2e.RequireNoError(t, "delete custom cpu model", err)

	// 8. list — verify the delete.
	listed, err = cc.List(ctx)
	e2e.RequireNoError(t, "list custom cpu models after delete", err)
	if containsCPUModel(listed, cputype) {
		t.Errorf("list after delete: custom cpu model %q still present", cputype)
	}

	_, err = cc.Get(ctx, name)
	e2e.RequireError(t, "get custom cpu model after delete", err)
}

// TestCustomCPUModelsValidation verifies client-side validation of Create
// options, none of which should reach the API.
func TestCustomCPUModelsValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	cc := client.Cluster().Qemu().CustomCPUModels()
	ctx := t.Context()

	err := cc.Create(ctx, nil)
	e2e.RequireError(t, "create with nil options", err)

	reportedModel := "kvm64"
	err = cc.Create(ctx, &qemu.CPUModelOptions{ReportedModel: &reportedModel})
	e2e.RequireError(t, "create with missing cputype", err)

	err = cc.Create(ctx, &qemu.CPUModelOptions{CPUType: "does-not-matter"})
	e2e.RequireError(t, "create with missing reported-model", err)

	err = cc.Update(ctx, "does-not-matter", nil)
	e2e.RequireError(t, "update with nil options", err)
}

// containsCPUModel reports whether the slice contains a custom CPU model
// with the given cputype.
func containsCPUModel(list []qemu.CPUModel, cputype string) bool {
	return slices.ContainsFunc(list, func(m qemu.CPUModel) bool { return m.CPUType == cputype })
}
