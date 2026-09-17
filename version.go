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
	"context"
)

// Version contains the Proxmox VE version information.
type Version struct {
	// Release is the version string, e.g. "8.2.2".
	Release string `json:"release,omitempty" url:"release,omitempty"`
	// Version is the Proxmox VE version string, e.g. "8.2.2".
	Version string `json:"version,omitempty" url:"version,omitempty"`
	// Repoid is the git commit hash of the running pve-manager.
	Repoid string `json:"repoid,omitempty" url:"repoid,omitempty"`
}

// Version retrieves the version information of the Proxmox VE server.
func (c *Client) Version(ctx context.Context) (*Version, error) {
	v := &Version{}
	if err := c.Get(ctx, "/version", v, nil); err != nil {
		return nil, err
	}
	return v, nil
}
