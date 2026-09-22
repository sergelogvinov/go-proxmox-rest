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

package vzdump

import "github.com/sergelogvinov/go-proxmox-rest/types"

// Compress, Mode, MailNotification, NotificationMode and
// PBSChangeDetectionMode are identical between this package and
// cluster/backup: both draw them from the same PVE::VZDump::Common
// confdesc. They live in the shared types package (see its doc comment)
// and are re-exported here as aliases so call sites keep reading as
// vzdump.Compress, vzdump.ModeSnapshot, ... . Options below is not
// shared — see the types package doc comment for why.
type (
	Compress               = types.Compress
	Mode                   = types.Mode
	MailNotification       = types.MailNotification
	NotificationMode       = types.NotificationMode
	PBSChangeDetectionMode = types.PBSChangeDetectionMode
)

const (
	CompressNone = types.CompressNone
	// CompressGzipLegacy is the legacy "1" alias for gzip.
	CompressGzipLegacy = types.CompressGzipLegacy
	CompressGzip       = types.CompressGzip
	CompressLZO        = types.CompressLZO
	CompressZstd       = types.CompressZstd

	ModeSnapshot = types.ModeSnapshot
	ModeSuspend  = types.ModeSuspend
	ModeStop     = types.ModeStop

	MailNotificationAlways  = types.MailNotificationAlways
	MailNotificationFailure = types.MailNotificationFailure

	NotificationModeAuto           = types.NotificationModeAuto
	NotificationModeLegacySendmail = types.NotificationModeLegacySendmail
	NotificationModeSystem         = types.NotificationModeSystem

	PBSChangeDetectionLegacy   = types.PBSChangeDetectionLegacy
	PBSChangeDetectionData     = types.PBSChangeDetectionData
	PBSChangeDetectionMetadata = types.PBSChangeDetectionMetadata
)

