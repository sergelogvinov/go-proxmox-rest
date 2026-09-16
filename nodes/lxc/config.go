package lxc

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
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
func (c *Client) Config(ctx context.Context, node string, vmid int, opts *ConfigOptions) (*Config, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var raw map[string]json.RawMessage
	if err := c.client.Get(ctx, configPath(node, vmid), &raw, p); err != nil {
		return nil, err
	}

	return decodeConfig(raw)
}

// UpdateConfig sets a container's configuration via
// PUT /nodes/{node}/lxc/{vmid}/config.
func (c *Client) UpdateConfig(ctx context.Context, node string, vmid int, cfg *Config) error {
	p, err := encodeConfig(cfg)
	if err != nil {
		return err
	}

	return c.client.Update(ctx, configPath(node, vmid), nil, p)
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

	addIndexed(p, "net", cfg.Net)
	addIndexed(p, "mp", cfg.MP)
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
		setIndexed(&cfg.Net, index, v)
	case "mp":
		setIndexed(&cfg.MP, index, v)
	case "unused":
		setIndexed(&cfg.Unused, index, v)
	}
}

// setIndexed sets (*m)[index] = v, allocating *m on first use.
func setIndexed(m *map[int]string, index int, v string) {
	if *m == nil {
		*m = map[int]string{}
	}

	(*m)[index] = v
}

// addIndexed adds one "prefix+index" entry to p per key in m.
func addIndexed(p map[string]string, prefix string, m map[int]string) {
	for idx, v := range m {
		p[prefix+strconv.Itoa(idx)] = v
	}
}
