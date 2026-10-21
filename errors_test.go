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
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sergelogvinov/go-proxmox-rest/internal/apierr"
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
		{
			name: "missing task",
			err:  &APIError{StatusCode: http.StatusBadRequest, Message: "Bad Request", Errors: map[string]string{"upid": "no such task"}},
			want: true,
		},
		{
			name: "client-side not found (pools.Get on an unknown pool)",
			err:  fmt.Errorf("pool %q: %w", "dev", apierr.ErrNotFound),
			want: true,
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

// TestAPIErrorMessage drives the client against raw HTTP responses in the
// wire shapes Proxmox errors arrive in: direct from pveproxy (message in
// the reason phrase) and through a reverse proxy (standard reason phrase,
// message in the JSON body).
func TestAPIErrorMessage(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		reason      string
		body        string
		wantMessage string
	}{
		{
			name:        "reverse proxy: message in body",
			status:      http.StatusInternalServerError,
			reason:      "Internal Server Error",
			body:        `{"data":null,"message":"cannot index groups: ha groups have been migrated to rules\n"}`,
			wantMessage: "cannot index groups: ha groups have been migrated to rules",
		},
		{
			name:        "direct pveproxy: message in reason phrase",
			status:      http.StatusInternalServerError,
			reason:      "cannot index groups: ha groups have been migrated to rules",
			body:        `{"data":null}`,
			wantMessage: "500 cannot index groups: ha groups have been migrated to rules",
		},
		{
			name:        "errors map keeps status text",
			status:      http.StatusBadRequest,
			reason:      "Parameter verification failed.",
			body:        `{"data":null,"message":"Parameter verification failed.\n","errors":{"vmid":"invalid format"}}`,
			wantMessage: "Bad Request",
		},
		{
			name:        "non-JSON body falls back to status line",
			status:      http.StatusBadGateway,
			reason:      "Bad Gateway",
			body:        `<html><body>502 Bad Gateway</body></html>`,
			wantMessage: "502 Bad Gateway",
		},
		{
			name:        "empty body falls back to status line",
			status:      http.StatusInternalServerError,
			reason:      "Internal Server Error",
			body:        ``,
			wantMessage: "500 Internal Server Error",
		},
		{
			name:        "whitespace-only message falls back to status line",
			status:      http.StatusInternalServerError,
			reason:      "Internal Server Error",
			body:        `{"data":null,"message":" \n"}`,
			wantMessage: "500 Internal Server Error",
		},
		{
			name:        "non-string message falls back to status line",
			status:      http.StatusInternalServerError,
			reason:      "Internal Server Error",
			body:        `{"data":null,"message":42}`,
			wantMessage: "500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := getRawResponse(t, tt.status, tt.reason, tt.body)

			var apiErr *APIError
			if !asAPIError(err, &apiErr) {
				t.Fatalf("expected *APIError, got %v (%T)", err, err)
			}
			if apiErr.StatusCode != tt.status {
				t.Fatalf("StatusCode = %d, want %d", apiErr.StatusCode, tt.status)
			}
			if apiErr.Message != tt.wantMessage {
				t.Fatalf("Message = %q, want %q", apiErr.Message, tt.wantMessage)
			}
		})
	}
}

// getRawResponse issues a GET through the client against a server that
// answers with a hand-written HTTP/1.1 response, so the reason phrase can
// differ from net/http's standard text for the status.
func getRawResponse(t *testing.T, status int, reason, body string) error {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		conn, buf, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("Hijack() error = %v", err)
			return
		}
		defer func() { _ = conn.Close() }()

		writeRawResponse(buf, status, reason, body)
	}))
	defer srv.Close()

	c, err := New(ClientConfig{}, WithURL(srv.URL),
		WithRetryWaitTime(time.Millisecond), WithRetryMaxWaitTime(time.Millisecond))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	return c.Get(context.Background(), "/cluster/ha/groups", nil, nil)
}

func writeRawResponse(buf *bufio.ReadWriter, status int, reason, body string) {
	fmt.Fprintf(buf, "HTTP/1.1 %d %s\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		status, reason, len(body), body)
	_ = buf.Flush()
}

// TestIsNotFoundBehindReverseProxy covers a 500 whose not-found text is
// only in the JSON body, as seen through a reverse proxy.
func TestIsNotFoundBehindReverseProxy(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "does not exist in body",
			body: `{"data":null,"message":"Configuration file 'nodes/pve1/qemu-server/100.conf' does not exist\n"}`,
			want: true,
		},
		{
			name: "unrelated message in body",
			body: `{"data":null,"message":"cannot index groups: ha groups have been migrated to rules\n"}`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := getRawResponse(t, http.StatusInternalServerError, "Internal Server Error", tt.body)
			if got := IsNotFound(err); got != tt.want {
				t.Fatalf("IsNotFound(%v) = %v, want %v", err, got, tt.want)
			}
		})
	}
}

// TestRetryCondition counts the attempts the client makes for each kind
// of error response: a final not-found 500 must not burn the retry
// budget, while rate limiting and other server errors still retry.
func TestRetryCondition(t *testing.T) {
	const retries = 2

	tests := []struct {
		name         string
		status       int
		body         string
		wantAttempts int
	}{
		{"500 does not exist is final", 500, `{"data":null,"message":"Configuration file 'x' does not exist\n"}`, 1},
		{"404 is final", 404, `{"data":null}`, 1},
		{"400 is final", 400, `{"data":null,"errors":{"vmid":"invalid format"}}`, 1},
		{"plain 500 retries", 500, `{"data":null,"message":"got timeout\n"}`, 1 + retries},
		{"503 retries", 503, `{"data":null}`, 1 + retries},
		{"429 retries", 429, `{"data":null}`, 1 + retries},
		{"SlowDown retries", 400, `{"data":null,"errors":{"api":"SlowDown"}}`, 1 + retries},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempts++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c, err := New(ClientConfig{}, WithURL(srv.URL), WithRetryCount(retries),
				WithRetryWaitTime(time.Millisecond), WithRetryMaxWaitTime(time.Millisecond))
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			defer c.Close()

			if err := c.Get(context.Background(), "/version", nil, nil); err == nil {
				t.Fatal("Get() succeeded, want an error")
			}
			if attempts != tt.wantAttempts {
				t.Fatalf("attempts = %d, want %d", attempts, tt.wantAttempts)
			}
		})
	}
}
