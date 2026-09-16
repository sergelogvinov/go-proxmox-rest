package qemu

import (
	"context"
	"strconv"
)

// Feature is a guest capability Client.Feature can check.
type Feature string

const (
	FeatureSnapshot Feature = "snapshot"
	FeatureClone    Feature = "clone"
	FeatureCopy     Feature = "copy"
)

// FeatureResult reports whether a guest supports a given Feature, as
// returned by Client.Feature.
type FeatureResult struct {
	// HasFeature is true if the guest (or its storage) supports the
	// checked feature.
	HasFeature bool `json:"hasFeature,omitempty" url:"hasFeature,omitempty"`
	// Nodes lists the nodes the guest's storage is shared with —
	// where the feature would also be usable.
	Nodes []string `json:"nodes,omitempty" url:"nodes,omitempty"`
}

// Feature checks whether a guest supports a given feature via
// GET /nodes/{node}/qemu/{vmid}/feature. snapname checks the feature
// against a specific snapshot's state instead of the guest's current
// one; pass "" to check the current state.
func (c *Client) Feature(ctx context.Context, node string, vmid int, feature Feature, snapname string) (*FeatureResult, error) {
	p := map[string]string{"feature": string(feature)}
	if snapname != "" {
		p["snapname"] = snapname
	}

	result := &FeatureResult{}
	if err := c.client.Get(ctx, "/nodes/"+node+"/qemu/"+strconv.Itoa(vmid)+"/feature", result, p); err != nil {
		return nil, err
	}

	return result, nil
}
