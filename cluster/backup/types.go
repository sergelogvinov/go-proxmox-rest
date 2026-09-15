package backup

import (
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Compress is the vzdump dump file compression algorithm.
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

// Mode is the vzdump backup mode.
type Mode string

const (
	// ModeSnapshot backs up a live snapshot of the guest.
	ModeSnapshot Mode = "snapshot"
	// ModeSuspend suspends the guest for the duration of the backup.
	ModeSuspend Mode = "suspend"
	// ModeStop stops the guest for the duration of the backup.
	ModeStop Mode = "stop"
)

// MailNotification controls when a job sends the deprecated legacy-sendmail
// notification.
type MailNotification string

const (
	// MailNotificationAlways sends a mail after every backup run.
	MailNotificationAlways MailNotification = "always"
	// MailNotificationFailure sends a mail only when a guest backup fails.
	MailNotificationFailure MailNotification = "failure"
)

// NotificationMode selects which notification system a job uses.
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

// PBSChangeDetectionMode is the PBS file-change detection mode used for
// container backups.
type PBSChangeDetectionMode string

const (
	// PBSChangeDetectionLegacy uses the legacy change detection/encoding.
	PBSChangeDetectionLegacy PBSChangeDetectionMode = "legacy"
	// PBSChangeDetectionData detects changes from file data.
	PBSChangeDetectionData PBSChangeDetectionMode = "data"
	// PBSChangeDetectionMetadata detects changes from file metadata.
	PBSChangeDetectionMetadata PBSChangeDetectionMode = "metadata"
)

// Fleecing describes a job's backup-fleecing settings, as returned (as a
// nested object) by GET /cluster/backup and GET /cluster/backup/{id}. On
// write, the same settings are sent as the opaque property-string
// JobOptions.Fleecing (e.g. "enabled=1,storage=local").
type Fleecing struct {
	// Enabled turns on backup fleecing.
	Enabled bool `json:"enabled,omitempty" url:"enabled,omitempty"`
	// Storage is the storage used to hold fleecing images.
	Storage string `json:"storage,omitempty" url:"storage,omitempty"`
}

// Performance describes a job's performance-related settings, as returned
// (as a nested object) by GET /cluster/backup and GET /cluster/backup/{id}.
// On write, the same settings are sent as the opaque property-string
// JobOptions.Performance (e.g. "max-workers=4").
type Performance struct {
	// MaxWorkers is, for VM guests, the number of concurrent IO workers.
	MaxWorkers int `json:"max-workers,omitempty" url:"max-workers,omitempty"`
	// PBSEntriesMax is, for container backups sent to PBS, the maximum
	// number of entries kept in memory at once.
	PBSEntriesMax int `json:"pbs-entries-max,omitempty" url:"pbs-entries-max,omitempty"`
}

// PruneBackups describes a job's retention settings, as returned (as a
// nested object) by GET /cluster/backup and GET /cluster/backup/{id}. On
// write, the same settings are sent as the opaque property-string
// JobOptions.PruneBackups (e.g. "keep-last=7,keep-daily=7").
type PruneBackups struct {
	// KeepAll keeps every backup; mutually exclusive with the other
	// Keep* fields.
	KeepAll bool `json:"keep-all,omitempty" url:"keep-all,omitempty"`
	// KeepLast keeps the last N backups.
	KeepLast int `json:"keep-last,omitempty" url:"keep-last,omitempty"`
	// KeepHourly keeps one backup for each of the last N hours.
	KeepHourly int `json:"keep-hourly,omitempty" url:"keep-hourly,omitempty"`
	// KeepDaily keeps one backup for each of the last N days.
	KeepDaily int `json:"keep-daily,omitempty" url:"keep-daily,omitempty"`
	// KeepWeekly keeps one backup for each of the last N weeks.
	KeepWeekly int `json:"keep-weekly,omitempty" url:"keep-weekly,omitempty"`
	// KeepMonthly keeps one backup for each of the last N months.
	KeepMonthly int `json:"keep-monthly,omitempty" url:"keep-monthly,omitempty"`
	// KeepYearly keeps one backup for each of the last N years.
	KeepYearly int `json:"keep-yearly,omitempty" url:"keep-yearly,omitempty"`
}

// Job describes a vzdump backup job as returned by GET /cluster/backup and
// GET /cluster/backup/{id}.
//
// Proxmox also accepts a deprecated pair of parameters (starttime + dow) as
// an alternative to Schedule when creating/updating a job, converting them
// to a Schedule internally; every GET response already reflects that
// conversion, so this client only ever exposes Schedule.
type Job struct {
	// ID is the job identifier.
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// Schedule is the backup schedule, a subset of systemd calendar
	// events, e.g. "sat 03:00".
	Schedule string `json:"schedule,omitempty" url:"schedule,omitempty"`
	// Enabled is false if the job is disabled.
	Enabled bool `json:"enabled,omitempty" url:"enabled,omitempty"`
	// RepeatMissed runs the job as soon as possible if it was missed
	// while the scheduler was not running.
	RepeatMissed bool `json:"repeat-missed,omitempty" url:"repeat-missed,omitempty"`
	// Comment is the job description.
	Comment string `json:"comment,omitempty" url:"comment,omitempty"`
	// NextRun is the UNIX timestamp of the next scheduled run. Only
	// populated by List, not by Get.
	NextRun int64 `json:"next-run,omitempty" url:"next-run,omitempty"`

	// VMID is the list of guest IDs to back up.
	VMID []string `json:"vmid,omitempty" url:"vmid,omitempty"`
	// Node restricts the job to run only on this node.
	Node string `json:"node,omitempty" url:"node,omitempty"`
	// All backs up every known guest on the host.
	All bool `json:"all,omitempty" url:"all,omitempty"`
	// Pool backs up every guest in this pool.
	Pool string `json:"pool,omitempty" url:"pool,omitempty"`
	// Exclude lists guest IDs to skip; implies All.
	Exclude []string `json:"exclude,omitempty" url:"exclude,omitempty"`
	// ExcludePath lists shell globs of files/directories to skip.
	ExcludePath []string `json:"exclude-path,omitempty" url:"exclude-path,omitempty"`

	// StdExcludes excludes temporary files and logs.
	StdExcludes bool `json:"stdexcludes,omitempty" url:"stdexcludes,omitempty"`
	// Compress is the dump file compression algorithm.
	Compress Compress `json:"compress,omitempty" url:"compress,omitempty"`
	// Pigz enables the pigz gzip compressor with N threads (0: half the
	// available cores).
	Pigz int `json:"pigz,omitempty" url:"pigz,omitempty"`
	// Zstd is the number of zstd compression threads (0: half the
	// available cores).
	Zstd int `json:"zstd,omitempty" url:"zstd,omitempty"`
	// Quiet suppresses output.
	Quiet bool `json:"quiet,omitempty" url:"quiet,omitempty"`
	// Mode is the backup mode.
	Mode Mode `json:"mode,omitempty" url:"mode,omitempty"`

	// Mailto is a deprecated comma-separated list of email addresses/users
	// to notify. Use notification targets/matchers instead.
	Mailto string `json:"mailto,omitempty" url:"mailto,omitempty"`
	// MailNotification is deprecated: when to send the legacy-sendmail
	// notification.
	MailNotification MailNotification `json:"mailnotification,omitempty" url:"mailnotification,omitempty"`
	// NotificationMode selects which notification system the job uses.
	NotificationMode NotificationMode `json:"notification-mode,omitempty" url:"notification-mode,omitempty"`

	// TmpDir stores temporary files in this directory.
	TmpDir string `json:"tmpdir,omitempty" url:"tmpdir,omitempty"`
	// DumpDir stores resulting files in this directory.
	DumpDir string `json:"dumpdir,omitempty" url:"dumpdir,omitempty"`
	// Script is a hook script run at various backup stages.
	Script string `json:"script,omitempty" url:"script,omitempty"`
	// Storage stores the resulting file on this storage.
	Storage string `json:"storage,omitempty" url:"storage,omitempty"`
	// Stop stops any running backup jobs on the host before starting.
	Stop bool `json:"stop,omitempty" url:"stop,omitempty"`
	// BWLimit limits I/O bandwidth in KiB/s (0: unlimited).
	BWLimit int `json:"bwlimit,omitempty" url:"bwlimit,omitempty"`
	// IONice is the IO priority used with the BFQ scheduler.
	IONice int `json:"ionice,omitempty" url:"ionice,omitempty"`
	// LockWait is the maximum time, in minutes, to wait for the global
	// lock.
	LockWait int `json:"lockwait,omitempty" url:"lockwait,omitempty"`
	// StopWait is the maximum time, in minutes, to wait for a guest to
	// stop.
	StopWait int `json:"stopwait,omitempty" url:"stopwait,omitempty"`

	// Performance holds other performance-related settings.
	Performance *Performance `json:"performance,omitempty" url:"performance,omitempty"`
	// Fleecing holds backup-fleecing settings (VM guests only).
	Fleecing *Fleecing `json:"fleecing,omitempty" url:"fleecing,omitempty"`
	// PruneBackups holds the retention settings applied after a
	// successful run, overriding the storage's own configuration.
	PruneBackups *PruneBackups `json:"prune-backups,omitempty" url:"prune-backups,omitempty"`
	// Remove prunes older backups according to PruneBackups.
	Remove bool `json:"remove,omitempty" url:"remove,omitempty"`

	// NotesTemplate is a template string used to generate backup notes;
	// requires Storage.
	NotesTemplate string `json:"notes-template,omitempty" url:"notes-template,omitempty"`
	// Protected marks the resulting backup(s) as protected; requires
	// Storage.
	Protected bool `json:"protected,omitempty" url:"protected,omitempty"`
	// PBSChangeDetectionMode is the PBS file-change detection mode used
	// for container backups.
	PBSChangeDetectionMode PBSChangeDetectionMode `json:"pbs-change-detection-mode,omitempty" url:"pbs-change-detection-mode,omitempty"`
}

// JobOptions holds the write parameters shared by POST /cluster/backup
// (Create) and PUT /cluster/backup/{id} (Update).
//
// ID is required by Create; Update ignores it (the job id is already part
// of the URL). Schedule is required by Create — Proxmox alternatively
// accepts the deprecated starttime/dow pair, which this client does not
// expose. Pointer fields are always sent when non-nil (even when
// zero/empty), which is how Update expresses "clear this field" — Create
// simply sends whatever is set. Delete is meaningful to Update only.
type JobOptions struct {
	// ID is the job identifier. Required by Create; not settable on
	// Update.
	ID string `url:"id"`
	// Schedule is the backup schedule, a subset of systemd calendar
	// events, e.g. "sat 03:00". Required by Create.
	Schedule *string `url:"schedule"`
	// Enabled enables/disables the job.
	Enabled *bool `url:"enabled"`
	// RepeatMissed runs the job as soon as possible if it was missed
	// while the scheduler was not running.
	RepeatMissed *bool `url:"repeat-missed"`
	// Comment is the job description.
	Comment *string `url:"comment"`

	// VMID is the list of guest IDs to back up. Conflicts with All,
	// Exclude and Pool. One of VMID, All or Pool is required by Create.
	VMID []string `url:"vmid"`
	// Node restricts the job to run only on this node.
	Node *string `url:"node"`
	// All backs up every known guest on the host. Conflicts with VMID.
	All *bool `url:"all"`
	// Pool backs up every guest in this pool. Conflicts with VMID.
	Pool *string `url:"pool"`
	// Exclude lists guest IDs to skip; implies All. Conflicts with VMID.
	Exclude []string `url:"exclude"`
	// ExcludePath lists shell globs of files/directories to skip.
	ExcludePath []string `url:"exclude-path"`

	// StdExcludes excludes temporary files and logs.
	StdExcludes *bool `url:"stdexcludes"`
	// Compress is the dump file compression algorithm.
	Compress *Compress `url:"compress"`
	// Pigz enables the pigz gzip compressor with N threads (0: half the
	// available cores).
	Pigz *int `url:"pigz"`
	// Zstd is the number of zstd compression threads (0: half the
	// available cores).
	Zstd *int `url:"zstd"`
	// Quiet suppresses output.
	Quiet *bool `url:"quiet"`
	// Mode is the backup mode.
	Mode *Mode `url:"mode"`

	// Mailto is a deprecated comma-separated list of email addresses/users
	// to notify. Use notification targets/matchers instead.
	Mailto *string `url:"mailto"`
	// MailNotification is deprecated: when to send the legacy-sendmail
	// notification.
	MailNotification *MailNotification `url:"mailnotification"`
	// NotificationMode selects which notification system the job uses.
	NotificationMode *NotificationMode `url:"notification-mode"`

	// TmpDir stores temporary files in this directory. Restricted to
	// root@pam.
	TmpDir *string `url:"tmpdir"`
	// DumpDir stores resulting files in this directory. Restricted to
	// root@pam.
	DumpDir *string `url:"dumpdir"`
	// Script is a hook script run at various backup stages. Restricted
	// to root@pam.
	Script *string `url:"script"`
	// Storage stores the resulting file on this storage.
	Storage *string `url:"storage"`
	// Stop stops any running backup jobs on the host before starting.
	Stop *bool `url:"stop"`
	// BWLimit limits I/O bandwidth in KiB/s (0: unlimited).
	BWLimit *int `url:"bwlimit"`
	// IONice is the IO priority used with the BFQ scheduler.
	IONice *int `url:"ionice"`
	// LockWait is the maximum time, in minutes, to wait for the global
	// lock.
	LockWait *int `url:"lockwait"`
	// StopWait is the maximum time, in minutes, to wait for a guest to
	// stop.
	StopWait *int `url:"stopwait"`

	// Performance is the opaque property-string of performance settings,
	// e.g. "max-workers=4".
	Performance *string `url:"performance"`
	// Fleecing is the opaque property-string of backup-fleecing settings
	// (VM guests only), e.g. "enabled=1,storage=local".
	Fleecing *string `url:"fleecing"`
	// PruneBackups is the opaque property-string of retention settings,
	// e.g. "keep-last=7,keep-daily=7", overriding the storage's own
	// configuration.
	PruneBackups *string `url:"prune-backups"`
	// Remove prunes older backups according to PruneBackups.
	Remove *bool `url:"remove"`

	// NotesTemplate is a template string used to generate backup notes;
	// requires Storage.
	NotesTemplate *string `url:"notes-template"`
	// Protected marks the resulting backup(s) as protected; requires
	// Storage.
	Protected *bool `url:"protected"`
	// PBSChangeDetectionMode is the PBS file-change detection mode used
	// for container backups.
	PBSChangeDetectionMode *PBSChangeDetectionMode `url:"pbs-change-detection-mode"`

	// Delete lists properties to reset to their default value (Update
	// only).
	Delete []string `url:"delete"`
}

// encode converts the options to form parameters.
func (o *JobOptions) encode() (map[string]string, error) {
	if o == nil {
		return nil, fmt.Errorf("backup: job options are required")
	}

	return params.Encode(o)
}
