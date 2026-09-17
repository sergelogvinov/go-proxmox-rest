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
