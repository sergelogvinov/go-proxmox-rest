/*
Copyright 2026 Proxmox Community.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hardware

// PCIDevice describes a local PCI device, as returned by
// GET /nodes/{node}/hardware/pci.
type PCIDevice struct {
	// ID is the PCI ID, e.g. "0000:01:00.0".
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Class is the PCI class of the device.
	Class string `json:"class,omitempty" url:"class,omitempty"`
	// Vendor is the vendor ID.
	Vendor string `json:"vendor,omitempty" url:"vendor,omitempty"`
	// VendorName is the resolved vendor name; only populated when the
	// scan was verbose (the default).
	VendorName string `json:"vendor_name,omitempty" url:"vendor_name,omitempty"`
	// Device is the device ID.
	Device string `json:"device,omitempty" url:"device,omitempty"`
	// DeviceName is the resolved device name; only populated when the
	// scan was verbose (the default).
	DeviceName string `json:"device_name,omitempty" url:"device_name,omitempty"`
	// SubsystemVendor is the subsystem vendor ID.
	SubsystemVendor string `json:"subsystem_vendor,omitempty" url:"subsystem_vendor,omitempty"`
	// SubsystemVendorName is the resolved subsystem vendor name; only
	// populated when the scan was verbose (the default).
	SubsystemVendorName string `json:"subsystem_vendor_name,omitempty" url:"subsystem_vendor_name,omitempty"`
	// SubsystemDevice is the subsystem device ID.
	SubsystemDevice string `json:"subsystem_device,omitempty" url:"subsystem_device,omitempty"`
	// SubsystemDeviceName is the resolved subsystem device name; only
	// populated when the scan was verbose (the default).
	SubsystemDeviceName string `json:"subsystem_device_name,omitempty" url:"subsystem_device_name,omitempty"`
	// IOMMUGroup is the IOMMU group the device belongs to, or -1 if none
	// was detected.
	IOMMUGroup int `json:"iommugroup,omitempty" url:"iommugroup,omitempty"`
	// Mdev is true if the device is capable of creating mediated
	// devices.
	Mdev bool `json:"mdev,omitempty" url:"mdev,omitempty"`
}

// PCIScanOptions filters/tunes the PCI device listing returned by
// pciResource.List. A nil *PCIScanOptions requests Proxmox's defaults
// (class-blacklist "05;06;0b", verbose output).
type PCIScanOptions struct {
	// ClassBlacklist overrides the PCI classes excluded from the result
	// (default: "05" Memory Controller, "06" Bridge, "0b" Processor).
	ClassBlacklist []string `url:"pci-class-blacklist,omitempty"`
	// Verbose includes vendor/device names and other extra info in the
	// response when true (the server default). Set to a pointer to
	// false to only return PCI IDs.
	Verbose *bool `url:"verbose,omitempty"`
}

// MdevType describes a mediated device type available for a given PCI
// device or resource-mapped device, as returned by
// GET /nodes/{node}/hardware/pci/{pci-id-or-mapping}/mdev.
type MdevType struct {
	// Type is the mdev type's name.
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Available is the number of still-available instances of this
	// type.
	Available int64 `json:"available,omitempty" url:"available,omitempty"`
	// Description is additional, free-form information about the type.
	Description string `json:"description,omitempty" url:"description,omitempty"`
	// Name is a human-readable name for the type.
	Name string `json:"name,omitempty" url:"name,omitempty"`
}

// USBDevice describes a local USB device, as returned by
// GET /nodes/{node}/hardware/usb.
type USBDevice struct {
	Busnum       int    `json:"busnum,omitempty" url:"busnum,omitempty"`
	Class        int    `json:"class,omitempty" url:"class,omitempty"`
	Devnum       int    `json:"devnum,omitempty" url:"devnum,omitempty"`
	Level        int    `json:"level,omitempty" url:"level,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty" url:"manufacturer,omitempty"`
	Port         int    `json:"port,omitempty" url:"port,omitempty"`
	Prodid       string `json:"prodid,omitempty" url:"prodid,omitempty"`
	Product      string `json:"product,omitempty" url:"product,omitempty"`
	Serial       string `json:"serial,omitempty" url:"serial,omitempty"`
	Speed        string `json:"speed,omitempty" url:"speed,omitempty"`
	Usbpath      string `json:"usbpath,omitempty" url:"usbpath,omitempty"`
	Vendid       string `json:"vendid,omitempty" url:"vendid,omitempty"`
}
