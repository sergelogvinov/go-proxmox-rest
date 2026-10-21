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

package firewall

import (
	"context"
	"testing"

	"github.com/sergelogvinov/go-proxmox-rest/types"
)

type call struct {
	method string
	path   string
	params map[string]string
}

// recorder is a Getter that records every request and answers with no
// data.
type recorder struct {
	calls []call
}

func (r *recorder) record(method, path string, p map[string]string) error {
	r.calls = append(r.calls, call{method: method, path: path, params: p})
	return nil
}

func (r *recorder) Get(_ context.Context, path string, _ any, p map[string]string) error {
	return r.record("GET", path, p)
}

func (r *recorder) Create(_ context.Context, path string, _ any, p map[string]string) error {
	return r.record("POST", path, p)
}

func (r *recorder) Update(_ context.Context, path string, _ any, p map[string]string) error {
	return r.record("PUT", path, p)
}

func (r *recorder) Delete(_ context.Context, path string, _ any, p map[string]string) error {
	return r.record("DELETE", path, p)
}

func (r *recorder) last(t *testing.T) call {
	t.Helper()

	if len(r.calls) == 0 {
		t.Fatal("no request was sent")
	}

	return r.calls[len(r.calls)-1]
}

func assertCall(t *testing.T, got call, method, path string, params map[string]string) {
	t.Helper()

	if got.method != method || got.path != path {
		t.Errorf("request = %s %s, want %s %s", got.method, got.path, method, path)
	}

	for k, v := range params {
		if got.params[k] != v {
			t.Errorf("param %q = %q, want %q (all: %v)", k, got.params[k], v, got.params)
		}
	}
}

// bases are the firewall scopes the public packages mount this module
// under; every resource must behave identically beneath each of them.
var bases = []string{
	"/cluster/firewall",
	"/nodes/pve/qemu/100/firewall",
	"/nodes/pve/lxc/200/firewall",
}

func TestRules(t *testing.T) {
	ctx := context.Background()

	for _, base := range bases {
		t.Run(base, func(t *testing.T) {
			r := &recorder{}
			rules := NewRules(r, base+"/rules")

			if err := rules.Create(ctx, &types.RuleOptions{Type: types.RuleTypeIn, Action: "ACCEPT"}); err != nil {
				t.Fatal(err)
			}
			assertCall(t, r.last(t), "POST", base+"/rules", map[string]string{"type": "in", "action": "ACCEPT"})

			if _, err := rules.Get(ctx, 3); err != nil {
				t.Fatal(err)
			}
			assertCall(t, r.last(t), "GET", base+"/rules/3", nil)

			if err := rules.Delete(ctx, 3, "abc"); err != nil {
				t.Fatal(err)
			}
			assertCall(t, r.last(t), "DELETE", base+"/rules/3", map[string]string{"digest": "abc"})
		})
	}
}

func TestRulesValidation(t *testing.T) {
	ctx := context.Background()
	r := &recorder{}
	rules := NewRules(r, "/cluster/firewall/rules")

	if err := rules.Create(ctx, nil); err == nil {
		t.Error("Create(nil) succeeded, want error")
	}

	if err := rules.Create(ctx, &types.RuleOptions{Type: types.RuleTypeIn}); err == nil {
		t.Error("Create without action succeeded, want error")
	}

	if err := rules.Update(ctx, 0, &types.RuleOptions{}); err == nil {
		t.Error("Update without type/action succeeded, want error")
	}

	moveTo := 5
	if err := rules.Update(ctx, 0, &types.RuleOptions{MoveTo: &moveTo}); err != nil {
		t.Errorf("Update with only MoveTo: %v", err)
	}

	if len(r.calls) != 1 {
		t.Errorf("sent %d requests, want only the MoveTo update", len(r.calls))
	}
}

