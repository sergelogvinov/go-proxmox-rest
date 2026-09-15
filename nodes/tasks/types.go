package tasks

// State is a task's current running state, as reported by Client.Status.
// Distinct from Task.Status (List/Log), which holds the exit-status
// string once a task has finished.
type State string

const (
	StateRunning State = "running"
	StateStopped State = "stopped"
)

// Source narrows List to archived (finished, the default), active
// (currently running), or all tasks.
type Source string

const (
	SourceArchive Source = "archive"
	SourceActive  Source = "active"
	SourceAll     Source = "all"
)

// Task describes a finished or currently running task, as returned by
// GET /nodes/{node}/tasks.
type Task struct {
	// UPID is the task's unique process ID.
	UPID string `json:"upid,omitempty" url:"upid,omitempty"`
	// Node is the node the task ran/runs on.
	Node string `json:"node,omitempty" url:"node,omitempty"`
	// PID is the worker process's PID.
	PID int `json:"pid,omitempty" url:"pid,omitempty"`
	// PStart is the worker process's start counter (used together with
	// PID to detect PID reuse).
	PStart int64 `json:"pstart,omitempty" url:"pstart,omitempty"`
	// StartTime is the unix timestamp the task started at.
	StartTime int64 `json:"starttime,omitempty" url:"starttime,omitempty"`
	// Type is the task type, e.g. "vzstart", "vzdump".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// ID is the task's subject id, e.g. a VMID, when applicable.
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// User is the user that started the task.
	User string `json:"user,omitempty" url:"user,omitempty"`
	// TokenID is the API token id that started the task, populated
	// only when User acted through an API token.
	TokenID string `json:"tokenid,omitempty" url:"tokenid,omitempty"`
	// EndTime is the unix timestamp the task finished at; zero while
	// still running.
	EndTime int64 `json:"endtime,omitempty" url:"endtime,omitempty"`
	// Status is the finished task's outcome, "OK" or an error message;
	// empty while still running.
	Status string `json:"status,omitempty" url:"status,omitempty"`
}

// ListOptions filters the tasks returned by Client.List. A nil
// *ListOptions (or the zero value) requests Proxmox's defaults (the most
// recent 50 archived tasks).
type ListOptions struct {
	// Start is the offset to begin listing from.
	Start int `url:"start,omitempty"`
	// Limit caps the number of tasks returned (Proxmox defaults to 50).
	Limit int `url:"limit,omitempty"`
	// UserFilter restricts the result to tasks started by a matching
	// user.
	UserFilter string `url:"userfilter,omitempty"`
	// TypeFilter restricts the result to tasks of this type, e.g.
	// "vzdump".
	TypeFilter string `url:"typefilter,omitempty"`
	// VMID restricts the result to tasks for this guest.
	VMID int `url:"vmid,omitempty"`
	// Errors restricts the result to tasks that ended in an error.
	Errors bool `url:"errors,omitempty"`
	// Source selects archived, active, or all tasks.
	Source Source `url:"source,omitempty"`
	// Since restricts the result to tasks started at or after this
	// unix timestamp.
	Since int64 `url:"since,omitempty"`
	// Until restricts the result to tasks started at or before this
	// unix timestamp.
	Until int64 `url:"until,omitempty"`
	// StatusFilter restricts the result to tasks in these states, e.g.
	// []string{"error", "warning"}.
	StatusFilter []string `url:"statusfilter,omitempty"`
}

// LogEntry is a single task log line, as returned by
// GET /nodes/{node}/tasks/{upid}/log.
type LogEntry struct {
	// N is the line number.
	N int64 `json:"n,omitempty" url:"n,omitempty"`
	// T is the line text.
	T string `json:"t,omitempty" url:"t,omitempty"`
}

// LogOptions filters the log lines returned by Client.Log. A nil
// *LogOptions (or the zero value) requests Proxmox's default window (the
// first 50 lines).
type LogOptions struct {
	// Start is the first line number to return.
	Start int `url:"start,omitempty"`
	// Limit caps the number of lines returned (Proxmox defaults to 50).
	Limit int `url:"limit,omitempty"`
}

// Status describes a task's current state, as returned by
// GET /nodes/{node}/tasks/{upid}/status.
type Status struct {
	// UPID is the task's unique process ID.
	UPID string `json:"upid,omitempty" url:"upid,omitempty"`
	// Node is the node the task ran/runs on.
	Node string `json:"node,omitempty" url:"node,omitempty"`
	// PID is the worker process's PID.
	PID int `json:"pid,omitempty" url:"pid,omitempty"`
	// PStart is the worker process's start counter.
	PStart int64 `json:"pstart,omitempty" url:"pstart,omitempty"`
	// StartTime is the unix timestamp the task started at.
	StartTime int64 `json:"starttime,omitempty" url:"starttime,omitempty"`
	// Type is the task type, e.g. "vzstart", "vzdump".
	Type string `json:"type,omitempty" url:"type,omitempty"`
	// ID is the task's subject id, e.g. a VMID, when applicable.
	ID string `json:"id,omitempty" url:"id,omitempty"`
	// User is the user that started the task.
	User string `json:"user,omitempty" url:"user,omitempty"`
	// Status is the task's current running state.
	Status State `json:"status,omitempty" url:"status,omitempty"`
	// ExitStatus is the finished task's exit status string, populated
	// only once Status is StateStopped.
	ExitStatus string `json:"exitstatus,omitempty" url:"exitstatus,omitempty"`
}
