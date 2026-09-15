package qemu

import (
	"context"
	"fmt"
)

// customCPUModelsResource provides access to
// GET/POST /cluster/qemu/custom-cpu-models and
// GET/PUT/DELETE /cluster/qemu/custom-cpu-models/{cputype}. Obtain it via
// Client.CustomCPUModels().
type customCPUModelsResource struct {
	client Getter
}

// List retrieves all custom CPU model definitions via
// GET /cluster/qemu/custom-cpu-models.
func (r *customCPUModelsResource) List(ctx context.Context) ([]CPUModel, error) {
	var models []CPUModel
	if err := r.client.Get(ctx, "/cluster/qemu/custom-cpu-models", &models, nil); err != nil {
		return nil, err
	}

	return models, nil
}

// Get retrieves a single custom CPU model definition via
// GET /cluster/qemu/custom-cpu-models/{cputype}. The "custom-" prefix on
// cputype is optional.
func (r *customCPUModelsResource) Get(ctx context.Context, cputype string) (*CPUModel, error) {
	var m CPUModel
	if err := r.client.Get(ctx, "/cluster/qemu/custom-cpu-models/"+cputype, &m, nil); err != nil {
		return nil, err
	}

	return &m, nil
}

// Create creates a new custom CPU model definition via
// POST /cluster/qemu/custom-cpu-models.
func (r *customCPUModelsResource) Create(ctx context.Context, opts *CPUModelOptions) error {
	if opts == nil {
		return fmt.Errorf("qemu: custom cpu model options are required")
	}
	if opts.CPUType == "" {
		return fmt.Errorf("qemu: custom cpu model cputype is required")
	}
	if opts.ReportedModel == nil || *opts.ReportedModel == "" {
		return fmt.Errorf("qemu: custom cpu model reported-model is required")
	}

	params, err := opts.encode()
	if err != nil {
		return err
	}

	return r.client.Create(ctx, "/cluster/qemu/custom-cpu-models", nil, params)
}

// Update modifies an existing custom CPU model definition via
// PUT /cluster/qemu/custom-cpu-models/{cputype}. The "custom-" prefix on
// cputype is optional.
//
// opts.CPUType is ignored (overwritten with cputype): Proxmox requires the
// model identifier in the update body too, even though it is already part
// of the URL.
func (r *customCPUModelsResource) Update(ctx context.Context, cputype string, opts *CPUModelOptions) error {
	if opts == nil {
		return fmt.Errorf("qemu: custom cpu model options are required")
	}

	opts.CPUType = cputype

	params, err := opts.encode()
	if err != nil {
		return err
	}

	return r.client.Update(ctx, "/cluster/qemu/custom-cpu-models/"+cputype, nil, params)
}

// Delete removes a custom CPU model definition via
// DELETE /cluster/qemu/custom-cpu-models/{cputype}. The "custom-" prefix on
// cputype is optional.
func (r *customCPUModelsResource) Delete(ctx context.Context, cputype string) error {
	return r.client.Delete(ctx, "/cluster/qemu/custom-cpu-models/"+cputype, nil, nil)
}
