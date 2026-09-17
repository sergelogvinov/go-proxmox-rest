package hardware

import (
	"context"
)

// usbResource provides access to /nodes/{node}/hardware/usb.
type usbResource struct {
	client Getter
	node   string
}

// List retrieves the node's local USB devices via
// GET /nodes/{node}/hardware/usb.
func (r *usbResource) List(ctx context.Context) ([]USBDevice, error) {
	var devices []USBDevice
	if err := r.client.Get(ctx, "/nodes/"+r.node+"/hardware/usb", &devices, nil); err != nil {
		return nil, err
	}

	return devices, nil
}
