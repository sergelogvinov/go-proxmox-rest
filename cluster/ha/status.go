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

package ha

import (
	"context"
	"fmt"
)

// statusResource provides access to GET /cluster/ha/status/current,
// GET /cluster/ha/status/manager_status, and the arm-ha/disarm-ha actions
// (exposed together as Update).
type statusResource struct {
	client Getter
}

// Status returns an accessor for the /cluster/ha/status resource.
func (c *Client) Status() *statusResource {
	return &statusResource{client: c.client}
}

// Current retrieves the live status of every part of the HA stack (quorum,
// the elected CRM master, each node's LRM, HA-managed services, and fencing
// arm state) via GET /cluster/ha/status/current, as one flat, heterogeneous
// list. Which StatusEntry fields are populated depends on its Type.
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Audit,match=all
func (s *statusResource) Current(ctx context.Context) ([]StatusEntry, error) {
	var entries []StatusEntry
	if err := s.client.Get(ctx, "/cluster/ha/status/current", &entries, nil); err != nil {
		return nil, err
	}

	return entries, nil
}

// ManagerStatus retrieves the elected CRM master's persisted status, merged
// with live quorum and per-node LRM info, via
// GET /cluster/ha/status/manager_status.
//
// +proxmox:rbac:path=/,method=GET,privs=Sys.Audit,match=all
func (s *statusResource) ManagerStatus(ctx context.Context) (*ManagerStatus, error) {
	var ms ManagerStatus
	if err := s.client.Get(ctx, "/cluster/ha/status/manager_status", &ms, nil); err != nil {
		return nil, err
	}

	return &ms, nil
}

// Update arms or disarms the cluster-wide HA fencing stack.
//
// armed=true re-arms it via POST /cluster/ha/status/arm-ha, which takes no
// parameters. armed=false disarms it via
// POST /cluster/ha/status/disarm-ha, releasing all watchdogs cluster-wide;
// resourceMode is then required and controls how HA-managed resources are
// treated while disarmed (ResourceModeFreeze leaves their state untouched,
// ResourceModeIgnore drops them from HA tracking until re-armed).
// resourceMode is ignored when armed is true.
//
// +proxmox:rbac:path=/,method=POST,privs=Sys.Console,match=all
func (s *statusResource) Update(ctx context.Context, armed bool, resourceMode ResourceMode) error {
	if armed {
		return s.client.Create(ctx, "/cluster/ha/status/arm-ha", nil, nil)
	}

	if resourceMode == "" {
		return fmt.Errorf("ha: resource mode is required to disarm")
	}

	return s.client.Create(ctx, "/cluster/ha/status/disarm-ha", nil, map[string]string{
		"resource-mode": string(resourceMode),
	})
}
