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
	"strings"
	"time"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
)

// qemuActionTaskType names the fake task backing each status action,
// mirroring real Proxmox's qmstart/qmstop/... task types.
var qemuActionTaskType = map[string]string{
	"start": "qmstart", "stop": "qmstop", "shutdown": "qmshutdown",
	"reset": "qmreset", "reboot": "qmreboot",
	"suspend": "qmsuspend", "resume": "qmresume",
}

// qemuCreateSkipKeys are Client.Create-only meta-parameters that name the
// request itself rather than a guest config property to store, mirroring
// configSkipKeys' role for config updates.
var qemuCreateSkipKeys = map[string]bool{
	"vmid":  true,
	"pool":  true,
	"start": true,
}

// handleQemuCreate backs POST /nodes/{node}/qemu — Client.Create. Like
// real Proxmox, creation is a task: the guest becomes visible (GET
// config, GET status, cluster/resources) only once the task completes,
// which is immediate in the fake's default instant mode.
func handleQemuCreate(w http.ResponseWriter, r *http.Request, n *nodeState) {
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

	n.cs.mu.Lock()
	_, exists := n.guests[vmid]
	n.cs.mu.Unlock()
	if exists {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("VM %d already exists", vmid), nil)
		return
	}

	cfg := map[string]string{}
	for k, vals := range query {
		if qemuCreateSkipKeys[k] || len(vals) == 0 {
			continue
		}
		cfg[k] = vals[0]
	}

	start := query.Get("start") == "1"

	upid := n.cs.startTask(n.name, "qmcreate", strconv.Itoa(vmid), func() error {
		n.cs.mu.Lock()
		defer n.cs.mu.Unlock()

		vm := &vmState{
			vmid:   vmid,
			node:   n.name,
			cfg:    cfg,
			status: qemu.VMStatusStopped,
		}
		if start {
			vm.status = qemu.VMStatusRunning
			vm.startedAt = time.Now()
		}

		n.guests[vmid] = vm

		return nil
	})

	writeData(w, upid)
}

// handleQemuConfig backs GET/PUT/POST
// /nodes/{node}/qemu/{vmid}/config — Config, UpdateConfig, and
// UpdateConfigAsync all hit this one path, distinguished only by HTTP
// method, so one handler dispatches on r.Method rather than three routes
// fighting over the same pattern.
func handleQemuConfig(w http.ResponseWriter, r *http.Request, n *nodeState) {
	vmid, ok := pathVMID(w, r)
	if !ok {
		return
	}

	n.cs.mu.Lock()
	vm := n.guests[vmid]
	n.cs.mu.Unlock()
	if vm == nil {
		notFound(w, "vm")
		return
	}

	switch r.Method {
	case http.MethodGet:
		n.cs.mu.Lock()
		cfg := cloneStringMap(vm.cfg)
		n.cs.mu.Unlock()
		writeData(w, cfg)

	case http.MethodPut:
		n.cs.mu.Lock()
		applyConfigUpdate(vm.cfg, r.URL.Query())
		n.cs.mu.Unlock()
		writeData(w, nil)

	case http.MethodPost:
		query := r.URL.Query()
		upid := n.cs.startTask(n.name, "qmconfig", strconv.Itoa(vmid), func() error {
			n.cs.mu.Lock()
			applyConfigUpdate(vm.cfg, query)
			n.cs.mu.Unlock()
			return nil
		})
		writeData(w, upid)

	default:
		methodNotAllowed(w)
	}
}

// handleQemuStatusCurrent backs GET /nodes/{node}/qemu/{vmid}/status/current.
func handleQemuStatusCurrent(w http.ResponseWriter, r *http.Request, n *nodeState) {
	vmid, ok := pathVMID(w, r)
	if !ok {
		return
	}

	n.cs.mu.Lock()
	defer n.cs.mu.Unlock()

	vm := n.guests[vmid]
	if vm == nil {
		notFound(w, "vm")
		return
	}

	running := vm.status == qemu.VMStatusRunning
	status := &qemu.Status{
		VMID:   vmid,
		Status: vm.status,
		Name:   vm.cfg["name"],
		CPUs:   float64(parseIntDefault(vm.cfg["cores"], 1)),
		MaxMem: memoryMBToBytes(parseIntDefault(vm.cfg["memory"], 512)),
	}
	if running {
		status.QMPStatus = "running"
		status.PID = fakePID(vmid)
		status.Uptime = int64(guestUptime(true, vm.startedAt).Seconds())
	}

	writeData(w, status)
}

// handleQemuResize backs PUT /nodes/{node}/qemu/{vmid}/resize — Resize. Only
// absolute sizes are supported (a bare number with an optional K/M/G/T
// suffix); Proxmox's "+<delta>" grow-by-amount form isn't modeled, since
// this repo's own qemu.ResizeOptions callers only ever send an absolute
// size.
func handleQemuResize(w http.ResponseWriter, r *http.Request, n *nodeState) {
	vmid, ok := pathVMID(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()
	disk := query.Get("disk")
	size := query.Get("size")

	if disk == "" || size == "" {
		writeError(w, http.StatusBadRequest, "parameter verification failed", map[string]string{"disk": "disk and size are required"})
		return
	}

	n.cs.mu.Lock()
	vm := n.guests[vmid]
	n.cs.mu.Unlock()
	if vm == nil {
		notFound(w, "vm")
		return
	}

	upid := n.cs.startTask(n.name, "qmresize", strconv.Itoa(vmid), func() error {
		n.cs.mu.Lock()
		defer n.cs.mu.Unlock()

		vm.cfg[disk] = setPropertySize(vm.cfg[disk], size)

		return nil
	})

	writeData(w, upid)
}

// setPropertySize replaces (or appends) a drive property string's "size="
// component with size, leaving every other component untouched.
func setPropertySize(prop, size string) string {
	if prop == "" {
		return prop
	}

	parts := strings.Split(prop, ",")
	for i, p := range parts {
		if strings.HasPrefix(p, "size=") {
			parts[i] = "size=" + size
			return strings.Join(parts, ",")
		}
	}

	return prop + ",size=" + size
}

// handleQemuAction backs POST /nodes/{node}/qemu/{vmid}/status/{action}
// for start/stop/reset/shutdown/reboot/suspend/resume, all of which are
// task-backed in real Proxmox — see docs/fakeapi.md §7.
func handleQemuAction(w http.ResponseWriter, r *http.Request, n *nodeState) {
	vmid, ok := pathVMID(w, r)
	if !ok {
		return
	}

	action := r.PathValue("action")
	taskType, known := qemuActionTaskType[action]
	if !known {
		notFound(w, "action")
		return
	}

	n.cs.mu.Lock()
	vm := n.guests[vmid]
	n.cs.mu.Unlock()
	if vm == nil {
		notFound(w, "vm")
		return
	}

	upid := n.cs.startTask(n.name, taskType, strconv.Itoa(vmid), func() error {
		n.cs.mu.Lock()
		defer n.cs.mu.Unlock()

		switch action {
		case "start":
			vm.status = qemu.VMStatusRunning
			vm.startedAt = time.Now()
		case "stop", "shutdown":
			vm.status = qemu.VMStatusStopped
			vm.startedAt = time.Time{}
		case "reboot", "reset":
			if vm.status == qemu.VMStatusRunning {
				vm.startedAt = time.Now()
			}
		}
		// suspend/resume: this repo's qemu.VMStatus only models
		// stopped/running (no "paused"), so there's no additional state
		// to flip for them beyond the task itself completing.

		return nil
	})

	writeData(w, upid)
}
