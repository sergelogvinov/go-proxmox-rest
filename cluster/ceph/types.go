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
