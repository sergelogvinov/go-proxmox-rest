package agent

import (
	"context"
	"encoding/json"
)

// Ping checks that the guest agent is responding, via
// POST /nodes/{node}/qemu/{vmid}/agent/ping (guest-ping). Returns an
// error if the agent doesn't respond; there is no other result.
func (c *Client) Ping(ctx context.Context, vmid int) error {
	_, err := postResult[json.RawMessage](ctx, c.client, path(c.node, vmid, "ping"), nil)
	return err
}

// GetTime retrieves the guest's system time via
// GET /nodes/{node}/qemu/{vmid}/agent/get-time (guest-get-time).
// Returns nanoseconds since the Unix epoch, UTC.
func (c *Client) GetTime(ctx context.Context, vmid int) (int64, error) {
	return getResult[int64](ctx, c.client, path(c.node, vmid, "get-time"), nil)
}

// Info retrieves the guest agent's own version and supported command
// list via GET /nodes/{node}/qemu/{vmid}/agent/info (guest-info).
func (c *Client) Info(ctx context.Context, vmid int) (*Info, error) {
	return getResultPtr[Info](ctx, c.client, path(c.node, vmid, "info"))
}

// FSFreezeStatus retrieves the guest filesystems' current freeze state
// via POST /nodes/{node}/qemu/{vmid}/agent/fsfreeze-status
// (guest-fsfreeze-status).
func (c *Client) FSFreezeStatus(ctx context.Context, vmid int) (FSFreezeState, error) {
	return postResult[FSFreezeState](ctx, c.client, path(c.node, vmid, "fsfreeze-status"), nil)
}

// FSFreezeFreeze freezes every guest filesystem via
// POST /nodes/{node}/qemu/{vmid}/agent/fsfreeze-freeze
// (guest-fsfreeze-freeze). Returns the number of filesystems frozen.
func (c *Client) FSFreezeFreeze(ctx context.Context, vmid int) (int, error) {
	return postResult[int](ctx, c.client, path(c.node, vmid, "fsfreeze-freeze"), nil)
}

// FSFreezeThaw thaws every previously frozen guest filesystem via
// POST /nodes/{node}/qemu/{vmid}/agent/fsfreeze-thaw
// (guest-fsfreeze-thaw). Returns the number of filesystems thawed.
func (c *Client) FSFreezeThaw(ctx context.Context, vmid int) (int, error) {
	return postResult[int](ctx, c.client, path(c.node, vmid, "fsfreeze-thaw"), nil)
}

// FSTrim runs fstrim on every guest filesystem via
// POST /nodes/{node}/qemu/{vmid}/agent/fstrim (guest-fstrim).
func (c *Client) FSTrim(ctx context.Context, vmid int) (*FSTrimResult, error) {
	return postResultPtr[FSTrimResult](ctx, c.client, path(c.node, vmid, "fstrim"))
}

// NetworkGetInterfaces retrieves the guest's network interfaces via
// GET /nodes/{node}/qemu/{vmid}/agent/network-get-interfaces
// (guest-network-get-interfaces).
func (c *Client) NetworkGetInterfaces(ctx context.Context, vmid int) ([]NetworkInterface, error) {
	return getResult[[]NetworkInterface](ctx, c.client, path(c.node, vmid, "network-get-interfaces"), nil)
}

// GetVCPUs retrieves the guest's virtual CPU hotplug state via
// GET /nodes/{node}/qemu/{vmid}/agent/get-vcpus (guest-get-vcpus).
func (c *Client) GetVCPUs(ctx context.Context, vmid int) ([]VCPU, error) {
	return getResult[[]VCPU](ctx, c.client, path(c.node, vmid, "get-vcpus"), nil)
}

// GetFSInfo retrieves the guest's mounted filesystems via
// GET /nodes/{node}/qemu/{vmid}/agent/get-fsinfo (guest-get-fsinfo).
func (c *Client) GetFSInfo(ctx context.Context, vmid int) ([]FSInfo, error) {
	return getResult[[]FSInfo](ctx, c.client, path(c.node, vmid, "get-fsinfo"), nil)
}

