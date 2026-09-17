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

package storage

// Storage describes a node's view of a storage: its enabled/active state
// and space usage, as returned by GET /nodes/{node}/storage (Client.List)
// and GET /nodes/{node}/storage/{storage}/status (Client.Status).
type Storage struct {
	// Storage is the storage identifier. List populates it directly from
	// the server; Status fills it in manually since Proxmox's own
	// response omits it (it's already the URL path segment).
	Storage string `json:"storage,omitempty" url:"storage,omitempty"`
	// Type is the storage plugin type, e.g. "dir", "zfs", "nfs", ...
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Content is the list of content types the storage can hold, e.g.
	// "images", "iso", "vztmpl".
	Content []string `json:"content,omitempty" url:"content,omitempty"`
	// Enabled is true unless the storage is disabled in the config.
	Enabled bool `json:"enabled,omitempty" url:"enabled,omitempty"`
	// Active is true if the storage is currently accessible on the
	// queried node.
	Active bool `json:"active,omitempty" url:"active,omitempty"`
	// Shared is true if the storage config marks it shared across
	// nodes.
	Shared bool `json:"shared,omitempty" url:"shared,omitempty"`
	// TotalSpace is the total storage space in bytes.
	TotalSpace int64 `json:"total,omitempty" url:"total,omitempty"`
	// UsedSpace is the used storage space in bytes.
	UsedSpace int64 `json:"used,omitempty" url:"used,omitempty"`
	// AvailableSpace is the available storage space in bytes.
	AvailableSpace int64 `json:"avail,omitempty" url:"avail,omitempty"`
	// UsedFraction is UsedSpace/TotalSpace (List only).
	UsedFraction float64 `json:"used_fraction,omitempty" url:"used_fraction,omitempty"`
	// SelectExisting is set when new volumes must be selected from
	// existing ones rather than newly created (List with
	// ListOptions.Format only).
	SelectExisting bool `json:"select_existing,omitempty" url:"select_existing,omitempty"`
	// Formats lists the storage's supported and default image format
	// (List with ListOptions.Format only).
	Formats *Formats `json:"formats,omitempty" url:"formats,omitempty"`
}

// Formats lists a storage's supported image formats and its default.
type Formats struct {
	Supported []string `json:"supported,omitempty" url:"supported,omitempty"`
	Default   string   `json:"default,omitempty"   url:"default,omitempty"`
}

// ListOptions filters the storages returned by Client.List. A nil
// *ListOptions (or the zero value) requests every storage the caller has
// access to, unfiltered.
type ListOptions struct {
	// Storage restricts the result to a single storage id.
	Storage string `url:"storage,omitempty"`
	// Content restricts the result to storages supporting every given
	// content type, e.g. []string{"images", "iso"}.
	Content []string `url:"content,omitempty"`
	// Enabled restricts the result to storages that are enabled.
	Enabled bool `url:"enabled,omitempty"`
	// Target, if set, restricts the result to shared storages whose
	// content is accessible on both the queried node and Target.
	Target string `url:"target,omitempty"`
	// Format, if set, additionally populates Storage.SelectExisting and
	// Storage.Formats on each returned entry.
	Format bool `url:"format,omitempty"`
}

// Volume describes a storage volume, as returned by
// GET /nodes/{node}/storage/{storage}/content (Client.Content().List) and
// GET /nodes/{node}/storage/{storage}/content/{volume}
// (Client.Content().Get, which populates only Path/Size/Used/Format/
// Notes/Protected — VolID and the rest are List-only).
type Volume struct {
	// VolID is the volume identifier, e.g. "local:iso/debian.iso"
	// (List only).
	VolID string `json:"volid,omitempty" url:"volid,omitempty"`
	// VMID is the volume's owning guest, when applicable.
	VMID int `json:"vmid,omitempty" url:"vmid,omitempty"`
	// Parent is the parent volume's id, for a linked clone.
	Parent string `json:"parent,omitempty" url:"parent,omitempty"`
	// Format is the format identifier, e.g. "raw", "qcow2", "subvol",
	// "iso", "tgz".
	Format string `json:"format,omitempty" url:"format,omitempty"`
	// Size is the volume size in bytes.
	Size int64 `json:"size,omitempty" url:"size,omitempty"`
	// ApproximateSize is an upper-bound size in bytes, present instead
	// of Size for storages where the exact size is impractical to
	// determine.
	ApproximateSize int64 `json:"approximate-size,omitempty" url:"approximate-size,omitempty"`
	// Used is the used space in bytes; most storage plugins do not
	// report anything meaningful here.
	Used int64 `json:"used,omitempty" url:"used,omitempty"`
	// CTime is the volume's creation time (unix timestamp).
	CTime int64 `json:"ctime,omitempty" url:"ctime,omitempty"`
	// Notes holds optional free-form notes. List returns only the
	// first line of a multi-line value; Get returns the full value.
	Notes string `json:"notes,omitempty" url:"notes,omitempty"`
	// Encrypted is the fingerprint if the whole backup is encrypted, or
	// "1" if just marked encrypted (Proxmox Backup Server storages
	// only).
	Encrypted string `json:"encrypted,omitempty" url:"encrypted,omitempty"`
	// Verification is the last backup verification result (Proxmox
	// Backup Server storages only).
	Verification *Verification `json:"verification,omitempty" url:"verification,omitempty"`
	// Protected marks the volume as protected against deletion/pruning;
	// currently only meaningful for backups.
	Protected bool `json:"protected,omitempty" url:"protected,omitempty"`
	// Path is the volume's filesystem path (Get only).
	Path string `json:"path,omitempty" url:"path,omitempty"`
}

