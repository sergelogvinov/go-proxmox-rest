package cluster

import (
	"context"
)

// optionsResource provides access to GET/PUT /cluster/options, the
// cluster-wide datacenter configuration.
type optionsResource struct {
	client Getter
}

// Options returns an accessor for GET/PUT /cluster/options.
func (c *Client) Options() *optionsResource {
	return &optionsResource{client: c.client}
}

// Get executes GET /cluster/options and returns the current cluster-wide
// configuration.
func (o *optionsResource) Get(ctx context.Context) (*Options, error) {
	var opts Options
	if err := o.client.Get(ctx, "/cluster/options", &opts, nil); err != nil {
		return nil, err
	}

	return &opts, nil
}

// Update executes PUT /cluster/options, applying the given options to the
// cluster-wide configuration. Only non-zero fields are sent; use
// Options.Delete to explicitly reset a field to its default.
func (o *optionsResource) Update(ctx context.Context, opts *Options) error {
	params, err := opts.encode()
	if err != nil {
		return err
	}

	return o.client.Update(ctx, "/cluster/options", nil, params)
}
