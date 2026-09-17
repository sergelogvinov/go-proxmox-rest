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
	"net/http"
	"testing"
)

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "404",
			err:  &APIError{StatusCode: http.StatusNotFound},
			want: true,
		},
		{
			name: "missing binary in message",
			err: &APIError{
				StatusCode: http.StatusInternalServerError,
				Message:    "500 binary not installed: /usr/bin/ceph-mon",
			},
			want: true,
		},
		{
			name: "other 500",
			err:  &APIError{StatusCode: http.StatusInternalServerError, Message: "internal error"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Fatalf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}
