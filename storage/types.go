package storage

import (
	"fmt"
	"strings"
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
type CreateOptions struct {
	// ID is the storage identifier, e.g. "local-zfs". Required.
	ID string
	// Type is the storage plugin type, e.g. "dir", "zfs", "nfs", ...
	// Required.
	Type string
	// Content is the list of content types the storage can hold,
	// e.g. "images", "iso", "vztmpl", "backup".
	Content []string
	// Nodes is the list of nodes the storage is available on.
	// Empty means all nodes.
	Nodes []string
	// Shared marks the storage as shared across nodes.
	Shared bool
	// Disabled marks the storage as disabled.
	Disabled bool
	// Enable marks the storage as enabled.
	Enable bool
	// MaxFiles is the maximum number of backup files per VM.
	MaxFiles int
	// PruneBackups is the prune-backups configuration string,
	// e.g. "keep-last=7,keep-daily=7".
	PruneBackups string
	// Comment is the storage description.
	Comment string
	// Username is the CIFS/Synology username.
	Username string
	// Password is the CIFS/Synology password (write-only).
	Password string
	// Domain is the CIFS domain.
	Domain string
	// Path is the local filesystem path (dir/zfs plugin types).
	Path string
	// Server is the remote server address (nfs/cifs/iscsi plugin types).
	Server string
	// Export is the NFS export path.
	Export string
	// Pool is the ZFS pool name (zfs plugin type).
	Pool string
	// BlockSize is the block size (zfs plugin type).
	BlockSize string
	// FSName is the CIFS share name.
	FSName string
	// Portal is the iSCSI portal address.
	Portal string
	// Target is the iSCSI target.
	Target string
	// VGName is the LVM volume group name.
	VGName string
	// ThinPool is the LVM-thin pool name.
	ThinPool string
	// Datastore is the Synology datastore name.
	Datastore string
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

	params := map[string]string{
		"storage": o.ID,
		"type":    o.Type,
	}

	if len(o.Content) > 0 {
		params["content"] = strings.Join(o.Content, ",")
	}
	if len(o.Nodes) > 0 {
		params["nodes"] = strings.Join(o.Nodes, ",")
	}
	if o.Shared {
		params["shared"] = "1"
	}
	if o.Disabled {
		params["disable"] = "1"
	}
	if o.Enable {
		params["enable"] = "1"
	}
	if o.MaxFiles > 0 {
		params["maxfiles"] = fmt.Sprintf("%d", o.MaxFiles)
	}
	if o.PruneBackups != "" {
		params["prune-backups"] = o.PruneBackups
	}
	if o.Comment != "" {
		params["comment"] = o.Comment
	}
	if o.Username != "" {
		params["username"] = o.Username
	}
	if o.Password != "" {
		params["password"] = o.Password
	}
	if o.Domain != "" {
		params["domain"] = o.Domain
	}
	if o.Path != "" {
		params["path"] = o.Path
	}
	if o.Server != "" {
		params["server"] = o.Server
	}
	if o.Export != "" {
		params["export"] = o.Export
	}
	if o.Pool != "" {
		params["pool"] = o.Pool
	}
	if o.BlockSize != "" {
		params["blocksize"] = o.BlockSize
	}
	if o.FSName != "" {
		params["fsname"] = o.FSName
	}
	if o.Portal != "" {
		params["portal"] = o.Portal
	}
	if o.Target != "" {
		params["target"] = o.Target
	}
	if o.VGName != "" {
		params["vgname"] = o.VGName
	}
	if o.ThinPool != "" {
		params["thinpool"] = o.ThinPool
	}
	if o.Datastore != "" {
		params["datastore"] = o.Datastore
	}

	return params, nil
}

// UpdateOptions holds the parameters for PUT /storage/{storage}.
// Only the fields that are set are sent; use pointers to distinguish
// "unset" from "clear".
type UpdateOptions struct {
	// Content is the list of content types the storage can hold.
	Content []string
	// Nodes is the list of nodes the storage is available on.
	Nodes []string
	// Shared marks the storage as shared across nodes.
	Shared *bool
	// Disabled marks the storage as disabled.
	Disabled *bool
	// Enable marks the storage as enabled.
	Enable *bool
	// MaxFiles is the maximum number of backup files per VM.
	MaxFiles *int
	// PruneBackups is the prune-backups configuration string.
	PruneBackups *string
	// Comment is the storage description.
	Comment *string
	// Username is the CIFS/Synology username.
	Username *string
	// Password is the CIFS/Synology password (write-only).
	Password *string
	// Domain is the CIFS domain.
	Domain *string
	// Path is the local filesystem path (dir/zfs plugin types).
	Path *string
	// Server is the remote server address (nfs/cifs/iscsi plugin types).
	Server *string
	// Export is the NFS export path.
	Export *string
	// Pool is the ZFS pool name (zfs plugin type).
	Pool *string
	// BlockSize is the block size (zfs plugin type).
	BlockSize *string
	// FSName is the CIFS share name.
	FSName *string
	// Portal is the iSCSI portal address.
	Portal *string
	// Target is the iSCSI target.
	Target *string
	// VGName is the LVM volume group name.
	VGName *string
	// ThinPool is the LVM-thin pool name.
	ThinPool *string
	// Datastore is the Synology datastore name.
	Datastore *string
	// Delete is the list of fields to remove from the configuration,
	// e.g. "maxfiles", "prune-backups".
	Delete []string
	// Digest prevents changes if the current configuration has changed
	// in between (value from GET /storage/{storage}).
	Digest string
}

// encode converts the options to form parameters.
func (o *UpdateOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("storage: update options are required")
	}

	params := map[string]string{}

	if len(o.Content) > 0 {
		params["content"] = strings.Join(o.Content, ",")
	}
	if len(o.Nodes) > 0 {
		params["nodes"] = strings.Join(o.Nodes, ",")
	}
	if o.Shared != nil {
		params["shared"] = boolToInt(o.Shared)
	}
	if o.Disabled != nil {
		params["disable"] = boolToInt(o.Disabled)
	}
	if o.Enable != nil {
		params["enable"] = boolToInt(o.Enable)
	}
	if o.MaxFiles != nil {
		params["maxfiles"] = fmt.Sprintf("%d", *o.MaxFiles)
	}
	if o.PruneBackups != nil {
		params["prune-backups"] = *o.PruneBackups
	}
	if o.Comment != nil {
		params["comment"] = *o.Comment
	}
	if o.Username != nil {
		params["username"] = *o.Username
	}
	if o.Password != nil {
		params["password"] = *o.Password
	}
	if o.Domain != nil {
		params["domain"] = *o.Domain
	}
	if o.Path != nil {
		params["path"] = *o.Path
	}
	if o.Server != nil {
		params["server"] = *o.Server
	}
	if o.Export != nil {
		params["export"] = *o.Export
	}
	if o.Pool != nil {
		params["pool"] = *o.Pool
	}
	if o.BlockSize != nil {
		params["blocksize"] = *o.BlockSize
	}
	if o.FSName != nil {
		params["fsname"] = *o.FSName
	}
	if o.Portal != nil {
		params["portal"] = *o.Portal
	}
	if o.Target != nil {
		params["target"] = *o.Target
	}
	if o.VGName != nil {
		params["vgname"] = *o.VGName
	}
	if o.ThinPool != nil {
		params["thinpool"] = *o.ThinPool
	}
	if o.Datastore != nil {
		params["datastore"] = *o.Datastore
	}
	if len(o.Delete) > 0 {
		params["delete"] = strings.Join(o.Delete, ",")
	}
	if o.Digest != "" {
		params["digest"] = o.Digest
	}

	return params, nil
}

// boolToInt converts a *bool to the "0"/"1" form used by the Proxmox API.
func boolToInt(b *bool) string {
	if *b {
		return "1"
	}
	return "0"
}
