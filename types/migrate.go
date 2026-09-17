package types

// migrate.go: BlockingHACause and BlockingHAResource are identical
// between nodes/qemu and nodes/lxc — both back onto the same HA-affinity
// migration-blocking check, nested inside each package's own
// (otherwise-diverged) NotAllowedNode/MigratePrecondition shapes. See
// nodes/qemu/migrate.go and nodes/lxc/migrate.go: the surrounding
// precondition response differs enough between the two (local disks,
// local/mapped resources and has-dbus-vmstate are QEMU-only; even the
// shared fields are named with underscores for QEMU and hyphens for LXC)
// that only this inner shape is genuinely identical and worth sharing.

// BlockingHACause explains why a HA resource blocks a migration.
type BlockingHACause string

const (
	BlockingHACauseNodeAffinity     BlockingHACause = "node-affinity"
	BlockingHACauseResourceAffinity BlockingHACause = "resource-affinity"
)

// BlockingHAResource is a single HA resource blocking a migration.
type BlockingHAResource struct {
	// SID is the blocking HA resource's id.
	SID string `json:"sid,omitempty" url:"sid,omitempty"`
	// Cause is why it blocks the migration.
	Cause BlockingHACause `json:"cause,omitempty" url:"cause,omitempty"`
}
