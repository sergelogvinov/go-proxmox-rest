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
