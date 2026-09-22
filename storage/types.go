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

import (
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Storage describes a storage, both as returned by GET /storage and
// GET /storage/{storage} (List, Get) and as accepted by POST /storage and
// PUT /storage/{storage} (Create, Update) — the read and write shapes are
// merged into one type since Proxmox's config read-back uses the same
// field names the write side accepts.
//
// ID and Type are required by Create; Update ignores Type (the storage id
// is already part of the URL and its plugin type cannot change afterwards)
// — sending it anyway is rejected by PUT's strict parameter validation, so
// don't build Update's *Storage by simply copying a fetched one that has
// Type set. The same applies to every field commented "Create only" below:
// Proxmox's PUT schema does not accept them at all.
//
// Pointer fields are sent by Create/Update iff non-nil (even when the
// pointed-to value is zero/empty) — that is how Update expresses "clear
// this field". Password is write-only (Proxmox never echoes credentials
// back on a read). Delete and Digest are meaningful to Update only.
type Storage struct {
	// ID is the storage identifier, e.g. "local-zfs". Required by Create.
	ID string `json:"storage,omitempty" url:"storage,omitempty"`
	// Type is the storage plugin type, e.g. "dir", "zfs", "nfs", ...
	// Required by Create; not settable on Update.
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Content is the list of content types the storage can hold,
	// e.g. "images", "iso", "vztmpl", "backup".
	Content []string `json:"content,omitempty" url:"content,omitempty"`
	// Nodes is the list of nodes the storage is available on.
	// Empty means all nodes.
	Nodes []string `json:"nodes,omitempty" url:"nodes,omitempty"`
	// Shared marks the storage as shared across nodes.
	Shared bool `json:"shared,omitempty" url:"shared,omitempty"`
	// Disabled marks the storage as disabled.
	Disabled *bool `json:"disable,omitempty" url:"disable,omitempty"`
	// PruneBackups is the prune-backups configuration string,
	// e.g. "keep-last=7,keep-daily=7".
	PruneBackups *string `json:"prune-backups,omitempty" url:"prune-backups,omitempty"`
	// MaxProtectedBackups is the maximal number of protected backups per
	// guest. Use -1 for unlimited.
	MaxProtectedBackups *int `json:"max-protected-backups,omitempty" url:"max-protected-backups,omitempty"`
	// BWLimit sets the I/O bandwidth limit for various operations,
	// in KiB/s.
	BWLimit *string `json:"bwlimit,omitempty" url:"bwlimit,omitempty"`
	// Namespace is the storage namespace.
	Namespace *string `json:"namespace,omitempty" url:"namespace,omitempty"`

	// Username is the CIFS/RBD/Synology username.
	Username *string `json:"username,omitempty" url:"username,omitempty"`
	// Password is the CIFS/Synology password. Write-only: never returned
	// by a read.
	Password *string `json:"-" url:"password,writeonly"`
	// Domain is the CIFS domain.
	Domain *string `json:"domain,omitempty" url:"domain,omitempty"`
	// EncryptionKey is the encryption key. Use "autogen" to generate one
	// automatically without a passphrase.
	EncryptionKey *string `json:"encryption-key,omitempty" url:"encryption-key,omitempty"`
	// MasterPubkey is a base64-encoded, PEM-formatted public RSA key used
	// to encrypt a copy of EncryptionKey, added to each encrypted backup.
	MasterPubkey *string `json:"master-pubkey,omitempty" url:"master-pubkey,omitempty"`
	// Keyring is the client keyring contents (for external Ceph clusters).
	Keyring *string `json:"keyring,omitempty" url:"keyring,omitempty"`
	// AuthSupported is the authentication method (pbs plugin type).
	// Create only.
	AuthSupported *string `json:"authsupported,omitempty" url:"authsupported,omitempty"`

	// Path is the local filesystem path (dir/zfs/btrfs plugin types).
	// Create only.
	Path *string `json:"path,omitempty" url:"path,omitempty"`
	// MountPoint is the mount point.
	MountPoint *string `json:"mountpoint,omitempty" url:"mountpoint,omitempty"`
	// Subdir is the subdir to mount.
	Subdir *string `json:"subdir,omitempty" url:"subdir,omitempty"`
	// ContentDirs overrides the default content type directories,
	// e.g. "images=/mnt/images".
	ContentDirs *string `json:"content-dirs,omitempty" url:"content-dirs,omitempty"`
	// CreateBasePath creates the base directory if it doesn't exist
	// (dir plugin type).
	CreateBasePath *bool `json:"create-base-path,omitempty" url:"create-base-path,omitempty"`
	// CreateSubdirs populates the directory with the default structure
	// (dir plugin type).
	CreateSubdirs *bool `json:"create-subdirs,omitempty" url:"create-subdirs,omitempty"`
	// Mkdir creates the directory if it doesn't exist and populates it
	// with default sub-dirs.
	//
	// Deprecated: use CreateBasePath and CreateSubdirs instead.
	Mkdir *bool `json:"mkdir,omitempty" url:"mkdir,omitempty"`
	// IsMountpoint marks the given path as an externally managed mount
	// point; the storage is considered offline if it is not mounted.
	// A "yes"/"no" value is a shortcut for using the target path.
	IsMountpoint *string `json:"is_mountpoint,omitempty" url:"is_mountpoint,omitempty"`
	// Sparse enables sparse volumes.
	Sparse *bool `json:"sparse,omitempty" url:"sparse,omitempty"`
	// Preallocation is the preallocation mode for raw and qcow2 images:
	// "off", "metadata", "falloc" or "full".
	Preallocation *string `json:"preallocation,omitempty" url:"preallocation,omitempty"`
	// Format is the default image format.
	Format *string `json:"format,omitempty" url:"format,omitempty"`
	// NoCow sets the NOCOW flag on files (btrfs plugin type). Disables
	// data checksumming, allowing direct I/O.
	NoCow *bool `json:"nocow,omitempty" url:"nocow,omitempty"`

	// Server is the remote server address (nfs/cifs/iscsi/esxi plugin
	// types).
	Server *string `json:"server,omitempty" url:"server,omitempty"`
	// Port connects to the storage on this port instead of the default
	// one (e.g. with PBS or ESXi). For NFS/CIFS, use MountOptions instead.
	Port *int `json:"port,omitempty" url:"port,omitempty"`
	// MonHost is the list of monitor IP addresses (for external Ceph
	// clusters).
	MonHost *string `json:"monhost,omitempty" url:"monhost,omitempty"`
	// MountOptions are the NFS/CIFS mount options (see "man nfs" or
	// "man mount.cifs").
	MountOptions *string `json:"options,omitempty" url:"options,omitempty"`
	// SkipCertVerification disables TLS certificate verification; only
	// enable on fully trusted networks.
	SkipCertVerification *bool `json:"skip-cert-verification,omitempty" url:"skip-cert-verification,omitempty"`
	// SMBVersion is the SMB protocol version, e.g. "default", "2.0",
	// "3.0", "3.11".
	SMBVersion *string `json:"smbversion,omitempty" url:"smbversion,omitempty"`
	// Share is the CIFS share name. Create only.
	Share *string `json:"share,omitempty" url:"share,omitempty"`
	// Fingerprint is the certificate SHA-256 fingerprint.
	Fingerprint *string `json:"fingerprint,omitempty" url:"fingerprint,omitempty"`

	// Export is the NFS export path. Create only.
	Export *string `json:"export,omitempty" url:"export,omitempty"`

	// Pool is the ZFS/RBD pool name.
	Pool *string `json:"pool,omitempty" url:"pool,omitempty"`
	// BlockSize is the block size (zfs plugin type).
	BlockSize *string `json:"blocksize,omitempty" url:"blocksize,omitempty"`
	// DataPool is the data pool, for erasure coding only (rbd plugin
	// type).
	DataPool *string `json:"data-pool,omitempty" url:"data-pool,omitempty"`
	// CephFSName is the Ceph filesystem name (cephfs plugin type).
	CephFSName *string `json:"fs-name,omitempty" url:"fs-name,omitempty"`
	// Fuse mounts CephFS through FUSE (cephfs plugin type).
	Fuse *bool `json:"fuse,omitempty" url:"fuse,omitempty"`
	// KRBD always accesses rbd through the krbd kernel module (rbd
	// plugin type).
	KRBD *bool `json:"krbd,omitempty" url:"krbd,omitempty"`
	// ZFSBasePath is the base path where to look for the created ZFS
	// block devices. Usually "/dev/zvol".
	ZFSBasePath *string `json:"zfs-base-path,omitempty" url:"zfs-base-path,omitempty"`
	// SnapshotAsVolumeChain enables support for creating storage-vendor
	// agnostic snapshots through volume backing-chains.
	SnapshotAsVolumeChain *bool `json:"snapshot-as-volume-chain,omitempty" url:"snapshot-as-volume-chain,omitempty"`

	// Portal is the iSCSI portal address. Create only.
	Portal *string `json:"portal,omitempty" url:"portal,omitempty"`
	// Target is the iSCSI target. Create only.
	Target *string `json:"target,omitempty" url:"target,omitempty"`
	// ISCSIProvider is the iSCSI provider. Create only.
	ISCSIProvider *string `json:"iscsiprovider,omitempty" url:"iscsiprovider,omitempty"`
	// VGName is the LVM volume group name. Create only.
	VGName *string `json:"vgname,omitempty" url:"vgname,omitempty"`
	// ThinPool is the LVM-thin pool name. Create only.
	ThinPool *string `json:"thinpool,omitempty" url:"thinpool,omitempty"`
	// SafeRemove zeroes out data when removing LVs (lvm plugin type).
	SafeRemove *bool `json:"saferemove,omitempty" url:"saferemove,omitempty"`
	// SafeRemoveStepSize is the wipe step size in MiB, capped to the
	// maximum supported by the storage.
	SafeRemoveStepSize *int `json:"saferemove-stepsize,omitempty" url:"saferemove-stepsize,omitempty"`
	// SafeRemoveThroughput is the wipe throughput (cstream -t parameter
	// value).
	SafeRemoveThroughput *string `json:"saferemove_throughput,omitempty" url:"saferemove_throughput,omitempty"`
	// TaggedOnly only lists logical volumes tagged with "pve-vm-ID"
	// (lvm plugin type).
	TaggedOnly *bool `json:"tagged_only,omitempty" url:"tagged_only,omitempty"`
	// NoWriteCache disables write caching on the target (iscsi plugin
	// type).
	NoWriteCache *bool `json:"nowritecache,omitempty" url:"nowritecache,omitempty"`
	// ComstarHostGroup is the host group for comstar views.
	ComstarHostGroup *string `json:"comstar_hg,omitempty" url:"comstar_hg,omitempty"`
	// ComstarTargetGroup is the target group for comstar views.
	ComstarTargetGroup *string `json:"comstar_tg,omitempty" url:"comstar_tg,omitempty"`
	// LioTpg is the target portal group for Linux LIO targets.
	LioTpg *string `json:"lio_tpg,omitempty" url:"lio_tpg,omitempty"`

	// Datastore is the Proxmox Backup Server/Synology datastore name.
	// Create only.
	Datastore *string `json:"datastore,omitempty" url:"datastore,omitempty"`

	// Base is the base volume for linked clones; automatically activated.
	// Create only.
	Base *string `json:"base,omitempty" url:"base,omitempty"`

	// Delete is the list of fields to remove from the configuration.
	// Write-only: never appears in a read response. Update only,
	// e.g. "prune-backups", "comment".
	Delete []string `json:"-" url:"delete,writeonly"`
	// Digest is the configuration's digest: returned by a read, and
	// accepted by Update to abort the change if the configuration has
	// been modified in between.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// encode converts the storage fields to form parameters for Create/Update.
func (o *Storage) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("storage: options are required")
	}

	return params.Encode(o)
}
