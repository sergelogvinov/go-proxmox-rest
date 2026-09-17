package qemu

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
	"github.com/sergelogvinov/go-proxmox-rest/types"
)

// MigrationType selects the transport used for a live migration's guest
// memory/state traffic.
type MigrationType string

const (
	// MigrationTypeSecure tunnels migration traffic over SSH. The
	// default.
	MigrationTypeSecure MigrationType = "secure"
	// MigrationTypeInsecure sends migration traffic unencrypted, for
	// higher throughput on a trusted, private network. Root only.
	MigrationTypeInsecure MigrationType = "insecure"
)

// BlockingHACause and BlockingHAResource are identical between nodes/qemu
// and nodes/lxc (see types/migrate.go's doc comment). They live in the
// shared types package and are re-exported here as aliases so existing
// call sites (qemu.BlockingHACause, qemu.BlockingHAResource, ...) keep
// working unchanged.
type (
	BlockingHACause    = types.BlockingHACause
	BlockingHAResource = types.BlockingHAResource
)

const (
	BlockingHACauseNodeAffinity     = types.BlockingHACauseNodeAffinity
	BlockingHACauseResourceAffinity = types.BlockingHACauseResourceAffinity
)

// MigratePrecondition describes whether/where a guest can currently be
// migrated, as returned by Client.MigratePrecondition
// (GET /nodes/{node}/qemu/{vmid}/migrate).
type MigratePrecondition struct {
	// Running is true if the guest is currently running (relevant
	// because it determines whether migration must be online).
	Running bool `json:"running,omitempty" url:"running,omitempty"`
	// AllowedNodes lists nodes the guest can be migrated to
	// unconditionally.
	AllowedNodes []string `json:"allowed_nodes,omitempty" url:"allowed_nodes,omitempty"`
	// NotAllowedNodes maps each node the guest cannot currently be
	// migrated to, to the reason(s) why.
	NotAllowedNodes map[string]NotAllowedNode `json:"not_allowed_nodes,omitempty" url:"not_allowed_nodes,omitempty"`
	// LocalDisks lists disks (including CD-ROM, unused, and
	// unreferenced ones) backed by node-local storage.
	LocalDisks []LocalDisk `json:"local_disks,omitempty" url:"local_disks,omitempty"`
	// LocalResources lists local (non-migratable) resources, e.g.
	// mapped PCI/USB devices, that block migration.
	LocalResources []string `json:"local_resources,omitempty" url:"local_resources,omitempty"`
	// MappedResources lists mapped resources (PCI/USB, ...) by name.
	//
	// Deprecated: use MappedResourceInfo instead.
	MappedResources []string `json:"mapped-resources,omitempty" url:"mapped-resources,omitempty"`
	// MappedResourceInfo holds the same mapped resources with
	// additional detail (e.g. whether each is live-migratable), as an
	// untyped object — Proxmox's own schema doesn't pin down its
	// shape.
	MappedResourceInfo map[string]any `json:"mapped-resource-info,omitempty" url:"mapped-resource-info,omitempty"`
	// HasDBusVMState is true if the source host supports migrating
	// additional VM state (e.g. conntrack entries).
	HasDBusVMState bool `json:"has-dbus-vmstate,omitempty" url:"has-dbus-vmstate,omitempty"`
	// DependentHAResources lists HA resource IDs ("<type>:<id>") that
	// will be migrated alongside the guest due to a positive HA
	// affinity rule.
	DependentHAResources []string `json:"dependent-ha-resources,omitempty" url:"dependent-ha-resources,omitempty"`
}

// NotAllowedNode explains why a guest cannot currently be migrated to
// one particular node.
type NotAllowedNode struct {
	// UnavailableStorages lists storages the guest needs that aren't
	// available on this node.
	UnavailableStorages []string `json:"unavailable_storages,omitempty" url:"unavailable_storages,omitempty"`
	// UnavailableResources lists mapped resources (PCI/USB, ...) with
	// no equivalent mapping on this node.
	UnavailableResources []string `json:"unavailable-resources,omitempty" url:"unavailable-resources,omitempty"`
	// BlockingHAResources lists HA resources preventing migration to
	// this node.
	BlockingHAResources []BlockingHAResource `json:"blocking-ha-resources,omitempty" url:"blocking-ha-resources,omitempty"`
}

// LocalDisk describes one guest disk backed by node-local storage, as
// listed in MigratePrecondition.LocalDisks.
type LocalDisk struct {
	// VolID is the disk's volume id.
	VolID string `json:"volid,omitempty" url:"volid,omitempty"`
	// Size is the disk's size in bytes.
	Size int64 `json:"size,omitempty" url:"size,omitempty"`
	// CDROM is true if the disk is a CD-ROM.
	CDROM bool `json:"cdrom,omitempty" url:"cdrom,omitempty"`
	// IsUnused is true if the disk is an unused (detached) volume.
	IsUnused bool `json:"is_unused,omitempty" url:"is_unused,omitempty"`
}

// MigratePrecondition retrieves whether/where a guest can currently be
// migrated via GET /nodes/{node}/qemu/{vmid}/migrate. target may be
// empty to check every node.
func (c *Client) MigratePrecondition(ctx context.Context, vmid int, target string) (*MigratePrecondition, error) {
	var p map[string]string
	if target != "" {
		p = map[string]string{"target": target}
	}

	pre := &MigratePrecondition{}
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/qemu/"+strconv.Itoa(vmid)+"/migrate", pre, p); err != nil {
		return nil, err
	}

	return pre, nil
}

// MigrateOptions holds the parameters for Client.Migrate
// (POST /nodes/{node}/qemu/{vmid}/migrate).
type MigrateOptions struct {
	// Target is the destination node. Required; must not be the
	// guest's current node.
	Target string `url:"target"`
	// Online performs a live migration if the guest is running.
	// Ignored (and forced false) if the guest is stopped.
	Online bool `url:"online,omitempty"`
	// Force allows migrating a guest that uses local devices (e.g.
	// mapped PCI/USB). Root only.
	Force bool `url:"force,omitempty"`
	// MigrationType overrides the migration traffic's transport. Root
	// only.
	MigrationType MigrationType `url:"migration_type,omitempty"`
	// MigrationNetwork is the CIDR of the (sub)network used for
	// migration traffic. Root only.
	MigrationNetwork string `url:"migration_network,omitempty"`
	// WithLocalDisks enables live storage migration for local disks.
	WithLocalDisks bool `url:"with-local-disks,omitempty"`
	// TargetStorage maps source to destination storage, e.g.
	// "local-lvm:local-zfs", a comma-separated list of such mappings,
	// or a single storage id to map every disk to.
	TargetStorage string `url:"targetstorage,omitempty"`
	// BWLimit overrides the migration's I/O bandwidth limit, in
	// KiB/s. Zero uses the datacenter/storage default.
	BWLimit int `url:"bwlimit,omitempty"`
	// WithConntrackState migrates conntrack entries for a running
	// guest.
	WithConntrackState bool `url:"with-conntrack-state,omitempty"`
}

// Migrate starts a guest migration via
// POST /nodes/{node}/qemu/{vmid}/migrate. Returns the migration task's
// UPID.
func (c *Client) Migrate(ctx context.Context, vmid int, opts *MigrateOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("qemu: migrate options are required")
	}
	if opts.Target == "" {
		return "", fmt.Errorf("qemu: migrate target is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+c.node+"/qemu/"+strconv.Itoa(vmid)+"/migrate", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
