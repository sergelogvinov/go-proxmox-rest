package capabilities

import (
	"context"
)

// qemuResource provides access to /nodes/{node}/capabilities/qemu.
type qemuResource struct {
	client Getter
}

// CPUModels retrieves the CPU models (built-in and custom) available for
// QEMU guests on the given node via
// GET /nodes/{node}/capabilities/qemu/cpu. arch, when empty, defaults to
// the host's own architecture.
//
// Custom models are filtered to those the caller has Mapping.{Audit,Use,
// Modify} on, unless it holds Sys.Audit on "/nodes" (which grants
// visibility of every custom model, for backward compatibility).
func (r *qemuResource) CPUModels(ctx context.Context, node string, arch Arch) ([]CPUModel, error) {
	var p map[string]string
	if arch != "" {
		p = map[string]string{"arch": string(arch)}
	}

	var models []CPUModel
	if err := r.client.Get(ctx, "/nodes/"+node+"/capabilities/qemu/cpu", &models, p); err != nil {
		return nil, err
	}

	return models, nil
}

// CPUFlags retrieves the VM-specific CPU flags available on the given
// node via GET /nodes/{node}/capabilities/qemu/cpu-flags. arch and accel,
// when empty, default to the host's own architecture and AccelKVM
// respectively. Proxmox always returns an empty list for ArchAarch64, since
// no VM-specific flags are defined for it yet.
func (r *qemuResource) CPUFlags(ctx context.Context, node string, arch Arch, accel Accel) ([]CPUFlag, error) {
	p := make(map[string]string, 2)
	if arch != "" {
		p["arch"] = string(arch)
	}
	if accel != "" {
		p["accel"] = string(accel)
	}
	if len(p) == 0 {
		p = nil
	}

	var flags []CPUFlag
	if err := r.client.Get(ctx, "/nodes/"+node+"/capabilities/qemu/cpu-flags", &flags, p); err != nil {
		return nil, err
	}

	return flags, nil
}

// Machines retrieves the supported QEMU/KVM machine types on the given
// node via GET /nodes/{node}/capabilities/qemu/machines. arch, when empty,
// defaults to the host's own architecture.
func (r *qemuResource) Machines(ctx context.Context, node string, arch Arch) ([]MachineType, error) {
	var p map[string]string
	if arch != "" {
		p = map[string]string{"arch": string(arch)}
	}

	var machines []MachineType
	if err := r.client.Get(ctx, "/nodes/"+node+"/capabilities/qemu/machines", &machines, p); err != nil {
		return nil, err
	}

	return machines, nil
}

// Migration retrieves the given node's QEMU live-migration capabilities
// via GET /nodes/{node}/capabilities/qemu/migration.
func (r *qemuResource) Migration(ctx context.Context, node string) (*MigrationCapabilities, error) {
	caps := &MigrationCapabilities{}
	if err := r.client.Get(ctx, "/nodes/"+node+"/capabilities/qemu/migration", caps, nil); err != nil {
		return nil, err
	}

	return caps, nil
}
