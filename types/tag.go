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

package types

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Tags is a Proxmox tag list, encoded on the wire as a single string of
// the form "<tag>[;<tag>...]" (e.g. "prod;web;team-a") — unlike the
// comma-joined convention params.Encode/Decode apply to ordinary
// []string fields. The format recurs across many resources: cluster's
// Resource.Tags and Options.RegisteredTags, pools' PoolMember.Tags,
// nodes/lxc's Config.Tags/Status.Tags, nodes/qemu's
// Config.Tags/Status.Tags, and others.
type Tags []string

// String joins the tags into Proxmox's ";"-separated wire format,
// satisfying fmt.Stringer so params.Encode sends it as a single form
// value instead of comma-joining it like an ordinary []string field.
func (t Tags) String() string {
	return strings.Join(t, ";")
}

// MarshalJSON encodes t into Proxmox's ";"-separated wire format.
func (t Tags) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON parses a Proxmox ";"-separated tags string into t.
func (t *Tags) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("types: tags must be a string: %w", err)
	}

	*t = nil
	if s == "" {
		return nil
	}
	for tag := range strings.SplitSeq(s, ";") {
		*t = append(*t, tag)
	}

	return nil
}
