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

package apierr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{"nil", nil, Other},
		{"non-API error", errors.New("dial tcp: connection refused"), Other},
		{"404", &APIError{StatusCode: http.StatusNotFound}, NotFound},
		{
			"500 does not exist",
			&APIError{StatusCode: 500, Message: "Configuration file 'nodes/pve/qemu-server/100.conf' does not exist"},
			NotFound,
		},
		{
			"500 no such resource behind reverse proxy",
			&APIError{StatusCode: 500, Message: "no such security group 'web'"},
			NotFound,
		},
		{"500 missing binary", &APIError{StatusCode: 500, Message: "500 binary not installed: /usr/bin/ceph-mon"}, NotFound},
		{
			"500 not-found text only in per-field errors",
			&APIError{StatusCode: 500, Message: "Internal Server Error", Errors: map[string]string{"vmid": "VM 100 does not exist"}},
			NotFound,
		},
		{
			"400 not-found text stays a parameter error",
			&APIError{StatusCode: 400, Message: "Bad Request", Errors: map[string]string{"storage": "storage 'x' does not exist"}},
			Other,
		},
		{
			"no such task",
			&APIError{StatusCode: 400, Message: "Bad Request", Errors: map[string]string{"upid": "no such task"}},
			NoSuchTask,
		},
		{"429", &APIError{StatusCode: http.StatusTooManyRequests}, RateLimited},
		{
			"SlowDown in per-field errors",
			&APIError{StatusCode: 500, Message: "Internal Server Error", Errors: map[string]string{"api": "SlowDown"}},
			RateLimited,
		},
		{"plain 500", &APIError{StatusCode: 500, Message: "VM 100 is locked (backup)"}, ServerError},
		{"503", &APIError{StatusCode: http.StatusServiceUnavailable, Message: "Service Unavailable"}, ServerError},
		{"wrapped API error", fmt.Errorf("qemu: config: %w", &APIError{StatusCode: http.StatusNotFound}), NotFound},
		{"wrapped ErrNotFound", fmt.Errorf("pool %q: %w", "dev", ErrNotFound), NotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.err); got != tt.want {
				t.Fatalf("Classify(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}
