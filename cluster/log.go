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
