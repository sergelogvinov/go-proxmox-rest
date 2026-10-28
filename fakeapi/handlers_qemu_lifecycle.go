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
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/storage"
)

// diskKeyPattern matches the guest config keys that hold a disk volume.
var diskKeyPattern = regexp.MustCompile(`^((ide|sata|scsi|virtio)\d+|efidisk0|tpmstate0)$`)

// netKeyPattern matches the guest config keys that hold a network device.
var netKeyPattern = regexp.MustCompile(`^net\d+$`)

// cloudInitDrivePattern matches the volume name of a guest's native
// cloud-init drive ("vm-<vmid>-cloudinit", optionally with a format
// suffix), which Proxmox attaches as a CD-ROM but owns like a disk.
var cloudInitDrivePattern = regexp.MustCompile(`^vm-\d+-cloudinit(\.\w+)?$`)

// diskVolume returns the volume id a disk config value names, or "" for an
// ordinary CD-ROM, an empty slot, or a value naming no storage volume. A
// native cloud-init drive counts as a disk.
func diskVolume(value string) string {
	vol, _, _ := strings.Cut(value, ",")
	if !strings.Contains(vol, ":") {
		return ""
	}

	if strings.Contains(value, "media=cdrom") {
		if _, name, _ := strings.Cut(vol, ":"); !cloudInitDrivePattern.MatchString(name) {
			return ""
		}
	}

	return vol
}

// diskProperty returns the value of one key=value property in a disk
// config value, or "".
func diskProperty(value, key string) string {
	for p := range strings.SplitSeq(value, ",") {
		if v, ok := strings.CutPrefix(p, key+"="); ok {
			return v
		}
	}

	return ""
}

// ownsVolume reports whether volid is one of vmid's own disk images
// ("vm-<vmid>-..." or, for a template, "base-<vmid>-..."), the volumes
// real Proxmox removes together with the guest. Content under a
// subdirectory ("iso/...", "import/...") is never a guest's own.
func ownsVolume(volid string, vmid int) bool {
	_, name, _ := strings.Cut(volid, ":")
	if strings.Contains(name, "/") {
		return false
	}

	return strings.HasPrefix(name, fmt.Sprintf("vm-%d-", vmid)) ||
		strings.HasPrefix(name, fmt.Sprintf("base-%d-", vmid))
}

// guestExists reports whether vmid names any QEMU guest or LXC container
// on any node. Called with state.mu held.
func guestExists(state *clusterState, vmid int) bool {
	for _, n := range state.nodes {
		if n.guests[vmid] != nil || n.containers[vmid] != nil {
			return true
		}
	}

	return false
}

// cloneMAC derives a stable MAC address for network device idx of a new
// guest, standing in for the fresh address real Proxmox generates on
// clone.
func cloneMAC(vmid, idx int) string {
	return fmt.Sprintf("BC:24:11:%02X:%02X:%02X", (vmid>>8)&0xff, vmid&0xff, idx&0xff)
}

// cloneUUID derives a stable SMBIOS UUID for a new guest, standing in for
// the fresh UUID real Proxmox generates on clone.
func cloneUUID(vmid int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", vmid)
}

