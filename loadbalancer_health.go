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

package proxmox

import (
	"sync"
	"time"
)

const (
	// defaultMaxFailures matches resty's WeightedRoundRobin default.
	defaultMaxFailures = 5
	// defaultRecovery matches resty's WeightedRoundRobin default.
	defaultRecovery = 120 * time.Second
)

// health is a passive circuit breaker keyed by base URL, mirroring
// resty.WeightedRoundRobin's semantics (docs/node-lb.md §7.1): it never
// probes on its own, only reacts to Feedback from real traffic. There is no
// half-open probe — the recovery ticker closes every open endpoint at once,
// and the first real request routed after recovery is the probe.
type health struct {
	mu          sync.Mutex
	maxFailures int
	failures    map[string]int
	open        map[string]bool

	ticker    *time.Ticker
	done      chan struct{}
	closeOnce sync.Once
}

// newHealth builds a health breaker with the given failure threshold and
// recovery interval, defaulting either when non-positive, and starts its
// recovery ticker goroutine.
func newHealth(maxFailures int, recovery time.Duration) *health {
	if maxFailures <= 0 {
		maxFailures = defaultMaxFailures
	}
	if recovery <= 0 {
		recovery = defaultRecovery
	}

	h := &health{
		maxFailures: maxFailures,
		failures:    make(map[string]int),
		open:        make(map[string]bool),
		ticker:      time.NewTicker(recovery),
		done:        make(chan struct{}),
	}

	go h.tick()

	return h
}

// tick runs the recovery loop until Close.
func (h *health) tick() {
	for {
		select {
		case <-h.ticker.C:
			h.recover()
		case <-h.done:
			return
		}
	}
}

// recover closes every open endpoint and zeroes its failure counter.
func (h *health) recover() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.failures = make(map[string]int)
	h.open = make(map[string]bool)
}

// usable reports whether baseURL is not in the breaker's open state.
func (h *health) usable(baseURL string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	return !h.open[baseURL]
}

// observe records one request outcome for baseURL. A success resets its
// failure counter; a failure increments it and opens the breaker once
// maxFailures is reached.
func (h *health) observe(baseURL string, success bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if success {
		delete(h.failures, baseURL)
		delete(h.open, baseURL)

		return
	}

	h.failures[baseURL]++
	if h.failures[baseURL] >= h.maxFailures {
		h.open[baseURL] = true
	}
}

// Close stops the recovery ticker goroutine. Safe to call more than once.
func (h *health) Close() error {
	h.closeOnce.Do(func() {
		close(h.done)
		h.ticker.Stop()
	})

	return nil
}
