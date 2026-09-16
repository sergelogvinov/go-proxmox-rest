package firewall

import (
	"context"
)

// optionsResource provides access to GET/PUT {path}, a guest's
// firewall configuration. Obtain it via Client.Options(node, vmid).
type optionsResource struct {
	client Getter
	path   string
}

// Get executes GET {path} and returns the guest's current firewall
// configuration.
func (o *optionsResource) Get(ctx context.Context) (*Options, error) {
	var opts Options
	if err := o.client.Get(ctx, o.path, &opts, nil); err != nil {
		return nil, err
	}

	return &opts, nil
}

// Update executes PUT {path}, applying the given options to the
// guest's firewall configuration. Only non-zero fields are sent; use
// Options.Delete to explicitly reset a field to its default.
func (o *optionsResource) Update(ctx context.Context, opts *Options) error {
	params, err := opts.encode()
	if err != nil {
		return err
	}

	return o.client.Update(ctx, o.path, nil, params)
}
