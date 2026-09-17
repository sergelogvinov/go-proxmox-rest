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

package nodes

import (
	"context"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// SyslogEntry is a single system log line, as returned by
// GET /nodes/{node}/syslog.
type SyslogEntry struct {
	// N is the line number.
	N int64 `json:"n,omitempty" url:"n,omitempty"`
	// T is the line text.
	T string `json:"t,omitempty" url:"t,omitempty"`
}

// SyslogOptions filters the log lines returned by Client.Syslog. A nil
// *SyslogOptions (or the zero value) requests Proxmox's default window.
type SyslogOptions struct {
	// Start is the first line number to return.
	Start int `url:"start,omitempty"`
	// Limit caps the number of lines returned.
	Limit int `url:"limit,omitempty"`
	// Since restricts the result to log lines from this date-time
	// onward, formatted "YYYY-MM-DD[ HH:MM[:SS]]".
	Since string `url:"since,omitempty"`
	// Until restricts the result to log lines up to this date-time,
	// formatted "YYYY-MM-DD[ HH:MM[:SS]]".
	Until string `url:"until,omitempty"`
	// Service filters by systemd service/unit name, e.g. "pvedaemon".
	// Proxmox aliases the values "postfix" and "sshd" to the actual
	// unit names "postfix@-" and "ssh" respectively.
	Service string `url:"service,omitempty"`
}

// Syslog retrieves system log lines for the node via GET /nodes/{node}/syslog.
// opts may be nil to request Proxmox's default window (its most recent
// lines).
func (c *Client) Syslog(ctx context.Context, opts *SyslogOptions) ([]SyslogEntry, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var entries []SyslogEntry
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/syslog", &entries, p); err != nil {
		return nil, err
	}

	return entries, nil
}
