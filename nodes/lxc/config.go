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

package lxc

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
	"github.com/sergelogvinov/go-proxmox-rest/internal/property"
)

// indexedPrefixes are the numerically-suffixed CT config families
// (net0, mp3, unused12, ...) that Config models as map[int]string
// instead of one field per index.
var indexedPrefixes = map[string]bool{
	"net":    true,
	"mp":     true,
	"unused": true,
}

// indexedKeyRe splits a config key into its alphabetic prefix and
// trailing numeric index, e.g. "mp3" -> ("mp", "3").
var indexedKeyRe = regexp.MustCompile(`^([a-z]+)(\d+)$`)

// configPath builds the /nodes/{node}/lxc/{vmid}/config URL for the
// given node and vmid.
func configPath(node string, vmid int) string {
	return "/nodes/" + node + "/lxc/" + strconv.Itoa(vmid) + "/config"
}

// Config retrieves a container's configuration via
// GET /nodes/{node}/lxc/{vmid}/config. opts may be nil to request the
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

// UpdateConfig sets a container's configuration via
// PUT /nodes/{node}/lxc/{vmid}/config.
func (c *Client) UpdateConfig(ctx context.Context, vmid int, cfg *Config) error {
	p, err := encodeConfig(cfg)
	if err != nil {
		return err
	}

	return c.client.Update(ctx, configPath(c.node, vmid), nil, p)
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
			return nil, fmt.Errorf("lxc: config field %s: %w", key, err)
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
		return nil, fmt.Errorf("lxc: config is required")
	}

	p, err := params.Encode(cfg)
	if err != nil {
		return nil, err
	}

	addIndexedNet(p, cfg.Net)
	addIndexedMountPoint(p, cfg.MP)
	addIndexed(p, "unused", cfg.Unused)

	return p, nil
}

// splitIndexedKey reports whether key is one of Proxmox's numerically
// suffixed CT config families, returning the family prefix and index if
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
	case "mp":
		setIndexedMountPoint(&cfg.MP, index, v)
	case "unused":
		setIndexed(&cfg.Unused, index, v)
	}
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

// setIndexedMountPoint parses v as an mpN property string and stores it
// at (*m)[index], allocating *m on first use.
func setIndexedMountPoint(m *map[int]MountPoint, index int, v string) {
	if *m == nil {
		*m = map[int]MountPoint{}
	}

	mp := MountPoint{}
	_ = property.Unmarshal(v, &mp)
	(*m)[index] = mp
}

// setIndexed sets (*m)[index] = v, allocating *m on first use.
func setIndexed(m *map[int]string, index int, v string) {
	if *m == nil {
		*m = map[int]string{}
	}

	(*m)[index] = v
}

// addIndexedNet adds one "net<index>" entry to p per key in m,
// serializing each network interface back to its property string.
func addIndexedNet(p map[string]string, m map[int]Net) {
	for idx, v := range m {
		p["net"+strconv.Itoa(idx)] = v.String()
	}
}

// addIndexedMountPoint adds one "mp<index>" entry to p per key in m,
// serializing each mount point back to its property string.
func addIndexedMountPoint(p map[string]string, m map[int]MountPoint) {
	for idx, v := range m {
		p["mp"+strconv.Itoa(idx)] = v.String()
	}
}

// addIndexed adds one "prefix+index" entry to p per key in m.
func addIndexed(p map[string]string, prefix string, m map[int]string) {
	for idx, v := range m {
		p[prefix+strconv.Itoa(idx)] = v
	}
}