// Options holds the parameters shared by Client.Create
// (POST /nodes/{node}/vzdump) and the response shape of Client.Defaults
// (GET /nodes/{node}/vzdump/defaults) — Proxmox's own schema reuses the
// same property set for both. Fields whose format is itself a property
// string (Performance, Fleecing, PruneBackups) are kept as opaque
// strings rather than parsed further; see PVE::VZDump::Common's
// "backup-performance"/"backup-fleecing" formats and the root package's
// storage.Storage.PruneBackups for the same convention.
type Options struct {
	// VMID lists the guest(s) to back up. Empty (with All unset) is
	// invalid — Create requires either VMID or All.
	VMID []string `json:"vmid,omitempty" url:"vmid,omitempty"`
	// NodeFilter restricts a CLI invocation to running only when
	// executed on this node; meaningless (and effectively unusable)
	// over the REST API, which always runs on the URL's node. Kept for
	// schema fidelity.
	NodeFilter string `json:"node,omitempty" url:"node,omitempty"`
	// All backs up every known guest on the node.
	All bool `json:"all,omitempty" url:"all,omitempty"`
	// StdExcludes excludes temporary files and logs. Proxmox defaults
	// this to true.
	StdExcludes bool `json:"stdexcludes,omitempty" url:"stdexcludes,omitempty"`
	// Compress selects the dump file's compression.
	Compress Compress `json:"compress,omitempty" url:"compress,omitempty"`
	// Pigz uses pigz instead of gzip when > 0: 1 uses half the cores,
	// >1 uses that many threads.
	Pigz int `json:"pigz,omitempty" url:"pigz,omitempty"`
	// Zstd sets the zstd thread count; 0 uses half the available
	// cores.
	Zstd int `json:"zstd,omitempty" url:"zstd,omitempty"`
	// Quiet suppresses non-error output.
	Quiet bool `json:"quiet,omitempty" url:"quiet,omitempty"`
	// Mode selects the guest-quiescing strategy.
	Mode Mode `json:"mode,omitempty" url:"mode,omitempty"`
	// Exclude lists guest(s) to skip; implies All.
	Exclude []string `json:"exclude,omitempty" url:"exclude,omitempty"`
	// ExcludePath lists shell-glob file/directory patterns to exclude
	// from container backups.
	ExcludePath []string `json:"exclude-path,omitempty" url:"exclude-path,omitempty"`
	// Mailto lists email addresses/usernames to notify. Deprecated in
	// favor of the notification system.
	Mailto []string `json:"mailto,omitempty" url:"mailto,omitempty"`
	// MailNotification selects when a legacy-sendmail notification is
	// sent. Deprecated.
	MailNotification MailNotification `json:"mailnotification,omitempty" url:"mailnotification,omitempty"`
	// NotificationMode selects which notification system to use.
	NotificationMode NotificationMode `json:"notification-mode,omitempty" url:"notification-mode,omitempty"`
	// TmpDir is the directory for temporary files. Root only.
	TmpDir string `json:"tmpdir,omitempty" url:"tmpdir,omitempty"`
	// DumpDir is the directory for the resulting backup files. Root
	// only.
	DumpDir string `json:"dumpdir,omitempty" url:"dumpdir,omitempty"`
	// Script is a hook script path, run at various points during the
	// backup. Root only.
	Script string `json:"script,omitempty" url:"script,omitempty"`
	// Storage is the target storage for the resulting backup file(s).
	Storage string `json:"storage,omitempty" url:"storage,omitempty"`
	// Stop stops any already-running backup jobs on this node before
	// proceeding.
	Stop bool `json:"stop,omitempty" url:"stop,omitempty"`
	// BWLimit caps I/O bandwidth in KiB/s; 0 means unlimited.
	BWLimit int `json:"bwlimit,omitempty" url:"bwlimit,omitempty"`
	// Ionice sets the IO priority (0-8) when using the BFQ scheduler;
	// 8 means idle priority.
	Ionice int `json:"ionice,omitempty" url:"ionice,omitempty"`
	// Performance holds performance-related settings as a property
	// string, e.g. "max-workers=8".
	Performance string `json:"performance,omitempty" url:"performance,omitempty"`
	// Fleecing holds backup-fleecing settings (VM only) as a property
	// string, e.g. "enabled=1,storage=local".
	Fleecing string `json:"fleecing,omitempty" url:"fleecing,omitempty"`
	// LockWait is the maximum time (minutes) to wait for the global
	// backup lock.
	LockWait int `json:"lockwait,omitempty" url:"lockwait,omitempty"`
	// StopWait is the maximum time (minutes) to wait for a guest to
	// stop.
	StopWait int `json:"stopwait,omitempty" url:"stopwait,omitempty"`
	// PruneBackups is the retention rule string used instead of the
	// storage's configured retention, e.g. "keep-last=3,keep-daily=7".
	PruneBackups string `json:"prune-backups,omitempty" url:"prune-backups,omitempty"`
	// Remove prunes older backups per PruneBackups. Proxmox defaults
	// this to true.
	Remove bool `json:"remove,omitempty" url:"remove,omitempty"`
	// Pool backs up every guest in the given pool.
	Pool string `json:"pool,omitempty" url:"pool,omitempty"`
	// NotesTemplate is a template string for generating backup notes,
	// e.g. "{{guestname}}". Requires Storage.
	NotesTemplate string `json:"notes-template,omitempty" url:"notes-template,omitempty"`
	// Protected marks the resulting backup(s) as protected. Requires
	// Storage.
	Protected bool `json:"protected,omitempty" url:"protected,omitempty"`
	// PBSChangeDetectionMode selects the file-change detection mode
	// for container backups sent to a Proxmox Backup Server.
	PBSChangeDetectionMode PBSChangeDetectionMode `json:"pbs-change-detection-mode,omitempty" url:"pbs-change-detection-mode,omitempty"`

	// JobID sets the backup notification's "backup-job" metadata field.
	// Create only — never appears in a Defaults response. Root only.
	JobID string `json:"-" url:"job-id,omitempty,writeonly"`
}
