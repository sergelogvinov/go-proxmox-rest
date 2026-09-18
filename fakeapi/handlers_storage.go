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
	"strconv"
	"strings"
	"time"

	"github.com/sergelogvinov/go-proxmox-rest/nodes/storage"
)

// handleStorageList backs GET /nodes/{node}/storage.
func handleStorageList(w http.ResponseWriter, r *http.Request, n *nodeState) {
	n.cs.mu.Lock()
	defer n.cs.mu.Unlock()

	list := make([]storage.Storage, 0, len(n.storages))
	for _, id := range sortedStorageIDs(n) {
		list = append(list, storageView(n.storages[id], true))
	}

	writeData(w, list)
}

// handleStorageStatus backs GET /nodes/{node}/storage/{storage}/status.
// The response omits the storage id, matching storage.Client.Status's
// doc comment ("it's already the URL path segment").
func handleStorageStatus(w http.ResponseWriter, r *http.Request, n *nodeState) {
	id := r.PathValue("storage")

	n.cs.mu.Lock()
	defer n.cs.mu.Unlock()

	st := n.storages[id]
	if st == nil {
		notFound(w, "storage")
		return
	}

	writeData(w, storageView(st, false))
}

func storageView(st *storageState, includeID bool) storage.Storage {
	s := storage.Storage{
		Type:           st.typ,
		Enabled:        true,
		Active:         true,
		TotalSpace:     st.total,
		UsedSpace:      st.used,
		AvailableSpace: st.avail,
	}
	if includeID {
		s.Storage = st.id
	}
	if st.total > 0 {
		s.UsedFraction = float64(st.used) / float64(st.total)
	}

	return s
}

// handleStorageContentCollection backs GET/POST
// /nodes/{node}/storage/{storage}/content — List and Create.
func handleStorageContentCollection(w http.ResponseWriter, r *http.Request, n *nodeState) {
	id := r.PathValue("storage")

	n.cs.mu.Lock()
	st := n.storages[id]
	n.cs.mu.Unlock()
	if st == nil {
		notFound(w, "storage")
		return
	}

	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()

		var vmidFilter int
		hasVMIDFilter := false
		if v := q.Get("vmid"); v != "" {
			if parsed, err := strconv.Atoi(v); err == nil {
				vmidFilter, hasVMIDFilter = parsed, true
			}
		}

		n.cs.mu.Lock()
		vols := make([]storage.Volume, 0, len(st.content))
		for _, v := range st.content {
			if hasVMIDFilter && v.VMID != vmidFilter {
				continue
			}
			vols = append(vols, v)
		}
		n.cs.mu.Unlock()

		writeData(w, vols)

	case http.MethodPost:
		q := r.URL.Query()
		filename := q.Get("filename")
		size := q.Get("size")
		if filename == "" || size == "" {
			writeError(w, http.StatusBadRequest, "parameter verification failed",
				map[string]string{"filename": "filename and size are required"})
			return
		}

		vmid, _ := strconv.Atoi(q.Get("vmid"))
		format := q.Get("format")
		if format == "" {
			format = "raw"
		}

		volid := id + ":" + filename
		vol := storage.Volume{
			VolID:  volid,
			VMID:   vmid,
			Format: format,
			Size:   sizeToBytes(size),
			CTime:  time.Now().Unix(),
		}

		n.cs.mu.Lock()
		st.content = append(st.content, vol)
		n.cs.mu.Unlock()

		writeData(w, volid)

	default:
		methodNotAllowed(w)
	}
}

