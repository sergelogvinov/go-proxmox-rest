package lxc

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// CloneOptions holds the parameters for Client.Clone
// (POST /nodes/{node}/lxc/{vmid}/clone).
type CloneOptions struct {
	// NewID is the VMID for the clone. Required.
	NewID int `url:"newid"`
	// Hostname sets a host name for the new container.
	Hostname string `url:"hostname,omitempty"`
	// Description sets a description for the new container.
	Description string `url:"description,omitempty"`
	// Pool adds the new container to the given pool.
	Pool string `url:"pool,omitempty"`
	// Snapname clones from the given snapshot instead of the
	// container's current state.
	Snapname string `url:"snapname,omitempty"`
	// Storage is the target storage for a full clone.
	Storage string `url:"storage,omitempty"`
	// Full forces a full clone (copying every disk) when true, or a
	// linked clone when false. Proxmox defaults to a full clone for a
	// normal container and a linked clone for a template.
	Full *bool `url:"full"`
	// Target is the target node. Only allowed when the source
	// container is on shared storage.
	Target string `url:"target,omitempty"`
	// BWLimit overrides the clone's I/O bandwidth limit, in KiB/s. Zero
	// uses the datacenter/storage default.
	BWLimit float64 `url:"bwlimit,omitempty"`
}

// Clone creates a copy of a container (or template) via
// POST /nodes/{node}/lxc/{vmid}/clone. Returns the clone task's UPID.
func (c *Client) Clone(ctx context.Context, node string, vmid int, opts *CloneOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("lxc: clone options are required")
	}
	if opts.NewID == 0 {
		return "", fmt.Errorf("lxc: clone newid is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+node+"/lxc/"+strconv.Itoa(vmid)+"/clone", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
