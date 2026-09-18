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

// Package jobs provides access to the Proxmox VE cluster-wide job API
// (endpoints under /cluster/jobs). This currently covers schedule-analyze
// (simulating future runs of a calendar-event schedule) and realm-sync
// (LDAP/AD realm synchronization jobs, behind the RealmSync() accessor) —
// the only two job families Proxmox exposes here. vzdump backup jobs and
// storage replication jobs live under /cluster/backup and
// /cluster/replication respectively (see the sibling backup and
// replication packages), not here.
package jobs

import (
	"context"
	"fmt"
	"maps"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the jobs package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
}

// Client provides access to the /cluster/jobs resource tree.
type Client struct {
	client Getter
}

// New returns a new jobs client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// RealmSync returns an accessor for the /cluster/jobs/realm-sync resource
// tree: listing, creating, updating and deleting LDAP/AD realm
// synchronization jobs.
func (c *Client) RealmSync() *realmSyncResource {
	return &realmSyncResource{client: c.client}
}

// ScheduleAnalyzeOptions holds the optional parameters for
// Client.ScheduleAnalyze.
type ScheduleAnalyzeOptions struct {
	// StartTime is the UNIX timestamp to start the calculation from.
	// Defaults to the current time.
	StartTime *int64 `url:"starttime,omitempty"`
	// Iterations is the number of events to simulate and return
	// (1-100, default 10).
	Iterations *int `url:"iterations,omitempty"`
}

// ScheduleEvent is a single future run computed by Client.ScheduleAnalyze.
type ScheduleEvent struct {
	// Timestamp is the UNIX timestamp of the run.
	Timestamp int64 `json:"timestamp,omitempty" url:"timestamp,omitempty"`
	// UTC is the same run's time, as a human-readable UTC string.
	UTC string `json:"utc,omitempty" url:"utc,omitempty"`
}

// ScheduleAnalyze computes the next scheduled run times for a
// systemd-calendar-event-style schedule via
// GET /cluster/jobs/schedule-analyze.
//
// This is a pure calculation: schedule need not belong to any actual
// configured job.
//
// No fixed Proxmox privilege is required; the endpoint is declared
// user => all. The URL is GET /cluster/jobs/schedule-analyze.
func (c *Client) ScheduleAnalyze(ctx context.Context, schedule string, opts *ScheduleAnalyzeOptions) ([]ScheduleEvent, error) {
	if schedule == "" {
		return nil, fmt.Errorf("jobs: schedule is required")
	}

	p := map[string]string{"schedule": schedule}
	if opts != nil {
		encoded, err := params.Encode(opts)
		if err != nil {
			return nil, err
		}
		maps.Copy(p, encoded)
	}

	var events []ScheduleEvent
	if err := c.client.Get(ctx, "/cluster/jobs/schedule-analyze", &events, p); err != nil {
		return nil, err
	}

	return events, nil
}
