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
	"context"
	"strconv"

	"github.com/sergelogvinov/go-proxmox-rest/types"
)

// Feature and FeatureResult are identical between nodes/lxc and
// nodes/qemu (see types/feature.go's doc comment). They live in the
// shared types package and are re-exported here as aliases so call
// sites read as lxc.Feature, lxc.FeatureResult, ... .
type (
	Feature       = types.Feature
	FeatureResult = types.FeatureResult
)

const (
	FeatureSnapshot = types.FeatureSnapshot
	FeatureClone    = types.FeatureClone
	FeatureCopy     = types.FeatureCopy
)

// Feature checks whether a container supports a given feature via
// GET /nodes/{node}/lxc/{vmid}/feature. snapname checks the feature
// against a specific snapshot's state instead of the container's current
// one; pass "" to check the current state.
//
// +proxmox:rbac:path=/vms/{vmid},method=GET,privs=VM.Audit,match=all
func (c *Client) Feature(ctx context.Context, vmid int, feature Feature, snapname string) (*FeatureResult, error) {
	p := map[string]string{"feature": string(feature)}
	if snapname != "" {
		p["snapname"] = snapname
	}

	result := &FeatureResult{}
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/lxc/"+strconv.Itoa(vmid)+"/feature", result, p); err != nil {
		return nil, err
	}

	return result, nil
}
