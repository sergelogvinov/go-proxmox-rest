package storage

import (
	"fmt"

	"github.com/sergelogvinov/proxmox/go-proxmox-rest/internal/params"
)

// Storage describes a storage as returned by GET /storage and
// GET /storage/{storage}.
type Storage struct {
	// Storage is the storage identifier, e.g. "local".
	Storage string `json:"storage,omitempty"`
	// Type is the storage plugin type, e.g. "dir", "zfs", "nfs", ...
	Type string `json:"type,omitempty"`
	// Content is the comma-separated list of content types the storage
	// can hold, e.g. "images,iso,vztmpl".
	Content string `json:"content,omitempty"`
	// Shared is 1 if the storage is shared across nodes.
	Shared int `json:"shared,omitempty"`
	// Enabled is 1 if the storage is enabled.
	Enabled int `json:"enabled,omitempty"`
	// Path is the local filesystem path (dir/zfs plugin types).
	Path string `json:"path,omitempty"`
	// Server is the remote server address (nfs/cifs/iscsi plugin types).
	Server string `json:"server,omitempty"`
	// Export is the NFS export path.
	Export string `json:"export,omitempty"`
	// Pool is the ZFS pool name (zfs plugin type).
	Pool string `json:"pool,omitempty"`
	// BlockSize is the block size (zfs plugin type).
	BlockSize string `json:"blocksize,omitempty"`
	// FSName is the CIFS share name.
	FSName string `json:"fsname,omitempty"`
	// Portal is the iSCSI portal address.
	Portal string `json:"portal,omitempty"`
	// Target is the iSCSI target.
	Target string `json:"target,omitempty"`
	// VGName is the LVM volume group name.
	VGName string `json:"vgname,omitempty"`
	// ThinPool is the LVM-thin pool name.
	ThinPool string `json:"thinpool,omitempty"`
	// Datastore is the Synology datastore name.
	Datastore string `json:"datastore,omitempty"`
	// Username is the CIFS/Synology username.
	Username string `json:"username,omitempty"`
	// Domain is the CIFS domain.
	Domain string `json:"domain,omitempty"`
	// MaxFiles is the maximum number of backup files per VM.
	MaxFiles int `json:"maxfiles,omitempty"`
	// PruneBackups is the prune-backups configuration string.
	PruneBackups string `json:"prune-backups,omitempty"`
	// Nodes is the comma-separated list of nodes this storage is
	// available on ("all" or empty means all nodes).
	Nodes string `json:"nodes,omitempty"`
	// AUS is 1 if the storage is used for VM backups (deprecated).
	AUS int `json:"aus,omitempty"`
	// SpaceUsed is the used space in bytes (GET /storage/{storage} only).
	SpaceUsed int64 `json:"used,omitempty"`
	// TotalSpace is the total space in bytes (GET /storage/{storage} only).
	TotalSpace int64 `json:"total,omitempty"`
	// AvailableSpace is the available space in bytes
	// (GET /storage/{storage} only).
	AvailableSpace int64 `json:"avail,omitempty"`
	// Active is 1 if the storage is active on the queried node
	// (GET /storage/{storage} only).
	Active int `json:"active,omitempty"`
}

// CreateOptions holds the parameters for POST /storage.
//
// Fields are encoded to form parameters via the `url` struct tags:
// zero values are omitted, []string is comma-joined.
type CreateOptions struct {
	// ID is the storage identifier, e.g. "local-zfs". Required.
	ID string `url:"storage"`
	// Type is the storage plugin type, e.g. "dir", "zfs", "nfs", ...
	// Required.
	Type string `url:"type"`
	// Content is the list of content types the storage can hold,
	// e.g. "images", "iso", "vztmpl", "backup".
	Content []string `url:"content"`
	// Nodes is the list of nodes the storage is available on.
	// Empty means all nodes.
	Nodes []string `url:"nodes"`
	// Shared marks the storage as shared across nodes.
	Shared bool `url:"shared"`
	// Disabled marks the storage as disabled.
	Disabled bool `url:"disable"`
	// Enable marks the storage as enabled.
	Enable bool `url:"enable"`
	// MaxFiles is the maximum number of backup files per VM.
	MaxFiles int `url:"maxfiles"`
	// PruneBackups is the prune-backups configuration string,
	// e.g. "keep-last=7,keep-daily=7".
	PruneBackups string `url:"prune-backups"`
	// Comment is the storage description.
	Comment string `url:"comment"`
	// Username is the CIFS/Synology username.
	Username string `url:"username"`
	// Password is the CIFS/Synology password (write-only).
	Password string `url:"password"`
	// Domain is the CIFS domain.
	Domain string `url:"domain"`
	// Path is the local filesystem path (dir/zfs plugin types).
	Path string `url:"path"`
	// Server is the remote server address (nfs/cifs/iscsi plugin types).
	Server string `url:"server"`
	// Server2 is the secondary NFS server address.
	Server2 string `url:"server2"`
	// Export is the NFS export path.
	Export string `url:"export"`
	// Pool is the ZFS pool name (zfs plugin type).
	Pool string `url:"pool"`
	// BlockSize is the block size (zfs plugin type).
	BlockSize string `url:"blocksize"`
	// FSName is the CIFS share name.
	FSName string `url:"fsname"`
	// Portal is the iSCSI portal address.
	Portal string `url:"portal"`
	// Target is the iSCSI target.
	Target string `url:"target"`
	// VGName is the LVM volume group name.
	VGName string `url:"vgname"`
	// ThinPool is the LVM-thin pool name.
	ThinPool string `url:"thinpool"`
	// Datastore is the Synology datastore name.
	Datastore string `url:"datastore"`
}

// encode converts the options to form parameters.
func (o *CreateOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("storage: create options are required")
	}
	if o.ID == "" {
		return nil, fmt.Errorf("storage: id is required")
	}
	if o.Type == "" {
		return nil, fmt.Errorf("storage: type is required")
	}

	return params.Encode(o)
}

// UpdateOptions holds the parameters for PUT /storage/{storage}.
//
// Pointer fields are always sent when non-nil (even when zero-valued),
// which is how "clear a field" is expressed; non-pointer fields are
// omitted when zero.
type UpdateOptions struct {
	// Content is the list of content types the storage can hold.
	Content []string `url:"content"`
	// Nodes is the list of nodes the storage is available on.
	Nodes []string `url:"nodes"`
	// Shared marks the storage as shared across nodes.
	Shared *bool `url:"shared"`
	// Disabled marks the storage as disabled.
	Disabled *bool `url:"disable"`
	// Enable marks the storage as enabled.
	Enable *bool `url:"enable"`
	// MaxFiles is the maximum number of backup files per VM.
	MaxFiles *int `url:"maxfiles"`
	// PruneBackups is the prune-backups configuration string.
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
	// Delete is the list of fields to remove from the configuration,
	// e.g. "maxfiles", "prune-backups".
	Delete []string `url:"delete"`
	// Digest prevents changes if the current configuration has changed
	// in between (value from GET /storage/{storage}).
	Digest string `url:"digest"`
}

// encode converts the options to form parameters.
func (o *UpdateOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("storage: update options are required")
	}

	return params.Encode(o)
}
