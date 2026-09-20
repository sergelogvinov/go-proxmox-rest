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

// AttachISOOptions holds the parameters for Client.AttachISO.
type AttachISOOptions struct {
	// Drive is the CD-ROM's config key, e.g. "ide2", "sata1", "scsi3".
	// Only the ide/sata/scsi buses support cdrom media (virtio does
	// not); the slot is created if it isn't already configured, or its
	// current contents (disk or ISO) are replaced if it is. Required.
	Drive string
	// Volume is the ISO's volume id to mount, e.g.
	// "local:iso/debian-12.iso". Required.
	Volume string
}

// AttachISO mounts an ISO into a guest's CD-ROM drive via
// POST /nodes/{node}/qemu/{vmid}/config, creating the drive if
// opts.Drive isn't already configured. Proxmox has no dedicated attach
// endpoint for this — it's a thin convenience wrapper around
// UpdateConfigAsync, writing "<volume>,media=cdrom" to the given slot
// the same way the web UI's "CD/DVD Drive" dialog does. Returns the
// config-update task's UPID.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Config.CDROM,match=all
func (c *Client) AttachISO(ctx context.Context, vmid int, opts *AttachISOOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("qemu: attach iso options are required")
	}
	if opts.Volume == "" {
		return "", fmt.Errorf("qemu: attach iso volume is required")
	}

	return c.setCDROM(ctx, vmid, opts.Drive, opts.Volume)
}

// DetachISO ejects whatever is mounted in a guest's CD-ROM drive via
// POST /nodes/{node}/qemu/{vmid}/config, leaving drive configured as an
// empty CD-ROM ("none,media=cdrom") rather than removing the slot from
// the config — removing it entirely is Unlink's job, not this one's.
// Returns the config-update task's UPID.
//
// +proxmox:rbac:path=/vms/{vmid},method=POST,privs=VM.Config.CDROM,match=all
func (c *Client) DetachISO(ctx context.Context, vmid int, drive string) (string, error) {
	return c.setCDROM(ctx, vmid, drive, "none")
}

// setCDROM builds a single-drive Config for the bus/slot named by drive
// and applies it via UpdateConfigAsync. Shared by AttachISO and
// DetachISO, which differ only in the volume written to the slot.
func (c *Client) setCDROM(ctx context.Context, vmid int, drive, volume string) (string, error) {
	if drive == "" {
		return "", fmt.Errorf("qemu: cdrom drive is required")
	}

	prefix, index, ok := splitIndexedKey(drive)
	cfg := &Config{}
	if !ok || prefix == "virtio" || !setConfigDrive(cfg, prefix, index, Drive{File: volume, Media: "cdrom"}) {
		return "", fmt.Errorf("qemu: cdrom drive %q must be an ide, sata, or scsi slot", drive)
	}

	return c.UpdateConfigAsync(ctx, vmid, cfg)
}
