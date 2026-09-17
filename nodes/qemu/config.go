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

package qemu

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
	"github.com/sergelogvinov/go-proxmox-rest/internal/property"
)

// indexedPrefixes are the numerically-suffixed VM config families
// (net0, ide2, hostpci3, ...) that Config models as map[int]string
// instead of one field per index. A scalar field that merely happens to
// end in a digit (efidisk0, tpmstate0, rng0, audio0, smbios1) is not in
// this set and is decoded/encoded as an ordinary named field instead.
var indexedPrefixes = map[string]bool{
	"net":      true,
	"ide":      true,
	"sata":     true,
	"scsi":     true,
	"virtio":   true,
	"virtiofs": true,
	"unused":   true,
	"usb":      true,
	"hostpci":  true,
	"serial":   true,
	"parallel": true,
	"ipconfig": true,
	"numa":     true,
}

// indexedKeyRe splits a config key into its alphabetic prefix and
// trailing numeric index, e.g. "hostpci3" -> ("hostpci", "3").
var indexedKeyRe = regexp.MustCompile(`^([a-z]+)(\d+)$`)

// configPath builds the /nodes/{node}/qemu/{vmid}/config URL for the
// given node and vmid.
func configPath(node string, vmid int) string {
	return "/nodes/" + node + "/qemu/" + strconv.Itoa(vmid) + "/config"
}

// Config retrieves a guest's configuration via
// GET /nodes/{node}/qemu/{vmid}/config. opts may be nil to request the
// configuration with pending changes applied.
func (c *Client) Config(ctx context.Context, vmid int, opts *ConfigOptions) (*Config, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var raw map[string]json.RawMessage
	if err := c.client.Get(ctx, configPath(c.node, vmid), &raw, p); err != nil {
		return nil, err
	}

	return decodeConfig(raw)
}

// UpdateConfig sets a guest's configuration synchronously via
// PUT /nodes/{node}/qemu/{vmid}/config. Prefer UpdateConfigAsync for
// changes involving hotplug or storage allocation, per Proxmox's own
// guidance. cfg.BackgroundDelay and cfg.ImportWorkingStorage are rejected
// by this endpoint (POST-only); leave them unset.
func (c *Client) UpdateConfig(ctx context.Context, vmid int, cfg *Config) error {
	p, err := encodeConfig(cfg)
	if err != nil {
		return err
	}

	return c.client.Update(ctx, configPath(c.node, vmid), nil, p)
}

