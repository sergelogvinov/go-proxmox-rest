package cluster

import (
	"context"
)

// Tasks retrieves the recent task list aggregated across every cluster
// node via GET /cluster/tasks. The result includes only the tasks the
// caller is allowed to see: everything with Sys.Audit on "/", or their own
// tasks otherwise.
func (c *Client) Tasks(ctx context.Context) ([]Task, error) {
	var tasks []Task
	if err := c.client.Get(ctx, "/cluster/tasks", &tasks, nil); err != nil {
		return nil, err
	}

	return tasks, nil
}
