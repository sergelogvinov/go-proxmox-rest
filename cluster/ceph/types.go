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

package ceph

import "github.com/sergelogvinov/go-proxmox-rest/types"

// Status, Health, MonMap, OSDMap, PGState, PGMap and MgrMap describe the
// Ceph status returned by GET /cluster/ceph/status. Proxmox's per-node
// GET /nodes/{node}/ceph/status returns the identical shape, so these
// live in the shared types package (see its doc comment) and are
// re-exported here as aliases so call sites keep reading as ceph.Status,
// ceph.Health, ... .
type (
	Status  = types.Status
	Health  = types.Health
	MonMap  = types.MonMap
	OSDMap  = types.OSDMap
	PGState = types.PGState
	PGMap   = types.PGMap
	MgrMap  = types.MgrMap
)