// GetMemoryBlocks retrieves the guest's memory block hotplug state via
// GET /nodes/{node}/qemu/{vmid}/agent/get-memory-blocks
// (guest-get-memory-blocks).
func (c *Client) GetMemoryBlocks(ctx context.Context, vmid int) ([]MemoryBlock, error) {
	return getResult[[]MemoryBlock](ctx, c.client, path(c.node, vmid, "get-memory-blocks"), nil)
}

// GetMemoryBlockInfo retrieves the guest's memory block size via
// GET /nodes/{node}/qemu/{vmid}/agent/get-memory-block-info
// (guest-get-memory-block-info).
func (c *Client) GetMemoryBlockInfo(ctx context.Context, vmid int) (*MemoryBlockInfo, error) {
	return getResultPtr[MemoryBlockInfo](ctx, c.client, path(c.node, vmid, "get-memory-block-info"))
}

// SuspendHybrid suspends the guest to both RAM and disk via
// POST /nodes/{node}/qemu/{vmid}/agent/suspend-hybrid
// (guest-suspend-hybrid).
func (c *Client) SuspendHybrid(ctx context.Context, vmid int) error {
	_, err := postResult[json.RawMessage](ctx, c.client, path(c.node, vmid, "suspend-hybrid"), nil)
	return err
}

// SuspendRAM suspends the guest to RAM via
// POST /nodes/{node}/qemu/{vmid}/agent/suspend-ram
// (guest-suspend-ram).
func (c *Client) SuspendRAM(ctx context.Context, vmid int) error {
	_, err := postResult[json.RawMessage](ctx, c.client, path(c.node, vmid, "suspend-ram"), nil)
	return err
}

// SuspendDisk suspends the guest to disk (hibernate) via
// POST /nodes/{node}/qemu/{vmid}/agent/suspend-disk
// (guest-suspend-disk).
func (c *Client) SuspendDisk(ctx context.Context, vmid int) error {
	_, err := postResult[json.RawMessage](ctx, c.client, path(c.node, vmid, "suspend-disk"), nil)
	return err
}

// Shutdown requests a guest-initiated shutdown via
// POST /nodes/{node}/qemu/{vmid}/agent/shutdown (guest-shutdown).
func (c *Client) Shutdown(ctx context.Context, vmid int) error {
	_, err := postResult[json.RawMessage](ctx, c.client, path(c.node, vmid, "shutdown"), nil)
	return err
}

// GetHostname retrieves the guest's host name via
// GET /nodes/{node}/qemu/{vmid}/agent/get-host-name
// (guest-get-host-name).
func (c *Client) GetHostname(ctx context.Context, vmid int) (*HostnameInfo, error) {
	return getResultPtr[HostnameInfo](ctx, c.client, path(c.node, vmid, "get-host-name"))
}

// GetOSInfo retrieves the guest operating system's identification via
// GET /nodes/{node}/qemu/{vmid}/agent/get-osinfo (guest-get-osinfo).
func (c *Client) GetOSInfo(ctx context.Context, vmid int) (*OSInfo, error) {
	return getResultPtr[OSInfo](ctx, c.client, path(c.node, vmid, "get-osinfo"))
}

// GetUsers retrieves the guest's currently logged-in users via
// GET /nodes/{node}/qemu/{vmid}/agent/get-users (guest-get-users).
func (c *Client) GetUsers(ctx context.Context, vmid int) ([]User, error) {
	return getResult[[]User](ctx, c.client, path(c.node, vmid, "get-users"), nil)
}

// GetTimezone retrieves the guest's configured time zone via
// GET /nodes/{node}/qemu/{vmid}/agent/get-timezone
// (guest-get-timezone).
func (c *Client) GetTimezone(ctx context.Context, vmid int) (*Timezone, error) {
	return getResultPtr[Timezone](ctx, c.client, path(c.node, vmid, "get-timezone"))
}

// getResultPtr is getResult for a struct result, returning a pointer
// rather than a zero-valued struct on error.
func getResultPtr[T any](ctx context.Context, c Getter, p string) (*T, error) {
	v, err := getResult[T](ctx, c, p, nil)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// postResultPtr is postResult for a struct result, returning a pointer
// rather than a zero-valued struct on error.
func postResultPtr[T any](ctx context.Context, c Getter, p string) (*T, error) {
	v, err := postResult[T](ctx, c, p, nil)
	if err != nil {
		return nil, err
	}

	return &v, nil
}
