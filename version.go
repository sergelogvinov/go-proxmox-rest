package proxmox

import (
	"context"
)

// Version contains the Proxmox VE version information.
type Version struct {
	// Release is the version string, e.g. "8.2.2".
	Release string `json:"release,omitempty"`
	// Version is the Proxmox VE version string, e.g. "8.2.2".
	Version string `json:"version,omitempty"`
	// Repoid is the git commit hash of the running pve-manager.
	Repoid string `json:"repoid,omitempty"`
}

// Version retrieves the version information of the Proxmox VE server.
func (c *Client) Version(ctx context.Context) (*Version, error) {
	v := &Version{}
	if err := c.Get(ctx, "/version", v, nil); err != nil {
		return nil, err
	}
	return v, nil
}