// Verification is a backup volume's last verification result.
type Verification struct {
	State string `json:"state,omitempty" url:"state,omitempty"`
	UPID  string `json:"upid,omitempty"  url:"upid,omitempty"`
}

// ContentListOptions filters the volumes returned by
// Client.Content().List. A nil *ContentListOptions (or the zero value)
// requests every volume unfiltered.
type ContentListOptions struct {
	// Content restricts the result to volumes of this content type,
	// e.g. "iso", "vztmpl", "backup", "images".
	Content string `url:"content,omitempty"`
	// VMID restricts the result to volumes owned by this guest.
	VMID int `url:"vmid,omitempty"`
}

// CreateVolumeOptions holds the parameters for allocating a new disk
// image via Client.Content().Create
// (POST /nodes/{node}/storage/{storage}/content).
type CreateVolumeOptions struct {
	// Filename is the name of the file to create. Required.
	Filename string `url:"filename"`
	// VMID is the owning guest. Required.
	VMID int `url:"vmid"`
	// Size is the image size in kilobytes, with an optional "M"
	// (megabyte) or "G" (gigabyte) suffix, e.g. "1048576" or "1G".
	// Required.
	Size string `url:"size"`
	// Format is the image format, e.g. "raw", "qcow2", "vmdk". Optional
	// if Filename already carries one of those extensions.
	Format string `url:"format,omitempty"`
}

// UpdateVolumeOptions holds the parameters for
// Client.Content().Update (PUT /nodes/{node}/storage/{storage}/content/
// {volume}). Pointer fields are sent only when non-nil.
//
// Both fields are rejected by most storage plugins (e.g. DirPlugin) for
// anything other than a backup-type volume — Proxmox dies with "only
// backups can have notes" / "only backups support attribute
// 'protected'" for a disk image, ISO, or template.
type UpdateVolumeOptions struct {
	// Notes replaces the volume's notes. Backup volumes only.
	Notes *string `url:"notes"`
	// Protected marks or unmarks the volume as protected against
	// deletion/pruning. Backup volumes only.
	Protected *bool `url:"protected"`
}

// GuestType narrows a prune-backups operation to one guest type.
type GuestType string

const (
	GuestTypeQemu GuestType = "qemu"
	GuestTypeLXC  GuestType = "lxc"
)

// PruneMark describes what a prune-backups dry run would do with a given
// backup volume.
type PruneMark string

const (
	// PruneMarkKeep means the backup would be kept.
	PruneMarkKeep PruneMark = "keep"
	// PruneMarkRemove means the backup would be removed.
	PruneMarkRemove PruneMark = "remove"
	// PruneMarkProtected means the backup is protected and would never
	// be removed.
	PruneMarkProtected PruneMark = "protected"
	// PruneMarkRenamed means the backup doesn't use the standard naming
	// scheme and would never be removed.
	PruneMarkRenamed PruneMark = "renamed"
)

// PruneOptions holds the parameters shared by
// Client.PruneBackups().DryRun (GET) and Client.PruneBackups().Delete
// (DELETE).
type PruneOptions struct {
	// PruneBackups is the retention rule string, e.g.
	// "keep-last=3,keep-daily=7", used instead of the storage's
	// configured retention. Required for both DryRun and Delete unless
	// the storage itself has a "prune-backups" retention configured —
	// Proxmox rejects both calls with "no prune-backups options
	// configured for storage '...'" otherwise.
	PruneBackups string `url:"prune-backups,omitempty"`
	// Type restricts the operation to backups of this guest type.
	Type GuestType `url:"type,omitempty"`
	// VMID restricts the operation to backups of this guest.
	VMID int `url:"vmid,omitempty"`
}

// PruneEntry describes a single backup volume's prune-backups outcome,
// as returned by Client.PruneBackups().DryRun.
type PruneEntry struct {
	// VolID is the backup volume's id.
	VolID string `json:"volid,omitempty" url:"volid,omitempty"`
	// CTime is the backup's creation time (unix timestamp).
	CTime int64 `json:"ctime,omitempty" url:"ctime,omitempty"`
	// Mark is whether the backup would be kept or removed.
	Mark PruneMark `json:"mark,omitempty" url:"mark,omitempty"`
	// Type is the guest type, one of "qemu", "lxc", "openvz", or
	// "unknown".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// VMID is the guest the backup belongs to, when known.
	VMID int `json:"vmid,omitempty" url:"vmid,omitempty"`
}
