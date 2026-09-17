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

// Package cluster provides access to the Proxmox VE cluster API
// (GET/POST endpoints under /cluster).
package cluster

import (
	"context"
	"net/url"

	"github.com/sergelogvinov/go-proxmox-rest/cluster/backup"
	"github.com/sergelogvinov/go-proxmox-rest/cluster/ceph"
	"github.com/sergelogvinov/go-proxmox-rest/cluster/firewall"
	"github.com/sergelogvinov/go-proxmox-rest/cluster/ha"
	"github.com/sergelogvinov/go-proxmox-rest/cluster/jobs"
	"github.com/sergelogvinov/go-proxmox-rest/cluster/mapping"
	"github.com/sergelogvinov/go-proxmox-rest/cluster/qemu"
	"github.com/sergelogvinov/go-proxmox-rest/cluster/replication"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the cluster package decoupled
// from the root package (no import cycle).
//
// CreateValues/UpdateValues (url.Values, supporting multiple values under
// the same key) are carried here, even though only cluster/mapping needs
// them, for the same reason Create/Update/Delete are: so every child
// package can be wired in from the single c.client value below.
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
	CreateValues(ctx context.Context, path string, out any, params url.Values) error
	UpdateValues(ctx context.Context, path string, out any, params url.Values) error
}

// Client provides access to the cluster API section.
type Client struct {
	client Getter
}

// New returns a new cluster client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// HA returns an accessor for the /cluster/ha resource tree, the
// cluster-wide high-availability configuration.
func (c *Client) HA() *ha.Client {
	return ha.New(c.client)
}

// Firewall returns an accessor for the /cluster/firewall resource tree,
// the cluster-wide firewall configuration.
func (c *Client) Firewall() *firewall.Client {
	return firewall.New(c.client)
}

// Jobs returns an accessor for the /cluster/jobs resource tree:
// schedule-analyze and realm-sync jobs.
func (c *Client) Jobs() *jobs.Client {
	return jobs.New(c.client)
}

// Backup returns an accessor for the /cluster/backup resource tree, the
// cluster-wide vzdump backup job schedule.
func (c *Client) Backup() *backup.Client {
	return backup.New(c.client)
}

// Ceph returns an accessor for the /cluster/ceph resource tree, the
// cluster-wide Ceph status and configuration.
func (c *Client) Ceph() *ceph.Client {
	return ceph.New(c.client)
}

// Mapping returns an accessor for the /cluster/mapping resource tree, the
// cluster-wide hardware mappings (PCI, USB, directory) shared by name
// across nodes.
func (c *Client) Mapping() *mapping.Client {
	return mapping.New(c.client)
}

// Qemu returns an accessor for the /cluster/qemu resource tree: available
// CPU flags and cluster-wide custom CPU model definitions.
func (c *Client) Qemu() *qemu.Client {
	return qemu.New(c.client)
}

// Replication returns an accessor for the /cluster/replication resource
// tree, the cluster-wide storage replication job schedule.
func (c *Client) Replication() *replication.Client {
	return replication.New(c.client)
}
