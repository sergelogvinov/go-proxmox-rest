//go:build e2e

// Package jobs_e2e exercises the cluster jobs module (schedule-analyze and
// realm-sync) against a live Proxmox VE cluster.
package jobs_e2e

import (
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/cluster/jobs"
	e2e "github.com/sergelogvinov/go-proxmox-rest/tests/e2e"
)

// TestScheduleAnalyze verifies that GET /cluster/jobs/schedule-analyze
// computes the requested number of future runs for a simple schedule. This
// is a pure calculation with no side effects, so it's always safe to run
// for real.
func TestScheduleAnalyze(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	iterations := 3
	events, err := client.Cluster().Jobs().ScheduleAnalyze(ctx, "*/15", &jobs.ScheduleAnalyzeOptions{
		Iterations: &iterations,
	})
	e2e.RequireNoError(t, "schedule-analyze", err)
	if len(events) != iterations {
		t.Errorf("schedule-analyze: got %d events, want %d", len(events), iterations)
	}
	for i, ev := range events {
		if ev.Timestamp == 0 {
			t.Errorf("schedule-analyze: events[%d].Timestamp = 0, want nonzero", i)
		}
		if ev.UTC == "" {
			t.Errorf("schedule-analyze: events[%d].UTC = %q, want non-empty", i, ev.UTC)
		}
	}
}

// TestScheduleAnalyzeValidation verifies client-side validation of
// ScheduleAnalyze's required schedule argument.
func TestScheduleAnalyzeValidation(t *testing.T) {
	cfg := e2e.MustConfig(t)
	if cfg.Parallel {
		t.Parallel()
	}

	client := e2e.NewE2EClient(t, cfg)
	ctx := t.Context()

	_, err := client.Cluster().Jobs().ScheduleAnalyze(ctx, "", nil)
	e2e.RequireError(t, "schedule-analyze with empty schedule", err)
}
