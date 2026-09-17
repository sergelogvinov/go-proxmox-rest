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
