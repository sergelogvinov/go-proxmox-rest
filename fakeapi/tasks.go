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
	"errors"
	"fmt"
	"time"
)

// fakeTaskUser is the user every fake task is attributed to.
const fakeTaskUser = "root@pam!fakeapi"

// errTaskStopped is finishTask's error for a task terminated via
// handleTaskStop (DELETE .../tasks/{upid}), matching real Proxmox's
// ExitStatus text for a manually stopped task.
var errTaskStopped = errors.New("task stopped")

// newUPID mints a UPID in the same shape real Proxmox uses:
// UPID:{node}:{pid}:{pstart}:{starttime}:{type}:{id}:{user}:, so code
// that parses or logs a UPID sees realistic input. Must be called with
// state.mu held (it consumes state.taskSeq).
func newUPID(state *clusterState, node, typ, id string) (upid string, startTime int64) {
	state.taskSeq++
	seq := state.taskSeq
	startTime = time.Now().Unix()

	upid = fmt.Sprintf("UPID:%s:%08X:%08X:%08X:%s:%s:%s:",
		node, seq, seq, startTime, typ, id, fakeTaskUser)

	return upid, startTime
}

// startTask records a new task and, in the default instant mode, runs
// apply synchronously before returning — so
// Tasks().Status(ctx, node, upid) immediately reports StateStopped/"OK".
// Under WithManualTasks, apply is staged instead: the task is left
// running until a test calls TaskController.Complete or Fail. Either way
// the returned UPID is what handlers should hand back to the client.
func (s *clusterState) startTask(node, typ, id string, apply func() error) string {
	s.mu.Lock()
	upid, startTime := newUPID(s, node, typ, id)
	ts := &taskState{
		upid:      upid,
		node:      node,
		typ:       typ,
		id:        id,
		user:      fakeTaskUser,
		running:   true,
		startTime: startTime,
	}
	s.tasks[upid] = ts
	manual := s.manual
	s.mu.Unlock()

	if manual {
		ts.apply = apply
		return upid
	}

	finishTask(s, ts, apply())

	return upid
}

// finishTask records apply's outcome on ts and clears its running flag.
// Called either synchronously by startTask (instant mode) or later by
// TaskController.Complete (manual mode).
func finishTask(s *clusterState, ts *taskState, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ts.running = false
	ts.endTime = time.Now().Unix()
	ts.apply = nil

	if err != nil {
		ts.ok = false
		ts.errMsg = err.Error()
	} else {
		ts.ok = true
	}
}

// TaskController drives the async task engine under WithManualTasks,
// returned by Cluster.Tasks.
type TaskController struct {
	state *clusterState
}

// Complete applies a manual-mode task's staged mutation and flips it to
// StateStopped/"OK". A no-op if upid is unknown or already finished.
func (tc *TaskController) Complete(upid string) {
	tc.state.mu.Lock()
	ts := tc.state.tasks[upid]
	var apply func() error
	if ts != nil {
		apply = ts.apply
	}
	tc.state.mu.Unlock()

	if ts == nil || apply == nil {
		return
	}

	finishTask(tc.state, ts, apply())
}

// Fail discards a manual-mode task's staged mutation and flips it to
// StateStopped with ExitStatus set to message. A no-op if upid is unknown
// or already finished.
func (tc *TaskController) Fail(upid, message string) {
	tc.state.mu.Lock()
	ts := tc.state.tasks[upid]
	tc.state.mu.Unlock()

	if ts == nil {
		return
	}

	finishTask(tc.state, ts, fmt.Errorf("%s", message))
}
