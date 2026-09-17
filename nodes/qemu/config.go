package qemu

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
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

// UpdateConfig sets a guest's configuration synchronously via
// PUT /nodes/{node}/qemu/{vmid}/config. Prefer UpdateConfigAsync for
// changes involving hotplug or storage allocation, per Proxmox's own
// guidance. cfg.BackgroundDelay and cfg.ImportWorkingStorage are rejected
// by this endpoint (POST-only); leave them unset.
func (c *Client) UpdateConfig(ctx context.Context, node string, vmid int, cfg *Config) error {
	p, err := encodeConfig(cfg)
	if err != nil {
		return err
	}

	return c.client.Update(ctx, configPath(node, vmid), nil, p)
}

// UpdateConfigAsync sets a guest's configuration via
// POST /nodes/{node}/qemu/{vmid}/config, as a background task — the
// endpoint to prefer for changes involving hotplug or storage
// allocation. Returns the task's UPID.
func (c *Client) UpdateConfigAsync(ctx context.Context, node string, vmid int, cfg *Config) (string, error) {
	p, err := encodeConfig(cfg)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Create(ctx, configPath(node, vmid), &upid, p); err != nil {
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

	addIndexed(p, "net", cfg.Net)
	addIndexed(p, "ide", cfg.IDE)
	addIndexed(p, "sata", cfg.SATA)
	addIndexed(p, "scsi", cfg.SCSI)
	addIndexed(p, "virtio", cfg.VirtIO)
	addIndexed(p, "virtiofs", cfg.VirtioFS)
	addIndexed(p, "unused", cfg.Unused)
	addIndexed(p, "usb", cfg.USB)
	addIndexed(p, "hostpci", cfg.HostPCI)
	addIndexed(p, "serial", cfg.Serial)
	addIndexed(p, "parallel", cfg.Parallel)
	addIndexed(p, "ipconfig", cfg.IPConfig)
	addIndexed(p, "numa", cfg.NUMA)

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
		setIndexed(&cfg.Net, index, v)
	case "ide":
		setIndexed(&cfg.IDE, index, v)
	case "sata":
		setIndexed(&cfg.SATA, index, v)
	case "scsi":
		setIndexed(&cfg.SCSI, index, v)
	case "virtio":
		setIndexed(&cfg.VirtIO, index, v)
	case "virtiofs":
		setIndexed(&cfg.VirtioFS, index, v)
	case "unused":
		setIndexed(&cfg.Unused, index, v)
	case "usb":
		setIndexed(&cfg.USB, index, v)
	case "hostpci":
		setIndexed(&cfg.HostPCI, index, v)
	case "serial":
		setIndexed(&cfg.Serial, index, v)
	case "parallel":
		setIndexed(&cfg.Parallel, index, v)
	case "ipconfig":
		setIndexed(&cfg.IPConfig, index, v)
	case "numa":
		setIndexed(&cfg.NUMA, index, v)
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
