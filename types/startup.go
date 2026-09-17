package types

import (
	"encoding/json"
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/property"
)

// startup.go: Startup is identical between nodes/qemu and nodes/lxc —
// both guests accept the same startup/shutdown ordering property string
// (see pct.conf's and qm.conf's shared "startup" option), the
// guest-scope counterpart to the Feature/Rule/... sharing already
// documented for the per-guest capabilities and firewalls.

// Startup describes a guest's startup and shutdown ordering. Proxmox
// accepts the order either as a bare first number or as order=<number>,
// followed by optional up/down delays in seconds.
type Startup struct {
	Order *int `cfg:"order,omitempty,default"`
	Up    *int `cfg:"up,omitempty"`
	Down  *int `cfg:"down,omitempty"`
}

// String converts the startup configuration to Proxmox's property-string
// format.
func (s Startup) String() string {
	value, _ := property.Marshal(s)
	return value
}

// UnmarshalJSON converts Proxmox's startup property string into Startup.
func (s *Startup) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("startup must be a property string: %w", err)
	}

	*s = Startup{}
	return property.Unmarshal(value, s)
}
