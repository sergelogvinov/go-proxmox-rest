package firewall

import (
	"context"
)

// optionsResource provides access to GET/PUT /cluster/firewall/options, the
// cluster-wide firewall configuration.
type optionsResource struct {
	client Getter
}

// Get executes GET /cluster/firewall/options and returns the current
// cluster-wide firewall configuration.
func (o *optionsResource) Get(ctx context.Context) (*Options, error) {
	var opts Options
	if err := o.client.Get(ctx, "/cluster/firewall/options", &opts, nil); err != nil {
		return nil, err
	}

	return &opts, nil
}

// Update executes PUT /cluster/firewall/options, applying the given options
// to the cluster-wide firewall configuration. Only non-zero fields are
// sent; use Options.Delete to explicitly reset a field to its default.
func (o *optionsResource) Update(ctx context.Context, opts *Options) error {
	params, err := opts.encode()
	if err != nil {
		return err
	}

	return o.client.Update(ctx, "/cluster/firewall/options", nil, params)
}