// handleQemuClone backs POST /nodes/{node}/qemu/{vmid}/clone — Clone. A
// clone is a task: the new guest appears once it completes. Each source
// disk becomes a new "vm-<newid>-disk-<n>" volume on the target storage
// (or the source disk's own storage), a native cloud-init drive a new
// "vm-<newid>-cloudinit" one, network devices get new MAC
// addresses and smbios1 a new UUID, and the guest joins pool if given.
// Linked clones are not distinguished from full ones.
func handleQemuClone(w http.ResponseWriter, r *http.Request, n *nodeState) {
	vmid, ok := pathVMID(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()

	newID, err := strconv.Atoi(q.Get("newid"))
	if err != nil || newID == 0 {
		writeError(w, http.StatusBadRequest, "parameter verification failed", map[string]string{"newid": "not a number"})
		return
	}

	cs := n.cs
	cs.mu.Lock()

	src := n.guests[vmid]
	if src == nil {
		cs.mu.Unlock()
		notFound(w, "vm")
		return
	}

	if guestExists(cs, newID) {
		cs.mu.Unlock()
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("unable to create VM %d: config file already exists", newID), nil)
		return
	}

	target := n
	if t := q.Get("target"); t != "" {
		target = cs.nodes[t]
		if target == nil {
			cs.mu.Unlock()
			notFound(w, "target node")
			return
		}
	}

	poolID := q.Get("pool")
	if poolID != "" && cs.pools[poolID] == nil {
		cs.mu.Unlock()
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("pool '%s' does not exist", poolID), nil)
		return
	}

	storageID := q.Get("storage")
	if storageID != "" && target.storages[storageID] == nil {
		cs.mu.Unlock()
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("storage '%s' does not exist", storageID), nil)
		return
	}

	srcCfg := cloneStringMap(src.cfg)
	cs.mu.Unlock()

	name := q.Get("name")
	description, hasDescription := q["description"]

	upid := cs.startTask(r.Context(), n.name, "qmclone", strconv.Itoa(vmid), func() error {
		cs.mu.Lock()
		defer cs.mu.Unlock()

		cfg := srcCfg
		delete(cfg, "template")
		delete(cfg, "lock")

		if name != "" {
			cfg["name"] = name
		}

		if hasDescription && len(description) > 0 {
			cfg["description"] = description[0]
		}

		disk := 0

		for _, k := range slices.Sorted(maps.Keys(cfg)) {
			v := cfg[k]

			switch {
			case diskKeyPattern.MatchString(k):
				vol := diskVolume(v)
				if vol == "" {
					continue
				}

				store, _, _ := strings.Cut(vol, ":")
				if storageID != "" {
					store = storageID
				}

				st := target.storages[store]
				if st == nil {
					return fmt.Errorf("storage '%s' does not exist", store)
				}

				var newVol string
				if _, name, _ := strings.Cut(vol, ":"); cloudInitDrivePattern.MatchString(name) {
					newVol = fmt.Sprintf("%s:vm-%d-cloudinit", store, newID)
				} else {
					newVol = fmt.Sprintf("%s:vm-%d-disk-%d", store, newID, disk)
					disk++
				}

				st.content = append(st.content, storage.Volume{
					VolID:  newVol,
					VMID:   newID,
					Format: "raw",
					Size:   sizeToBytes(diskProperty(v, "size")),
					CTime:  time.Now().Unix(),
				})

				_, opts, _ := strings.Cut(v, ",")
				cfg[k] = strings.TrimSuffix(newVol+","+opts, ",")

			case netKeyPattern.MatchString(k):
				idx, _ := strconv.Atoi(strings.TrimPrefix(k, "net"))

				model, rest, _ := strings.Cut(v, ",")
				if m, _, ok := strings.Cut(model, "="); ok {
					model = m
				}

				cfg[k] = strings.TrimSuffix(model+"="+cloneMAC(newID, idx)+","+rest, ",")

			case k == "smbios1":
				parts := strings.Split(v, ",")
				for i, p := range parts {
					if strings.HasPrefix(p, "uuid=") {
						parts[i] = "uuid=" + cloneUUID(newID)
					}
				}

				cfg[k] = strings.Join(parts, ",")
			}
		}

		target.guests[newID] = &vmState{
			vmid:   newID,
			node:   target.name,
			cfg:    cfg,
			status: qemu.VMStatusStopped,
		}

		if poolID != "" {
			if p := cs.pools[poolID]; p != nil {
				p.vmids = append(p.vmids, newID)
			}
		}

		return nil
	})

	writeData(w, upid)
}

// handleQemuDelete backs DELETE /nodes/{node}/qemu/{vmid} — Delete. Like
// real Proxmox it refuses a running guest. The task removes the guest, its
// own disk volumes (see ownsVolume) from this node's storages, and its pool
// membership; volumes it merely references, such as an attached ISO, stay.
func handleQemuDelete(w http.ResponseWriter, r *http.Request, n *nodeState) {
	vmid, ok := pathVMID(w, r)
	if !ok {
		return
	}

	cs := n.cs
	cs.mu.Lock()
	vm := n.guests[vmid]
	running := vm != nil && vm.status == qemu.VMStatusRunning
	cs.mu.Unlock()

	if vm == nil {
		notFound(w, "vm")
		return
	}

	if running {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("VM %d is running - destroy failed", vmid), nil)
		return
	}

	upid := cs.startTask(r.Context(), n.name, "qmdestroy", strconv.Itoa(vmid), func() error {
		cs.mu.Lock()
		defer cs.mu.Unlock()

		for k, v := range vm.cfg {
			if !diskKeyPattern.MatchString(k) {
				continue
			}

			vol := diskVolume(v)
			if vol == "" || !ownsVolume(vol, vmid) {
				continue
			}

			store, _, _ := strings.Cut(vol, ":")
			if st := n.storages[store]; st != nil {
				if idx := findVolume(st, vol); idx >= 0 {
					st.content = slices.Delete(st.content, idx, idx+1)
				}
			}
		}

		for _, p := range cs.pools {
			p.vmids = slices.DeleteFunc(p.vmids, func(id int) bool { return id == vmid })
		}

		delete(n.guests, vmid)

		return nil
	})

	writeData(w, upid)
}

// handleQemuCloudInit backs PUT /nodes/{node}/qemu/{vmid}/cloudinit —
// CloudInitUpdate. The fake keeps no cloud-init drive contents, so it only
// checks that the guest exists.
func handleQemuCloudInit(w http.ResponseWriter, r *http.Request, n *nodeState) {
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

	writeData(w, nil)
}
