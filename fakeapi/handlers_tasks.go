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

package fakeapi

import (
	"net/http"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/tasks"
)

// handleTaskStatus backs GET /nodes/{node}/tasks/{upid}/status.
func handleTaskStatus(w http.ResponseWriter, r *http.Request, n *nodeState) {
	upid := r.PathValue("upid")

	n.cs.mu.Lock()
	ts := n.cs.tasks[upid]
	n.cs.mu.Unlock()
	if ts == nil {
		notFound(w, "task")
		return
	}

	n.cs.mu.Lock()
	status := &tasks.Status{
		UPID: ts.upid, Node: ts.node, Type: ts.typ, ID: ts.id,
		User: ts.user, StartTime: ts.startTime,
	}
	if ts.running {
		status.Status = tasks.StateRunning
	} else {
		status.Status = tasks.StateStopped
		if ts.ok {
			status.ExitStatus = "OK"
		} else {
			status.ExitStatus = ts.errMsg
		}
	}
	n.cs.mu.Unlock()

	writeData(w, status)
}

// handleTaskStop backs DELETE /nodes/{node}/tasks/{upid} — Stop, which
// only has an effect on a still-running (i.e. WithManualTasks) task: it
// marks it failed with "task stopped" instead of running its staged
// mutation.
func handleTaskStop(w http.ResponseWriter, r *http.Request, n *nodeState) {
	upid := r.PathValue("upid")

	n.cs.mu.Lock()
	ts := n.cs.tasks[upid]
	running := ts != nil && ts.running
	n.cs.mu.Unlock()
	if ts == nil {
		notFound(w, "task")
		return
	}

	if running {
		finishTask(n.cs, ts, errTaskStopped)
	}

	writeData(w, nil)
}
