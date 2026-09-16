package qemu

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Snapshot describes one guest snapshot, as returned by
// snapshotResource.List. Proxmox includes a pseudo-entry with
// Name == "current" representing the guest's live state (not an actual
// snapshot) — SnapTime/VMState/Parent are meaningless for it, while
// Digest/Running are populated only for it.
type Snapshot struct {
	// Name is the snapshot's identifier, or "current" for the
	// live-state pseudo-entry.
	Name string `json:"name,omitempty" url:"name,omitempty"`
	// Description is the snapshot's description ("You are here!" for
	// the "current" pseudo-entry).
	Description string `json:"description,omitempty" url:"description,omitempty"`
	// VMState is true if the snapshot includes RAM.
	VMState bool `json:"vmstate,omitempty" url:"vmstate,omitempty"`
	// SnapTime is the snapshot's creation time (unix timestamp).
	SnapTime int64 `json:"snaptime,omitempty" url:"snaptime,omitempty"`
	// Parent is the parent snapshot's identifier, if any.
	Parent string `json:"parent,omitempty" url:"parent,omitempty"`
	// SnapState is the snapshot's in-progress state
	// ("delete"/"prepare"), when a snapshot operation on it hasn't
	// completed cleanly.
	SnapState string `json:"snapstate,omitempty" url:"snapstate,omitempty"`
	// Digest is the current configuration's digest. Populated only for
	// the "current" pseudo-entry.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
	// Running is true if the guest is currently running. Populated
	// only for the "current" pseudo-entry.
	Running bool `json:"running,omitempty" url:"running,omitempty"`
}

// CreateSnapshotOptions holds the parameters for
// snapshotResource.Create (POST .../snapshot).
type CreateSnapshotOptions struct {
	// Snapname is the new snapshot's name. Required; "current" and
	// "pending" (case-insensitive) are reserved and rejected by
	// Proxmox.
	Snapname string `url:"snapname"`
	// VMState saves the guest's RAM state too, allowing Rollback to
	// resume a running guest exactly where it left off.
	VMState bool `url:"vmstate,omitempty"`
	// Description is a free-form description or comment.
	Description string `url:"description,omitempty"`
}

// snapshotResource provides access to
// /nodes/{node}/qemu/{vmid}/snapshot.
type snapshotResource struct {
	client Getter
}

// snapshotPath builds the .../snapshot[/{snapname}[/{sub}]] URL for the
// given node and vmid.
func snapshotPath(node string, vmid int, snapname, sub string) string {
	p := "/nodes/" + node + "/qemu/" + strconv.Itoa(vmid) + "/snapshot"
	if snapname != "" {
		p += "/" + snapname
	}
	if sub != "" {
		p += "/" + sub
	}

	return p
}

// List retrieves every snapshot via
// GET /nodes/{node}/qemu/{vmid}/snapshot, including the "current"
// live-state pseudo-entry.
func (r *snapshotResource) List(ctx context.Context, node string, vmid int) ([]Snapshot, error) {
	var snapshots []Snapshot
	if err := r.client.Get(ctx, snapshotPath(node, vmid, "", ""), &snapshots, nil); err != nil {
		return nil, err
	}

	return snapshots, nil
}

// Create takes a new snapshot via
// POST /nodes/{node}/qemu/{vmid}/snapshot. Returns the snapshot task's
// UPID.
func (r *snapshotResource) Create(ctx context.Context, node string, vmid int, opts *CreateSnapshotOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("qemu: snapshot create options are required")
	}
	if opts.Snapname == "" {
		return "", fmt.Errorf("qemu: snapshot create snapname is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := r.client.Create(ctx, snapshotPath(node, vmid, "", ""), &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}

// GetConfig retrieves the guest configuration as it was captured in a
// snapshot via GET /nodes/{node}/qemu/{vmid}/snapshot/{snapname}/config.
// This is the same Config shape returned by Client.Config, since
// Proxmox stores a full configuration copy inside each snapshot.
func (r *snapshotResource) GetConfig(ctx context.Context, node string, vmid int, snapname string) (*Config, error) {
	var raw map[string]json.RawMessage
	if err := r.client.Get(ctx, snapshotPath(node, vmid, snapname, "config"), &raw, nil); err != nil {
		return nil, err
	}

	return decodeConfig(raw)
}

// UpdateConfig replaces a snapshot's description via
// PUT /nodes/{node}/qemu/{vmid}/snapshot/{snapname}/config — the only
// snapshot metadata field Proxmox allows changing after the fact.
func (r *snapshotResource) UpdateConfig(ctx context.Context, node string, vmid int, snapname, description string) error {
	p := map[string]string{"description": description}

	return r.client.Update(ctx, snapshotPath(node, vmid, snapname, "config"), nil, p)
}

// Rollback restores the guest to a snapshot's state via
// POST /nodes/{node}/qemu/{vmid}/snapshot/{snapname}/rollback. start
// requests the guest be started afterwards if it wasn't already
// (Proxmox always starts it automatically when the snapshot includes
// RAM). Returns the rollback task's UPID.
func (r *snapshotResource) Rollback(ctx context.Context, node string, vmid int, snapname string, start bool) (string, error) {
	var p map[string]string
	if start {
		p = map[string]string{"start": "1"}
	}

	var upid string
	if err := r.client.Create(ctx, snapshotPath(node, vmid, snapname, "rollback"), &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}

// Delete removes a snapshot via
// DELETE /nodes/{node}/qemu/{vmid}/snapshot/{snapname}. force removes
// the snapshot from the configuration even if deleting its underlying
// disk snapshots fails. Returns the deletion task's UPID.
func (r *snapshotResource) Delete(ctx context.Context, node string, vmid int, snapname string, force bool) (string, error) {
	var p map[string]string
	if force {
		p = map[string]string{"force": "1"}
	}

	var upid string
	if err := r.client.Delete(ctx, snapshotPath(node, vmid, snapname, ""), &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
