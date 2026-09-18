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

package cluster

import (
	"context"
	"strconv"
)

// Log retrieves recent cluster log entries via GET /cluster/log.
//
// limit, when non-zero, caps the number of entries returned (most recent
// first); zero requests every entry the server has. Without Sys.Syslog on
// "/", Proxmox restricts the result to the caller's own log entries.
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Syslog,match=any
func (c *Client) Log(ctx context.Context, limit int) ([]LogEntry, error) {
	var params map[string]string
	if limit != 0 {
		params = map[string]string{"max": strconv.Itoa(limit)}
	}

	var entries []LogEntry
	if err := c.client.Get(ctx, "/cluster/log", &entries, params); err != nil {
		return nil, err
	}

	return entries, nil
}