// handleStorageContentItem backs GET/PUT/POST/DELETE
// /nodes/{node}/storage/{storage}/content/{volume} — Get, Update, Copy,
// and Delete. volume may be a bare volume name (resolved against
// storageID) or a full "storage:name" volume id, matching
// storage.Client.Content(storageID).Get's doc comment.
func handleStorageContentItem(w http.ResponseWriter, r *http.Request, n *nodeState) {
	id := r.PathValue("storage")
	volume := r.PathValue("volume")

	n.cs.mu.Lock()
	st := n.storages[id]
	n.cs.mu.Unlock()
	if st == nil {
		notFound(w, "storage")
		return
	}

	full := volume
	if !strings.Contains(volume, ":") {
		full = id + ":" + volume
	}

	switch r.Method {
	case http.MethodGet:
		n.cs.mu.Lock()
		idx := findVolume(st, full)
		var vol storage.Volume
		if idx >= 0 {
			vol = st.content[idx]
		}
		n.cs.mu.Unlock()

		if idx < 0 {
			notFound(w, "volume")
			return
		}
		writeData(w, vol)

	case http.MethodPut:
		q := r.URL.Query()
		notes, hasNotes := q["notes"]
		protected, hasProtected := q["protected"]

		n.cs.mu.Lock()
		idx := findVolume(st, full)
		if idx >= 0 {
			if hasNotes && len(notes) > 0 {
				st.content[idx].Notes = notes[0]
			}
			if hasProtected && len(protected) > 0 {
				st.content[idx].Protected = protected[0] == "1" || strings.EqualFold(protected[0], "true")
			}
		}
		n.cs.mu.Unlock()

		if idx < 0 {
			notFound(w, "volume")
			return
		}
		writeData(w, nil)

	case http.MethodPost:
		handleStorageContentCopy(w, r, n, id, st, full)

	case http.MethodDelete:
		n.cs.mu.Lock()
		idx := findVolume(st, full)
		if idx >= 0 {
			st.content = append(st.content[:idx], st.content[idx+1:]...)
		}
		n.cs.mu.Unlock()

		if idx < 0 {
			notFound(w, "volume")
			return
		}
		// Content().Delete is unconditional in real Proxmox too (see
		// docs/fakeapi.md §13) and this fake always completes it
		// synchronously, so the UPID it returns is always empty.
		writeData(w, "")

	default:
		methodNotAllowed(w)
	}
}

// handleStorageContentCopy backs the POST case of handleStorageContentItem
// — Content().Copy. It copies the source volume (already resolved to its
// full "storage:name" id as full) to a new volume named by the "target"
// query parameter, on the same storage id but optionally a different node
// (the "target_node" query parameter). Like guest creation/config changes,
// this is modeled as a task, matching real Proxmox: the copy is not
// visible in the target's content list until the task completes, which is
// immediate in the fake's default instant mode.
func handleStorageContentCopy(w http.ResponseWriter, r *http.Request, n *nodeState, storageID string, st *storageState, full string) {
	q := r.URL.Query()

	target := q.Get("target")
	if target == "" {
		writeError(w, http.StatusBadRequest, "parameter verification failed", map[string]string{"target": "target is required"})
		return
	}

	targetNode := q.Get("target_node")
	if targetNode == "" {
		targetNode = n.name
	}

	n.cs.mu.Lock()
	idx := findVolume(st, full)
	var src storage.Volume
	if idx >= 0 {
		src = st.content[idx]
	}
	destNode, destNodeOK := n.cs.nodes[targetNode]
	var destSt *storageState
	if destNodeOK {
		destSt = destNode.storages[storageID]
	}
	n.cs.mu.Unlock()

	if idx < 0 {
		notFound(w, "volume")
		return
	}
	if !destNodeOK {
		notFound(w, "target node")
		return
	}
	if destSt == nil {
		notFound(w, "target storage")
		return
	}

	newVolID := storageID + ":" + target

	n.cs.mu.Lock()
	dup := findVolume(destSt, newVolID) >= 0
	n.cs.mu.Unlock()
	if dup {
		writeError(w, http.StatusBadRequest, newVolID+" already exists", nil)
		return
	}

	upid := n.cs.startTask(n.name, "imgcopy", full, func() error {
		n.cs.mu.Lock()
		defer n.cs.mu.Unlock()

		copyVol := src
		copyVol.VolID = newVolID
		copyVol.CTime = time.Now().Unix()

		destSt.content = append(destSt.content, copyVol)

		return nil
	})

	writeData(w, upid)
}

func findVolume(st *storageState, volid string) int {
	for i, v := range st.content {
		if v.VolID == volid {
			return i
		}
	}
	return -1
}
