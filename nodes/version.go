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

package nodes

import (
	"context"
)

// Version contains the Proxmox VE version information as reported by a
// single node. Identical in shape to the root Version (GET /version), but
// requested for a specific node rather than whichever node the client
// happens to be talking to.
type Version struct {
	// Release is the current installed Proxmox VE release, e.g. "8.2".
	Release string `json:"release,omitempty" url:"release,omitempty"`
	// Version is the currently installed pve-manager package version.
	Version string `json:"version,omitempty" url:"version,omitempty"`
	// Repoid is the short git commit hash from which this version was
	// built.
	Repoid string `json:"repoid,omitempty" url:"repoid,omitempty"`
}

// Version retrieves the API version details of the node via
// GET /nodes/{node}/version.
func (c *Client) Version(ctx context.Context) (*Version, error) {
	v := &Version{}
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/version", v, nil); err != nil {
		return nil, err
	}

	return v, nil
}
