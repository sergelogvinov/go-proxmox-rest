package hardware

import (
	"context"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// pciResource provides access to /nodes/{node}/hardware/pci.
type pciResource struct {
	client Getter
}

// List retrieves the given node's local PCI devices via
// GET /nodes/{node}/hardware/pci. opts may be nil to request Proxmox's
// default class blacklist and verbose output.
func (r *pciResource) List(ctx context.Context, node string, opts *PCIScanOptions) ([]PCIDevice, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var devices []PCIDevice
	if err := r.client.Get(ctx, "/nodes/"+node+"/hardware/pci", &devices, p); err != nil {
		return nil, err
	}

	return devices, nil
}

// MdevTypes retrieves the mediated device types available for a PCI device
// (or resource mapping) via
// GET /nodes/{node}/hardware/pci/{pciIDOrMapping}/mdev.
//
// pciIDOrMapping is either a raw PCI ID (e.g. "0000:01:00.0") or the name
// of a cluster.Mapping().PCI() resource mapping.
func (r *pciResource) MdevTypes(ctx context.Context, node, pciIDOrMapping string) ([]MdevType, error) {
	var types []MdevType
	if err := r.client.Get(ctx, "/nodes/"+node+"/hardware/pci/"+pciIDOrMapping+"/mdev", &types, nil); err != nil {
		return nil, err
	}

	return types, nil
}
