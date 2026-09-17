package lxc

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
	"github.com/sergelogvinov/go-proxmox-rest/types"
)

// BlockingHACause and BlockingHAResource are identical between nodes/lxc
// and nodes/qemu (see types/migrate.go's doc comment). They live in the
// shared types package and are re-exported here as aliases so call sites
// read as lxc.BlockingHACause, lxc.BlockingHAResource, ... .
type (
	BlockingHACause    = types.BlockingHACause
	BlockingHAResource = types.BlockingHAResource
)

const (
	BlockingHACauseNodeAffinity     = types.BlockingHACauseNodeAffinity
	BlockingHACauseResourceAffinity = types.BlockingHACauseResourceAffinity
)

// MigratePrecondition describes whether/where a container can currently
// be migrated, as returned by Client.MigratePrecondition
// (GET /nodes/{node}/lxc/{vmid}/migrate). Unlike nodes/qemu's version,
// LXC has no local-disk/local-resource/mapped-resource inventory to
// report — a container's disks are always node-local storage, and there
// is no PCI/USB passthrough to check.
type MigratePrecondition struct {
	// Running is true if the container is currently running (relevant
	// because it determines whether migration must be online or use
	// restart mode).
	Running bool `json:"running,omitempty" url:"running,omitempty"`
	// AllowedNodes lists nodes the container can be migrated to
	// unconditionally.
	AllowedNodes []string `json:"allowed-nodes,omitempty" url:"allowed-nodes,omitempty"`
	// NotAllowedNodes maps each node the container cannot currently be
	// migrated to, to the reason(s) why.
	NotAllowedNodes map[string]NotAllowedNode `json:"not-allowed-nodes,omitempty" url:"not-allowed-nodes,omitempty"`
	// DependentHAResources lists HA resource IDs ("<type>:<id>") that
	// will be migrated alongside the container due to a positive HA
	// affinity rule.
	DependentHAResources []string `json:"dependent-ha-resources,omitempty" url:"dependent-ha-resources,omitempty"`
}

// NotAllowedNode explains why a container cannot currently be migrated
// to one particular node.
type NotAllowedNode struct {
	// BlockingHAResources lists HA resources preventing migration to
	// this node.
	BlockingHAResources []BlockingHAResource `json:"blocking-ha-resources,omitempty" url:"blocking-ha-resources,omitempty"`
}

// MigratePrecondition retrieves whether/where a container can currently
// be migrated via GET /nodes/{node}/lxc/{vmid}/migrate. target may be
// empty to check every node.
func (c *Client) MigratePrecondition(ctx context.Context, node string, vmid int, target string) (*MigratePrecondition, error) {
	var p map[string]string
	if target != "" {
		p = map[string]string{"target": target}
	}

	pre := &MigratePrecondition{}
	if err := c.client.Get(ctx, "/nodes/"+node+"/lxc/"+strconv.Itoa(vmid)+"/migrate", pre, p); err != nil {
		return nil, err
	}

	return pre, nil
}

// MigrateOptions holds the parameters for Client.Migrate
// (POST /nodes/{node}/lxc/{vmid}/migrate).
type MigrateOptions struct {
	// Target is the destination node. Required; must not be the
	// container's current node.
	Target string `url:"target"`
	// Online performs a live migration if the container is running.
	Online bool `url:"online,omitempty"`
	// Restart uses restart migration: the container is shut down,
	// migrated, and started again on the target node.
	Restart bool `url:"restart,omitempty"`
	// TargetStorage maps source to destination storage, e.g.
	// "local-lvm:local-zfs", a comma-separated list of such mappings, or
	// "1" to map each source storage to itself.
	TargetStorage string `url:"target-storage,omitempty"`
	// BWLimit overrides the migration's I/O bandwidth limit, in KiB/s.
	// Zero uses the datacenter/storage default.
	BWLimit int `url:"bwlimit,omitempty"`
	// Timeout is the maximum number of seconds to wait for shutdown
	// when using Restart. Proxmox defaults to 180.
	Timeout int `url:"timeout,omitempty"`
}

// Migrate starts a container migration via
// POST /nodes/{node}/lxc/{vmid}/migrate. Returns the migration task's
// UPID.
func (c *Client) Migrate(ctx context.Context, node string, vmid int, opts *MigrateOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("lxc: migrate options are required")
	}
	if opts.Target == "" {
		return "", fmt.Errorf("lxc: migrate target is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+node+"/lxc/"+strconv.Itoa(vmid)+"/migrate", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