func TestAliases(t *testing.T) {
	ctx := context.Background()

	for _, base := range bases {
		t.Run(base, func(t *testing.T) {
			r := &recorder{}
			aliases := NewAliases(r, base+"/aliases")

			if err := aliases.Create(ctx, &types.AliasOptions{Name: "lan", CIDR: "10.0.0.0/8"}); err != nil {
				t.Fatal(err)
			}
			assertCall(t, r.last(t), "POST", base+"/aliases", map[string]string{"name": "lan", "cidr": "10.0.0.0/8"})

			if err := aliases.Update(ctx, "lan", &types.AliasOptions{CIDR: "10.1.0.0/16"}); err != nil {
				t.Fatal(err)
			}
			assertCall(t, r.last(t), "PUT", base+"/aliases/lan", map[string]string{"cidr": "10.1.0.0/16"})

			if err := aliases.Update(ctx, "lan", &types.AliasOptions{}); err == nil {
				t.Error("Update without cidr succeeded, want error")
			}
		})
	}
}

func TestIPSets(t *testing.T) {
	ctx := context.Background()

	for _, base := range bases {
		t.Run(base, func(t *testing.T) {
			r := &recorder{}
			sets := NewIPSets(r, base+"/ipset")

			// Update reuses Create's POST, keyed by "rename".
			if err := sets.Update(ctx, "old", &types.IPSetOptions{Comment: new("c")}); err != nil {
				t.Fatal(err)
			}
			assertCall(t, r.last(t), "POST", base+"/ipset", map[string]string{"rename": "old", "name": "old"})

			if err := sets.Update(ctx, "old", &types.IPSetOptions{Name: "new"}); err != nil {
				t.Fatal(err)
			}
			assertCall(t, r.last(t), "POST", base+"/ipset", map[string]string{"rename": "old", "name": "new"})

			if err := sets.Delete(ctx, "new", true); err != nil {
				t.Fatal(err)
			}
			assertCall(t, r.last(t), "DELETE", base+"/ipset/new", map[string]string{"force": "1"})

			entries := sets.Entries("new")

			if err := entries.Create(ctx, &types.IPSetEntryOptions{CIDR: "10.0.0.1"}); err != nil {
				t.Fatal(err)
			}
			assertCall(t, r.last(t), "POST", base+"/ipset/new", map[string]string{"cidr": "10.0.0.1"})

			// Delete resends cidr as a parameter.
			if err := entries.Delete(ctx, "10.0.0.1", ""); err != nil {
				t.Fatal(err)
			}
			got := r.last(t)
			assertCall(t, got, "DELETE", base+"/ipset/new/10.0.0.1", map[string]string{"cidr": "10.0.0.1"})

			if _, ok := got.params["digest"]; ok {
				t.Errorf("empty digest was sent: %v", got.params)
			}
		})
	}
}

func TestOptions(t *testing.T) {
	ctx := context.Background()
	r := &recorder{}
	opts := NewOptions[types.GuestOptions](r, "/nodes/pve/qemu/100/firewall/options")

	if err := opts.Update(ctx, nil); err == nil {
		t.Error("Update(nil) succeeded, want error")
	}

	if err := opts.Update(ctx, &types.GuestOptions{Enable: 1}); err != nil {
		t.Fatal(err)
	}
	assertCall(t, r.last(t), "PUT", "/nodes/pve/qemu/100/firewall/options", map[string]string{"enable": "1"})

	if _, err := opts.Get(ctx); err != nil {
		t.Fatal(err)
	}
	assertCall(t, r.last(t), "GET", "/nodes/pve/qemu/100/firewall/options", nil)
}

func TestRefsAndLog(t *testing.T) {
	ctx := context.Background()
	r := &recorder{}

	if _, err := Refs(ctx, r, "/cluster/firewall/refs", ""); err != nil {
		t.Fatal(err)
	}

	if got := r.last(t); got.path != "/cluster/firewall/refs" || len(got.params) != 0 {
		t.Errorf("Refs(\"\") = %+v, want no type param", got)
	}

	if _, err := Refs(ctx, r, "/cluster/firewall/refs", types.RefTypeIPSet); err != nil {
		t.Fatal(err)
	}
	assertCall(t, r.last(t), "GET", "/cluster/firewall/refs", map[string]string{"type": "ipset"})

	if _, err := Log(ctx, r, "/nodes/pve/lxc/200/firewall/log", nil); err != nil {
		t.Fatal(err)
	}

	if got := r.last(t); len(got.params) != 0 {
		t.Errorf("Log(nil) sent params %v", got.params)
	}
}
