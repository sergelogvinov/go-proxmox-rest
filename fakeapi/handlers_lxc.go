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
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc"
)

// lxcActionTaskType names the fake task backing each status action,
// mirroring real Proxmox's vzstart/vzstop/... task types. LXC has no
// "reset" action, unlike QEMU.
var lxcActionTaskType = map[string]string{
	"start": "vzstart", "stop": "vzstop", "shutdown": "vzshutdown",
	"reboot": "vzreboot", "suspend": "vzsuspend", "resume": "vzresume",
}

// lxcCreateSkipKeys are Client.Create-only meta-parameters that name the
// request itself rather than a container config property to store,
// mirroring qemuCreateSkipKeys' role for QEMU guest creation.
var lxcCreateSkipKeys = map[string]bool{
	"vmid":       true,
	"ostemplate": true,
	"pool":       true,
	"start":      true,
}

// handleLXCCreate backs POST /nodes/{node}/lxc — Client.Create. Like
// real Proxmox, creation is a task: the container becomes visible (GET
// config, GET status, cluster/resources) only once the task completes,
// which is immediate in the fake's default instant mode.
func handleLXCCreate(w http.ResponseWriter, r *http.Request, n *nodeState) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	query := r.URL.Query()

	vmid, err := strconv.Atoi(query.Get("vmid"))
	if err != nil || vmid == 0 {
		writeError(w, http.StatusBadRequest, "parameter verification failed", map[string]string{"vmid": "not a number"})
		return
	}

	if query.Get("ostemplate") == "" {
		writeError(w, http.StatusBadRequest, "parameter verification failed", map[string]string{"ostemplate": "property is missing and it is not optional"})
		return
	}

	n.cs.mu.Lock()
	_, exists := n.containers[vmid]
	n.cs.mu.Unlock()
	if exists {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("CT %d already exists", vmid), nil)
		return
	}

	cfg := map[string]string{}
	for k, vals := range query {
		if lxcCreateSkipKeys[k] || len(vals) == 0 {
			continue
		}
		cfg[k] = vals[0]
	}

	start := query.Get("start") == "1"

	upid := n.cs.startTask(n.name, "vzcreate", strconv.Itoa(vmid), func() error {
		n.cs.mu.Lock()
		defer n.cs.mu.Unlock()

		ct := &containerState{
			vmid:   vmid,
			node:   n.name,
			cfg:    cfg,
			status: lxc.StateStopped,
		}
		if start {
			ct.status = lxc.StateRunning
			ct.startedAt = time.Now()
		}

		n.containers[vmid] = ct

		return nil
	})

	writeData(w, upid)
}

// handleLXCConfig backs GET/PUT /nodes/{node}/lxc/{vmid}/config — see
// handleQemuConfig's doc comment for why one handler dispatches on
// r.Method instead of two routes.
func handleLXCConfig(w http.ResponseWriter, r *http.Request, n *nodeState) {
	vmid, ok := pathVMID(w, r)
	if !ok {
		return
	}

	n.cs.mu.Lock()
	ct := n.containers[vmid]
	n.cs.mu.Unlock()
	if ct == nil {
		notFound(w, "container")
		return
	}

	switch r.Method {
	case http.MethodGet:
		n.cs.mu.Lock()
		cfg := cloneStringMap(ct.cfg)
		n.cs.mu.Unlock()
		writeData(w, cfg)

	case http.MethodPut:
		n.cs.mu.Lock()
		applyConfigUpdate(ct.cfg, r.URL.Query())
		n.cs.mu.Unlock()
		writeData(w, nil)

	default:
		methodNotAllowed(w)
	}
}

// handleLXCStatusCurrent backs GET /nodes/{node}/lxc/{vmid}/status/current.
func handleLXCStatusCurrent(w http.ResponseWriter, r *http.Request, n *nodeState) {
	vmid, ok := pathVMID(w, r)
	if !ok {
		return
	}

	n.cs.mu.Lock()
	defer n.cs.mu.Unlock()

	ct := n.containers[vmid]
	if ct == nil {
		notFound(w, "container")
		return
	}

	status := &lxc.Status{
		VMID:   vmid,
		Status: ct.status,
		Name:   ct.cfg["hostname"],
		CPUs:   float64(parseIntDefault(ct.cfg["cores"], 1)),
		MaxMem: memoryMBToBytes(parseIntDefault(ct.cfg["memory"], 512)),
	}
	if ct.status == lxc.StateRunning {
		status.Uptime = int64(guestUptime(true, ct.startedAt).Seconds())
	}

	writeData(w, status)
}

// handleLXCAction backs POST /nodes/{node}/lxc/{vmid}/status/{action} for
// start/stop/shutdown/reboot/suspend/resume.
func handleLXCAction(w http.ResponseWriter, r *http.Request, n *nodeState) {
	vmid, ok := pathVMID(w, r)
	if !ok {
		return
	}

	action := r.PathValue("action")
	taskType, known := lxcActionTaskType[action]
	if !known {
		notFound(w, "action")
		return
	}

	n.cs.mu.Lock()
	ct := n.containers[vmid]
	n.cs.mu.Unlock()
	if ct == nil {
		notFound(w, "container")
		return
	}

	upid := n.cs.startTask(n.name, taskType, strconv.Itoa(vmid), func() error {
		n.cs.mu.Lock()
		defer n.cs.mu.Unlock()

		switch action {
		case "start":
			ct.status = lxc.StateRunning
			ct.startedAt = time.Now()
		case "stop", "shutdown":
			ct.status = lxc.StateStopped
			ct.startedAt = time.Time{}
		case "reboot":
			if ct.status == lxc.StateRunning {
				ct.startedAt = time.Now()
			}
		}
		// suspend/resume: this repo's lxc.State only models
		// stopped/running, so there's no additional state to flip for
		// them beyond the task itself completing.

		return nil
	})

	writeData(w, upid)
}
