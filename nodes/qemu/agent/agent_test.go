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

package agent_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	proxmox "github.com/sergelogvinov/go-proxmox-rest"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/qemu/agent"
)

const guestErrorBody = `{"data":{"result":{"error":{"class":"GenericError","desc":"failed to open /mnt/data: Permission denied"}}}}`

// newAgent returns an agent client whose server answers every request with
// body.
func newAgent(t *testing.T, body string) *agent.Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	c, err := proxmox.New(proxmox.ClientConfig{}, proxmox.WithURL(srv.URL))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	return c.Nodes("pve1").Qemu().Agent()
}

// call runs one agent command and returns its value, nil for a command
// that has none.
type call func(ctx context.Context, c *agent.Client) (any, error)

// command is one agent command that unwraps the {"result": ...} envelope,
// with a result it decodes.
type command struct {
	name    string
	call    call
	success string
	want    any
}

func commands() []command {
	return []command{
		{
			name:    "Ping",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return nil, c.Ping(ctx, 100) },
			success: `{}`,
		},
		{
			name:    "GetTime",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.GetTime(ctx, 100) },
			success: `1762500000000000000`,
			want:    int64(1762500000000000000),
		},
		{
			name:    "Info",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.Info(ctx, 100) },
			success: `{"version":"10.0.0","supported_commands":[{"name":"guest-ping","enabled":true,"success-response":true}]}`,
			want: &agent.Info{
				Version:           "10.0.0",
				SupportedCommands: []agent.SupportedCommand{{Name: "guest-ping", Enabled: true, SuccessResponse: true}},
			},
		},
		{
			name:    "FSFreezeStatus",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.FSFreezeStatus(ctx, 100) },
			success: `"frozen"`,
			want:    agent.FSFreezeStateFrozen,
		},
		{
			name:    "FSFreezeFreeze",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.FSFreezeFreeze(ctx, 100) },
			success: `3`,
			want:    3,
		},
		{
			name:    "FSFreezeThaw",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.FSFreezeThaw(ctx, 100) },
			success: `3`,
			want:    3,
		},
		{
			name:    "FSTrim",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.FSTrim(ctx, 100) },
			success: `{"paths":[{"path":"/","trimmed":4096,"minimum":0},{"path":"/boot","error":"not supported"}]}`,
			want: &agent.FSTrimResult{Paths: []agent.FSTrimPathResult{
				{Path: "/", Trimmed: 4096},
				{Path: "/boot", Error: "not supported"},
			}},
		},
		{
			name:    "NetworkGetInterfaces",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.NetworkGetInterfaces(ctx, 100) },
			success: `[{"name":"eth0","hardware-address":"bc:24:11:00:00:01"}]`,
			want:    []agent.NetworkInterface{{Name: "eth0", HardwareAddress: "bc:24:11:00:00:01"}},
		},
		{
			name:    "GetVCPUs",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.GetVCPUs(ctx, 100) },
			success: `[{"logical-id":0,"online":true,"can-offline":false}]`,
			want:    []agent.VCPU{{Online: true}},
		},
		{
			name:    "GetFSInfo",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.GetFSInfo(ctx, 100) },
			success: `[{"name":"sda1","mountpoint":"/","type":"xfs","used-bytes":10,"total-bytes":100}]`,
			want:    []agent.FSInfo{{Name: "sda1", Mountpoint: "/", Type: "xfs", UsedBytes: 10, TotalBytes: 100}},
		},
		{
			name:    "GetMemoryBlocks",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.GetMemoryBlocks(ctx, 100) },
			success: `[{"phys-index":1,"online":true,"can-offline":true}]`,
			want:    []agent.MemoryBlock{{PhysIndex: 1, Online: true, CanOffline: true}},
		},
		{
			name:    "GetMemoryBlockInfo",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.GetMemoryBlockInfo(ctx, 100) },
			success: `{"size":134217728}`,
			want:    &agent.MemoryBlockInfo{Size: 134217728},
		},
		{
			name:    "SuspendHybrid",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return nil, c.SuspendHybrid(ctx, 100) },
			success: `null`,
		},
		{
			name:    "SuspendRAM",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return nil, c.SuspendRAM(ctx, 100) },
			success: `null`,
		},
		{
			name:    "SuspendDisk",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return nil, c.SuspendDisk(ctx, 100) },
			success: `null`,
		},
		{
			name:    "Shutdown",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return nil, c.Shutdown(ctx, 100) },
			success: `null`,
		},
		{
			name:    "GetHostname",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.GetHostname(ctx, 100) },
			success: `{"host-name":"vm100"}`,
			want:    &agent.HostnameInfo{Hostname: "vm100"},
		},
		{
			name:    "GetOSInfo",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.GetOSInfo(ctx, 100) },
			success: `{"id":"fedora","pretty-name":"Fedora Linux"}`,
			want:    &agent.OSInfo{ID: "fedora", PrettyName: "Fedora Linux"},
		},
		{
			name:    "GetUsers",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.GetUsers(ctx, 100) },
			success: `[{"user":"root","login-time":1762500000.5}]`,
			want:    []agent.User{{User: "root", LoginTime: 1762500000.5}},
		},
		{
			name:    "GetTimezone",
			call:    func(ctx context.Context, c *agent.Client) (any, error) { return c.GetTimezone(ctx, 100) },
			success: `{"zone":"Europe/Zurich","offset":3600}`,
			want:    &agent.Timezone{Zone: "Europe/Zurich", Offset: 3600},
		},
	}
}

