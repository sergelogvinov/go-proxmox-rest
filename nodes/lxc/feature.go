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
