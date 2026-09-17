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

// Storage describes a storage as returned by GET /storage and
// GET /storage/{storage}.
type Storage struct {
	// Storage is the storage identifier, e.g. "local".
	Storage string `json:"storage,omitempty" url:"storage,omitempty"`
	// Type is the storage plugin type, e.g. "dir", "zfs", "nfs", ...
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// Content is the list of content types the storage can hold,
	// e.g. "images", "iso", "vztmpl".
	Content []string `json:"content,omitempty" url:"content,omitempty"`
	// Shared is 1 if the storage is shared across nodes.
	Shared int `json:"shared,omitempty" url:"shared,omitempty"`
	// Enabled is 1 if the storage is enabled.
	Enabled int `json:"enabled,omitempty" url:"enabled,omitempty"`
	// Path is the local filesystem path (dir/zfs plugin types).
	Path string `json:"path,omitempty" url:"path,omitempty"`
	// Server is the remote server address (nfs/cifs/iscsi plugin types).
	Server string `json:"server,omitempty" url:"server,omitempty"`
	// Export is the NFS export path.
	Export string `json:"export,omitempty" url:"export,omitempty"`
	// Pool is the ZFS pool name (zfs plugin type).
	Pool string `json:"pool,omitempty" url:"pool,omitempty"`
	// BlockSize is the block size (zfs plugin type).
	BlockSize string `json:"blocksize,omitempty" url:"blocksize,omitempty"`
	// FSName is the CIFS share name.
	FSName string `json:"fsname,omitempty" url:"fsname,omitempty"`
	// Portal is the iSCSI portal address.
	Portal string `json:"portal,omitempty" url:"portal,omitempty"`
	// Target is the iSCSI target.
	Target string `json:"target,omitempty" url:"target,omitempty"`
	// VGName is the LVM volume group name.
	VGName string `json:"vgname,omitempty" url:"vgname,omitempty"`
	// ThinPool is the LVM-thin pool name.
	ThinPool string `json:"thinpool,omitempty" url:"thinpool,omitempty"`
	// Datastore is the Synology datastore name.
	Datastore string `json:"datastore,omitempty" url:"datastore,omitempty"`
	// Username is the CIFS/Synology username.
	Username string `json:"username,omitempty" url:"username,omitempty"`
	// Domain is the CIFS domain.
	Domain string `json:"domain,omitempty" url:"domain,omitempty"`
	// MaxFiles is the maximum number of backup files per VM.
	MaxFiles int `json:"maxfiles,omitempty" url:"maxfiles,omitempty"`
	// PruneBackups is the prune-backups configuration string.
	PruneBackups string `json:"prune-backups,omitempty" url:"prune-backups,omitempty"`
	// Nodes is the comma-separated list of nodes this storage is
	// available on ("all" or empty means all nodes).
	Nodes string `json:"nodes,omitempty" url:"nodes,omitempty"`
	// SpaceUsed is the used space in bytes (GET /storage/{storage} only).
	SpaceUsed int64 `json:"used,omitempty" url:"used,omitempty"`
	// TotalSpace is the total space in bytes (GET /storage/{storage} only).
	TotalSpace int64 `json:"total,omitempty" url:"total,omitempty"`
	// AvailableSpace is the available space in bytes
	// (GET /storage/{storage} only).
	AvailableSpace int64 `json:"avail,omitempty" url:"avail,omitempty"`
	// Active is 1 if the storage is active on the queried node
	// (GET /storage/{storage} only).
	Active int `json:"active,omitempty" url:"active,omitempty"`
}

// Options holds the write parameters shared by POST /storage (Create) and
// PUT /storage/{storage} (Update).
//
// ID and Type are required by Create; Update ignores them (the storage id
// is already part of the URL and its type cannot change afterwards).
// Pointer fields are always sent when non-nil (even when zero/empty),
// which is how Update expresses "clear this field" — Create simply sends
// whatever is set. Delete and Digest are meaningful to Update only.
type Options struct {
	// ID is the storage identifier, e.g. "local-zfs". Required by Create.
	ID string `url:"storage"`
	// Type is the storage plugin type, e.g. "dir", "zfs", "nfs", ...
	// Required by Create; not settable on Update.
	Type string `url:"type"`
	// Content is the list of content types the storage can hold,
	// e.g. "images", "iso", "vztmpl", "backup".
	Content []string `url:"content"`
	// Nodes is the list of nodes the storage is available on.
	// Empty means all nodes.
	Nodes []string `url:"nodes"`
	// Shared marks the storage as shared across nodes.
	Shared *bool `url:"shared"`
	// Disabled marks the storage as disabled.
	Disabled *bool `url:"disable"`
	// Enable marks the storage as enabled.
	Enable *bool `url:"enable"`
	// MaxFiles is the maximum number of backup files per VM.
	MaxFiles *int `url:"maxfiles"`
	// PruneBackups is the prune-backups configuration string,
	// e.g. "keep-last=7,keep-daily=7".
	PruneBackups *string `url:"prune-backups"`
	// Comment is the storage description.
	Comment *string `url:"comment"`
	// Username is the CIFS/Synology username.
	Username *string `url:"username"`
	// Password is the CIFS/Synology password (write-only).
	Password *string `url:"password"`
	// Domain is the CIFS domain.
	Domain *string `url:"domain"`
	// Path is the local filesystem path (dir/zfs plugin types).
	Path *string `url:"path"`
	// Server is the remote server address (nfs/cifs/iscsi plugin types).
	Server *string `url:"server"`
	// Server2 is the secondary NFS server address.
	Server2 *string `url:"server2"`
	// Export is the NFS export path.
	Export *string `url:"export"`
	// Pool is the ZFS pool name (zfs plugin type).
	Pool *string `url:"pool"`
	// BlockSize is the block size (zfs plugin type).
	BlockSize *string `url:"blocksize"`
	// FSName is the CIFS share name.
	FSName *string `url:"fsname"`
	// Portal is the iSCSI portal address.
	Portal *string `url:"portal"`
	// Target is the iSCSI target.
	Target *string `url:"target"`
	// VGName is the LVM volume group name.
	VGName *string `url:"vgname"`
	// ThinPool is the LVM-thin pool name.
	ThinPool *string `url:"thinpool"`
	// Datastore is the Synology datastore name.
	Datastore *string `url:"datastore"`
	// Delete is the list of fields to remove from the configuration
	// (Update only), e.g. "maxfiles", "prune-backups".
	Delete []string `url:"delete"`
	// Digest prevents changes if the current configuration has changed
	// in between (value from GET /storage/{storage}). Update only.
	Digest string `url:"digest"`
}

// encode converts the options to form parameters.
func (o *Options) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("storage: options are required")
	}

	return params.Encode(o)
}
