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

package lxc

import (
	"reflect"
	"testing"
)

func TestEncodeConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		if _, err := EncodeConfig(nil); err == nil {
			t.Fatal("EncodeConfig(nil) error = nil, want error")
		}
	})

	t.Run("flat keys, read-only omitted", func(t *testing.T) {
		p, err := EncodeConfig(&Config{
			Hostname:   "ct1",
			Nameserver: "1.1.1.1",
			Parent:     "snap1",
		})
		if err != nil {
			t.Fatalf("EncodeConfig() error = %v", err)
		}

		want := map[string]string{"hostname": "ct1", "nameserver": "1.1.1.1"}
		if !reflect.DeepEqual(p, want) {
			t.Fatalf("EncodeConfig() = %v, want %v", p, want)
		}
	})
}
