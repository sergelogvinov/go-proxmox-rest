package nodes

import (
	"context"
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Config describes a node's persistent configuration
// (/etc/pve/nodes/{node}/config), as returned by GET /nodes/{node}/config
// and applied by PUT /nodes/{node}/config.
//
// It merges the read and write shapes into a single type, matching
// cluster.Options/firewall.Options: every field is optional, Proxmox only
// includes a key in the GET response once it has been explicitly set, and
// PUT only changes fields that are non-zero (via the `url` tag) — to
// explicitly reset a field to its default, list it in Delete instead of
// trying to send a zero value.
//
// WakeOnLan, ACME and the ACMEDomain* fields are Proxmox property strings
// (e.g. "mac=AA:BB:CC:DD:EE:FF,broadcast-address=192.168.1.255"); unlike
// some other endpoints, Proxmox's node config handler
// (PVE::API2::NodeConfig::get_config) returns the raw config file value
// as-is without expanding it into a nested object, so they are kept as
// opaque strings on both read and write.
type Config struct {
	// Description is shown in the web UI's node notes panel and saved as
	// "#"-prefixed comment lines at the top of the config file. Proxmox
	// always appends a trailing "\n" when reassembling those lines on
	// read (PVE::JSONSchema::parse_config), so a single-line value
	// written here comes back from Config with one trailing newline.
	Description string `json:"description,omitempty" url:"description,omitempty"`
	// StartallOnbootDelay is the initial delay, in seconds, before
	// starting all guests with on-boot enabled (0-300, default 0).
	StartallOnbootDelay int `json:"startall-onboot-delay,omitempty" url:"startall-onboot-delay,omitempty"`
	// BallooningTarget is the RAM usage target for ballooning, as a
	// percentage of total memory (0-100, default 80).
	BallooningTarget int `json:"ballooning-target,omitempty" url:"ballooning-target,omitempty"`
	// WakeOnLan is the node's wake-on-LAN property string, e.g.
	// "mac=AA:BB:CC:DD:EE:FF".
	WakeOnLan string `json:"wakeonlan,omitempty" url:"wakeonlan,omitempty"`
	// ACME is the node's ACME account/domain-list property string.
	ACME string `json:"acme,omitempty" url:"acme,omitempty"`
	// Location overrides the datacenter-wide default location for this
	// node.
	Location string `json:"location,omitempty" url:"location,omitempty"`
	// ACMEDomain0-5 are up to six additional ACME domain/plugin property
	// strings, e.g. "example.com,plugin=dns-provider".
	ACMEDomain0 string `json:"acmedomain0,omitempty" url:"acmedomain0,omitempty"`
	ACMEDomain1 string `json:"acmedomain1,omitempty" url:"acmedomain1,omitempty"`
	ACMEDomain2 string `json:"acmedomain2,omitempty" url:"acmedomain2,omitempty"`
	ACMEDomain3 string `json:"acmedomain3,omitempty" url:"acmedomain3,omitempty"`
	ACMEDomain4 string `json:"acmedomain4,omitempty" url:"acmedomain4,omitempty"`
	ACMEDomain5 string `json:"acmedomain5,omitempty" url:"acmedomain5,omitempty"`
	// Delete lists properties to reset to their default value.
	// Write-only (PUT); Proxmox never returns it from GET — tagged
	// "writeonly" so Decode skips it even if a future GET response
	// happened to echo a "delete" key back.
	Delete []string `json:"-" url:"delete,omitempty,writeonly"`
	// Digest is the configuration digest, usable on a subsequent Update
	// to guard against concurrent changes.
	Digest string `json:"digest,omitempty" url:"digest,omitempty"`
}

// encode converts the config to form parameters.
func (cfg *Config) encode() (map[string]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nodes: config is required")
	}

	return params.Encode(cfg)
}

// Config retrieves the node's persistent configuration via
// GET /nodes/{node}/config.
func (c *Client) Config(ctx context.Context) (*Config, error) {
	cfg := &Config{}
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/config", cfg, nil); err != nil {
		return nil, err
	}

	return cfg, nil
}

// UpdateConfig applies cfg to the node's persistent configuration via
// PUT /nodes/{node}/config. Only non-zero fields are sent; use
// Config.Delete to explicitly reset a field to its default.
func (c *Client) UpdateConfig(ctx context.Context, cfg *Config) error {
	p, err := cfg.encode()
	if err != nil {
		return err
	}

	return c.client.Update(ctx, "/nodes/"+c.node+"/config", nil, p)
}
