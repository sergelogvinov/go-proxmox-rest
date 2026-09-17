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

// vzdump.go: the vzdump backup enums Proxmox reuses verbatim between the
// cluster-wide scheduled-job API (cluster/backup, PVE::API2::Backup) and
// the per-node ad-hoc backup API (nodes/vzdump,
// PVE::API2::VZDump/PVE::VZDump::Common): both draw Compress/Mode/
// MailNotification/NotificationMode/PBSChangeDetectionMode from the same
// PVE::VZDump::Common confdesc. Unlike those enums, the two packages'
// Job/JobOptions (cluster/backup) and Options (nodes/vzdump) types are
// not shared: the cluster-wide job is a persistent scheduled config with
// its own CRUD shape, while the per-node type is a single merged
// Create-request/Defaults-response shape with property-string fields
// (Performance/Fleecing/PruneBackups) instead of cluster/backup's typed
// nested structs — different enough to keep local.

package types

// Compress is a backup's dump file compression.
type Compress string

const (
	// CompressNone disables compression.
	CompressNone Compress = "0"
	// CompressGzipLegacy is a deprecated alias for CompressGzip, kept for
	// backwards compatibility with old vzdump configs.
	CompressGzipLegacy Compress = "1"
	// CompressGzip compresses with gzip.
	CompressGzip Compress = "gzip"
	// CompressLZO compresses with lzo.
	CompressLZO Compress = "lzo"
	// CompressZstd compresses with zstd.
	CompressZstd Compress = "zstd"
)

// Mode is a backup's guest-quiescing strategy.
type Mode string

const (
	// ModeSnapshot backs up a live snapshot of the guest.
	ModeSnapshot Mode = "snapshot"
	// ModeSuspend suspends the guest for the duration of the backup.
	ModeSuspend Mode = "suspend"
	// ModeStop stops the guest for the duration of the backup.
	ModeStop Mode = "stop"
)

// MailNotification selects when a legacy-sendmail backup notification is
// sent. Deprecated by Proxmox in favor of the notification system
// (NotificationMode); kept for backward compatibility.
type MailNotification string

const (
	// MailNotificationAlways sends a mail after every backup run.
	MailNotificationAlways MailNotification = "always"
	// MailNotificationFailure sends a mail only when a guest backup fails.
	MailNotificationFailure MailNotification = "failure"
)

// NotificationMode selects which notification system a backup job uses.
type NotificationMode string

const (
	// NotificationModeAuto sends mail when Mailto is set, otherwise uses
	// the notification system.
	NotificationModeAuto NotificationMode = "auto"
	// NotificationModeLegacySendmail always uses Mailto/MailNotification
	// via the local sendmail command.
	NotificationModeLegacySendmail NotificationMode = "legacy-sendmail"
	// NotificationModeSystem always uses PVE's notification system.
	NotificationModeSystem NotificationMode = "notification-system"
)

// PBSChangeDetectionMode selects how container backups sent to a Proxmox
// Backup Server detect file changes.
type PBSChangeDetectionMode string

const (
	// PBSChangeDetectionLegacy uses the legacy change detection/encoding.
	PBSChangeDetectionLegacy PBSChangeDetectionMode = "legacy"
	// PBSChangeDetectionData detects changes from file data.
	PBSChangeDetectionData PBSChangeDetectionMode = "data"
	// PBSChangeDetectionMetadata detects changes from file metadata.
	PBSChangeDetectionMetadata PBSChangeDetectionMode = "metadata"
)
