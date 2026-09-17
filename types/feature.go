package types

// feature.go: Feature and FeatureResult are identical between nodes/qemu
// and nodes/lxc — both back onto the same PVE::API2::Qemu/LXC "feature"
// handler shape (feature name in, hasFeature/nodes out), the guest-scope
// counterpart to the Rule/Alias/... sharing already documented for the
// per-guest firewalls.

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
