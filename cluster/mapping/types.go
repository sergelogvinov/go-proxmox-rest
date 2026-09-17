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

package mapping

import "net/url"

// Check is a single diagnostic entry for a mapping's configuration on a
// specific node, only present when List is called with a check-node.
type Check struct {
	// Severity is "warning" or "error".
	Severity string `json:"severity,omitempty" url:"severity,omitempty"`
	// Message describes the problem.
	Message string `json:"message,omitempty" url:"message,omitempty"`
}

// toValues merges a set of already form-encoded scalar parameters with a
// mapping's "map" entries, sent as repeated "map" values rather than a
// single comma-joined one (see Getter's doc comment for why).
func toValues(scalars map[string]string, mapEntries []string) url.Values {
	values := make(url.Values, len(scalars)+1)
	for k, v := range scalars {
		values.Set(k, v)
	}
	for _, entry := range mapEntries {
		values.Add("map", entry)
	}

	return values
}