func isZero(v any) bool {
	return v == nil || reflect.ValueOf(v).IsZero()
}

func TestCommandsGuestError(t *testing.T) {
	for _, tt := range commands() {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.call(t.Context(), newAgent(t, guestErrorBody))

			var guestErr *agent.GuestError
			if !errors.As(err, &guestErr) {
				t.Fatalf("error = %v, want a *GuestError", err)
			}

			if guestErr.Class != "GenericError" || guestErr.Desc != "failed to open /mnt/data: Permission denied" {
				t.Errorf("GuestError = %+v, want the guest's class and desc", guestErr)
			}

			if !isZero(got) {
				t.Errorf("value = %+v on error, want zero", got)
			}
		})
	}
}

func TestCommandsDecodeResult(t *testing.T) {
	for _, tt := range commands() {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.call(t.Context(), newAgent(t, `{"data":{"result":`+tt.success+`}}`))
			if err != nil {
				t.Fatalf("error = %v, want nil", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("value = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestGuestErrorDetection covers what is and is not a guest error, which
// is Proxmox's own rule: an "error" key holding an object with a "desc".
func TestGuestErrorDetection(t *testing.T) {
	getOSInfo := func(ctx context.Context, c *agent.Client) (any, error) { return c.GetOSInfo(ctx, 100) }
	thaw := func(ctx context.Context, c *agent.Client) (any, error) { return c.FSFreezeThaw(ctx, 100) }

	tests := []struct {
		name      string
		call      call
		result    string
		want      any
		wantError bool
	}{
		{name: "error with desc", call: getOSInfo, result: `{"error":{"class":"CommandNotFound","desc":"disabled"}}`, wantError: true},
		{name: "error with desc and no class", call: getOSInfo, result: `{"error":{"desc":"disabled"}}`, wantError: true},
		{name: "empty error object", call: getOSInfo, result: `{"error":{}}`, want: &agent.OSInfo{}},
		{name: "error without desc", call: getOSInfo, result: `{"error":{"class":"CommandNotFound"}}`, want: &agent.OSInfo{}},
		{name: "empty desc", call: getOSInfo, result: `{"error":{"class":"GenericError","desc":""}}`, want: &agent.OSInfo{}},
		{name: "error is a string", call: getOSInfo, result: `{"error":"boom"}`, want: &agent.OSInfo{}},
		{name: "error is null", call: getOSInfo, result: `{"error":null}`, want: &agent.OSInfo{}},
		{name: "error is an array", call: getOSInfo, result: `{"error":[{"desc":"x"}]}`, want: &agent.OSInfo{}},
		{name: "uppercase key", call: getOSInfo, result: `{"ERROR":{"class":"GenericError","desc":"x"}}`, want: &agent.OSInfo{}},
		{name: "capitalised key", call: getOSInfo, result: `{"Error":{"class":"GenericError","desc":"x"}}`, want: &agent.OSInfo{}},
		{name: "capitalised desc", call: getOSInfo, result: `{"error":{"class":"GenericError","Desc":"x"}}`, want: &agent.OSInfo{}},
		{name: "null result", call: thaw, result: `null`, want: 0},
		{name: "missing result", call: thaw, result: ``, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"data":{}}`
			if tt.result != "" {
				body = `{"data":{"result":` + tt.result + `}}`
			}

			got, err := tt.call(t.Context(), newAgent(t, body))

			var guestErr *agent.GuestError
			if errors.As(err, &guestErr) != tt.wantError {
				t.Fatalf("error = %v, want a *GuestError: %v", err, tt.wantError)
			}

			if tt.wantError {
				return
			}

			if err != nil {
				t.Fatalf("error = %v, want nil", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("value = %+v, want %+v", got, tt.want)
			}
		})
	}
}
