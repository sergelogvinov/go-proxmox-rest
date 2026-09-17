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
	"maps"
	"net/url"
	"strconv"
	"strings"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/lxc"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu"
)

// flattenQemuConfig converts cfg into the same flat wire-parameter map
// nodes/qemu's own (unexported) encodeConfig builds: params.Encode(cfg)
// covers every ordinary named field (including property-string-valued
// ones like Memory/CPU/Machine, which implement fmt.Stringer), and the
// loops below add one "prefix+index" entry per populated indexed-family
// map, mirroring nodes/qemu/config.go's addIndexedXxx helpers field for
// field. See vmState's doc comment (state.go) for why this map — not a
// live qemu.Config — is what's actually stored.
func flattenQemuConfig(cfg *qemu.Config) map[string]string {
	if cfg == nil {
		return map[string]string{}
	}

	p, err := params.Encode(cfg)
	if err != nil {
		p = map[string]string{}
	}

	for idx, v := range cfg.Net {
		p["net"+strconv.Itoa(idx)] = v.String()
	}
	for idx, v := range cfg.IDE {
		p["ide"+strconv.Itoa(idx)] = v.String()
	}
	for idx, v := range cfg.SATA {
		p["sata"+strconv.Itoa(idx)] = v.String()
	}
	for idx, v := range cfg.SCSI {
		p["scsi"+strconv.Itoa(idx)] = v.String()
	}
	for idx, v := range cfg.VirtIO {
		p["virtio"+strconv.Itoa(idx)] = v.String()
	}
	for idx, v := range cfg.VirtioFS {
		p["virtiofs"+strconv.Itoa(idx)] = v.String()
	}
	for idx, v := range cfg.Unused {
		p["unused"+strconv.Itoa(idx)] = v
	}
	for idx, v := range cfg.USB {
		p["usb"+strconv.Itoa(idx)] = v.String()
	}
	for idx, v := range cfg.HostPCI {
		p["hostpci"+strconv.Itoa(idx)] = v.String()
	}
	for idx, v := range cfg.Serial {
		p["serial"+strconv.Itoa(idx)] = v
	}
	for idx, v := range cfg.IPConfig {
		p["ipconfig"+strconv.Itoa(idx)] = v.String()
	}
	for idx, v := range cfg.NUMA {
		p["numa"+strconv.Itoa(idx)] = v.String()
	}

	return p
}

// flattenLXCConfig is flattenQemuConfig's LXC counterpart, mirroring
// nodes/lxc/config.go's (unexported) encodeConfig.
func flattenLXCConfig(cfg *lxc.Config) map[string]string {
	if cfg == nil {
		return map[string]string{}
	}

	p, err := params.Encode(cfg)
	if err != nil {
		p = map[string]string{}
	}

	for idx, v := range cfg.Net {
		p["net"+strconv.Itoa(idx)] = v.String()
	}
	for idx, v := range cfg.MP {
		p["mp"+strconv.Itoa(idx)] = v.String()
	}
	for idx, v := range cfg.Unused {
		p["unused"+strconv.Itoa(idx)] = v
	}

	return p
}

// configSkipKeys are write-only meta-parameters that apply to a config
// update itself rather than naming a config property to store, mirroring
// qemu.Config/lxc.Config's own "writeonly" fields (digest/force/skiplock/
// background_delay/import-working-storage on the qemu side, revert on
// both). "revert" is accepted but ignored, like "digest", since the fake
// does not model pending changes.
var configSkipKeys = map[string]bool{
	"delete":                 true,
	"revert":                 true,
	"force":                  true,
	"skiplock":               true,
	"digest":                 true,
	"background_delay":       true,
	"import-working-storage": true,
}

// applyConfigUpdate merges query's parameters into cfg, honoring a
// comma-joined "delete" parameter the same way Proxmox does: remove those
// keys instead of setting them. Called with the guest's state mutex held.
func applyConfigUpdate(cfg map[string]string, query url.Values) {
	if del := query.Get("delete"); del != "" {
		for k := range strings.SplitSeq(del, ",") {
			delete(cfg, strings.TrimSpace(k))
		}
	}

	for k, vals := range query {
		if configSkipKeys[k] || len(vals) == 0 {
			continue
		}
		cfg[k] = vals[0]
	}
}

// cloneStringMap returns a shallow copy of m, so a handler can release
// the state lock before JSON-encoding the result.
func cloneStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	maps.Copy(out, m)

	return out
}

// parseIntDefault parses s as a base-10 int, returning def if s is empty
// or not a valid number — used to read a guest's cfg["cores"]/["memory"]
// back out for status responses without round-tripping through the full
// property-string grammar those fields can also take.
func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}

	return n
}
