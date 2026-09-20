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

package qemu

import (
	"context"
	"fmt"
)

// AttachDriveOptions holds the parameters for Client.AttachDrive.
type AttachDriveOptions struct {
	// Drive is the bus/slot to attach to, e.g. "scsi0", "virtio2",
	// "ide1", "sata3". The slot is created if it isn't already
	// configured, or its current contents replaced if it is. Required.
	Drive string
	// Options is the drive's full property set. Options.File — the
	// backing volume, e.g. "local-lvm:vm-100-disk-0" — is required;
	// every other field (Cache, SSD, IOThread, Discard, ...) is
	// optional and passed through as-is.
	Options Drive
}

// AttachDrive attaches an existing volume to a guest's disk bus via
// POST /nodes/{node}/qemu/{vmid}/config, creating the slot if
// opts.Drive isn't already configured. Proxmox has no dedicated attach
// endpoint for this — it's a thin convenience wrapper around
// UpdateConfigAsync, writing opts.Options's property string to the
// given slot the same way the web UI's "Add: Hard Disk" (existing
// volume) or "Edit" dialog does. Unlike AttachISO, media is left
// unset — use AttachISO for CD-ROM drives. Returns the config-update
// task's UPID.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Config.Disk,match=all
func (c *Client) AttachDrive(ctx context.Context, vmid int, opts *AttachDriveOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("qemu: attach drive options are required")
	}
	if opts.Options.File == "" {
		return "", fmt.Errorf("qemu: attach drive volume (Options.File) is required")
	}

	prefix, index, ok := splitIndexedKey(opts.Drive)
	cfg := &Config{}
	if !ok || !setConfigDrive(cfg, prefix, index, opts.Options) {
		return "", fmt.Errorf("qemu: drive %q must be an ide, sata, scsi, or virtio slot", opts.Drive)
	}

	return c.UpdateConfigAsync(ctx, vmid, cfg)
}

// DetachDriveOptions holds the parameters for Client.DetachDrive.
type DetachDriveOptions struct {
	// Drive is the disk's current config key, e.g. "scsi0", "virtio2".
	// Required.
	Drive string
	// Force allows detaching a disk that's still mounted/in use;
	// Proxmox otherwise rejects the request. Root only.
	Force bool
}

// DetachDrive detaches a disk from a guest's bus via
// POST /nodes/{node}/qemu/{vmid}/config, using Config's Delete
// mechanism. Proxmox converts the slot into an "unusedN" entry that
// keeps the underlying volume rather than deleting it — matching the
// web UI's "Detach" action — rather than the slot disappearing
// outright. To free the volume too, follow up with Unlink on the
// resulting unusedN key (found via a subsequent Config call). Returns
// the config-update task's UPID.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Config.Disk,match=all
func (c *Client) DetachDrive(ctx context.Context, vmid int, opts *DetachDriveOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("qemu: detach drive options are required")
	}

	prefix, _, ok := splitIndexedKey(opts.Drive)
	if !ok || !isDriveFamily(prefix) {
		return "", fmt.Errorf("qemu: drive %q must be an ide, sata, scsi, or virtio slot", opts.Drive)
	}

	cfg := &Config{
		Delete: []string{opts.Drive},
		Force:  opts.Force,
	}

	return c.UpdateConfigAsync(ctx, vmid, cfg)
}

// isDriveFamily reports whether prefix names one of the disk-bus config
// families (ide/sata/scsi/virtio) that Drive-typed entries live in —
// unlike, say, "net" or "usb".
func isDriveFamily(prefix string) bool {
	switch prefix {
	case "ide", "sata", "scsi", "virtio":
		return true
	default:
		return false
	}
}

// setConfigDrive places d into cfg's map for the drive-family prefix
// ("ide", "sata", "scsi", or "virtio"), reporting whether prefix was
// recognized as one of those.
func setConfigDrive(cfg *Config, prefix string, index int, d Drive) bool {
	if !isDriveFamily(prefix) {
		return false
	}

	switch prefix {
	case "ide":
		cfg.IDE = map[int]Drive{index: d}
	case "sata":
		cfg.SATA = map[int]Drive{index: d}
	case "scsi":
		cfg.SCSI = map[int]Drive{index: d}
	case "virtio":
		cfg.VirtIO = map[int]Drive{index: d}
	}

	return true
}