// UpdateConfigAsync sets a guest's configuration via
// POST /nodes/{node}/qemu/{vmid}/config, as a background task — the
// endpoint to prefer for changes involving hotplug or storage
// allocation. Returns the task's UPID.
func (c *Client) UpdateConfigAsync(ctx context.Context, vmid int, cfg *Config) (string, error) {
	p, err := encodeConfig(cfg)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Create(ctx, configPath(c.node, vmid), &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}

// decodeConfig splits raw's numerically-suffixed hardware keys into
// cfg's map[int]string fields, then decodes everything else (the
// ordinary named fields) through params.Decode as usual.
func decodeConfig(raw map[string]json.RawMessage) (*Config, error) {
	cfg := &Config{}
	scalar := make(map[string]json.RawMessage, len(raw))

	for key, val := range raw {
		prefix, index, ok := splitIndexedKey(key)
		if !ok {
			scalar[key] = val
			continue
		}

		var v string
		if err := json.Unmarshal(val, &v); err != nil {
			return nil, fmt.Errorf("qemu: config field %s: %w", key, err)
		}
		setIndexedField(cfg, prefix, index, v)
	}

	b, err := json.Marshal(scalar)
	if err != nil {
		return nil, err
	}
	if err := params.Decode(b, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// encodeConfig encodes cfg's ordinary named fields through params.Encode
// as usual, then adds one "prefix+index" entry per populated
// map[int]string field.
func encodeConfig(cfg *Config) (map[string]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("qemu: config is required")
	}

	p, err := params.Encode(cfg)
	if err != nil {
		return nil, err
	}

	addIndexedNet(p, cfg.Net)
	addIndexedDrive(p, "ide", cfg.IDE)
	addIndexedDrive(p, "sata", cfg.SATA)
	addIndexedDrive(p, "scsi", cfg.SCSI)
	addIndexedDrive(p, "virtio", cfg.VirtIO)
	addIndexedVirtioFS(p, cfg.VirtioFS)
	addIndexed(p, "unused", cfg.Unused)
	addIndexedUSB(p, cfg.USB)
	addIndexedHostPCI(p, cfg.HostPCI)
	addIndexed(p, "serial", cfg.Serial)
	addIndexedIPConfig(p, cfg.IPConfig)
	addIndexedNUMA(p, cfg.NUMA)

	return p, nil
}

// splitIndexedKey reports whether key is one of Proxmox's numerically
// suffixed VM config families, returning the family prefix and index if
// so.
func splitIndexedKey(key string) (prefix string, index int, ok bool) {
	m := indexedKeyRe.FindStringSubmatch(key)
	if m == nil || !indexedPrefixes[m[1]] {
		return "", 0, false
	}

	idx, err := strconv.Atoi(m[2])
	if err != nil {
		return "", 0, false
	}

	return m[1], idx, true
}

// setIndexedField assigns v into cfg's map field for the given family
// prefix, allocating the map on first use.
func setIndexedField(cfg *Config, prefix string, index int, v string) {
	switch prefix {
	case "net":
		setIndexedNet(&cfg.Net, index, v)
	case "ide":
		setIndexedDrive(&cfg.IDE, index, v)
	case "sata":
		setIndexedDrive(&cfg.SATA, index, v)
	case "scsi":
		setIndexedDrive(&cfg.SCSI, index, v)
	case "virtio":
		setIndexedDrive(&cfg.VirtIO, index, v)
	case "virtiofs":
		setIndexedVirtioFS(&cfg.VirtioFS, index, v)
	case "unused":
		setIndexed(&cfg.Unused, index, v)
	case "usb":
		setIndexedUSB(&cfg.USB, index, v)
	case "hostpci":
		setIndexedHostPCI(&cfg.HostPCI, index, v)
	case "serial":
		setIndexed(&cfg.Serial, index, v)
	case "ipconfig":
		setIndexedIPConfig(&cfg.IPConfig, index, v)
	case "numa":
		setIndexedNUMA(&cfg.NUMA, index, v)
	}
}

// setIndexed sets (*m)[index] = v, allocating *m on first use.
func setIndexed(m *map[int]string, index int, v string) {
	if *m == nil {
		*m = map[int]string{}
	}

	(*m)[index] = v
}

// setIndexedDrive parses v as a drive property string and stores it at
// (*m)[index], allocating *m on first use.
func setIndexedDrive(m *map[int]Drive, index int, v string) {
	if *m == nil {
		*m = map[int]Drive{}
	}

	drive := Drive{}
	_ = property.Unmarshal(v, &drive)
	(*m)[index] = drive
}

// setIndexedNet parses v as a netN property string and stores it at
// (*m)[index], allocating *m on first use.
func setIndexedNet(m *map[int]Net, index int, v string) {
	if *m == nil {
		*m = map[int]Net{}
	}

	net := Net{}
	_ = property.Unmarshal(v, &net)
	(*m)[index] = net
}

// setIndexedNUMA parses v as a NUMA property string and stores it at
// (*m)[index], allocating *m on first use.
func setIndexedNUMA(m *map[int]NUMA, index int, v string) {
	if *m == nil {
		*m = map[int]NUMA{}
	}

	numa := NUMA{}
	_ = property.Unmarshal(v, &numa)
	(*m)[index] = numa
}

// setIndexedHostPCI parses v as a hostpciN property string and stores it
// at (*m)[index], allocating *m on first use.
func setIndexedHostPCI(m *map[int]HostPCI, index int, v string) {
	if *m == nil {
		*m = map[int]HostPCI{}
	}

	hostpci := HostPCI{}
	_ = property.Unmarshal(v, &hostpci)
	(*m)[index] = hostpci
}

// setIndexedUSB parses v as a usbN property string and stores it at
// (*m)[index], allocating *m on first use.
func setIndexedUSB(m *map[int]USB, index int, v string) {
	if *m == nil {
		*m = map[int]USB{}
	}

	usb := USB{}
	_ = property.Unmarshal(v, &usb)
	(*m)[index] = usb
}

// setIndexedIPConfig parses v as an ipconfigN property string and stores
// it at (*m)[index], allocating *m on first use.
func setIndexedIPConfig(m *map[int]IPConfig, index int, v string) {
	if *m == nil {
		*m = map[int]IPConfig{}
	}

	ipconfig := IPConfig{}
	_ = property.Unmarshal(v, &ipconfig)
	(*m)[index] = ipconfig
}

// setIndexedVirtioFS parses v as a virtiofsN property string and stores it
// at (*m)[index], allocating *m on first use.
func setIndexedVirtioFS(m *map[int]VirtioFS, index int, v string) {
	if *m == nil {
		*m = map[int]VirtioFS{}
	}

	virtiofs := VirtioFS{}
	_ = property.Unmarshal(v, &virtiofs)
	(*m)[index] = virtiofs
}

// addIndexed adds one "prefix+index" entry to p per key in m.
func addIndexed(p map[string]string, prefix string, m map[int]string) {
	for idx, v := range m {
		p[prefix+strconv.Itoa(idx)] = v
	}
}

// addIndexedDrive adds one "prefix+index" entry to p per key in m,
// serializing each drive back to its property string.
func addIndexedDrive(p map[string]string, prefix string, m map[int]Drive) {
	for idx, v := range m {
		p[prefix+strconv.Itoa(idx)] = v.String()
	}
}

// addIndexedNet adds one "net<index>" entry to p per key in m,
// serializing each network interface back to its property string.
func addIndexedNet(p map[string]string, m map[int]Net) {
	for idx, v := range m {
		p["net"+strconv.Itoa(idx)] = v.String()
	}
}

// addIndexedNUMA adds one "numa<index>" entry to p per key in m,
// serializing each NUMA node back to its property string.
func addIndexedNUMA(p map[string]string, m map[int]NUMA) {
	for idx, v := range m {
		p["numa"+strconv.Itoa(idx)] = v.String()
	}
}

// addIndexedHostPCI adds one "hostpci<index>" entry to p per key in m,
// serializing each passthrough device back to its property string.
func addIndexedHostPCI(p map[string]string, m map[int]HostPCI) {
	for idx, v := range m {
		p["hostpci"+strconv.Itoa(idx)] = v.String()
	}
}

// addIndexedUSB adds one "usb<index>" entry to p per key in m,
// serializing each USB device back to its property string.
func addIndexedUSB(p map[string]string, m map[int]USB) {
	for idx, v := range m {
		p["usb"+strconv.Itoa(idx)] = v.String()
	}
}

// addIndexedIPConfig adds one "ipconfig<index>" entry to p per key in m,
// serializing each per-NIC IP configuration back to its property string.
func addIndexedIPConfig(p map[string]string, m map[int]IPConfig) {
	for idx, v := range m {
		p["ipconfig"+strconv.Itoa(idx)] = v.String()
	}
}

// addIndexedVirtioFS adds one "virtiofs<index>" entry to p per key in m,
// serializing each virtiofs share back to its property string.
func addIndexedVirtioFS(p map[string]string, m map[int]VirtioFS) {
	for idx, v := range m {
		p["virtiofs"+strconv.Itoa(idx)] = v.String()
	}
}
