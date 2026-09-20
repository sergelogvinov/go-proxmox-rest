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

// Package storage provides access to the Proxmox VE per-node storage API
// (endpoints under /nodes/{node}/storage): each storage's status on that
// node, its content (volumes), and backup retention pruning.
//
// This intentionally stops short of most of the remaining endpoint
// surface under /nodes/{node}/storage/{storage}/*: rrd/rrddata (graph
// data meant for the web UI), file-restore (single-file restore from a
// backup, a binary download), and import-metadata (introspection of an
// importable guest, tied to the import workflow this client doesn't
// otherwise support yet). None of these fit this client's JSON-envelope
// request/response model as cleanly as the rest of the API, or are
// worth the surface area yet. Content().Copy, DownloadURL,
// OCIRegistryPull, Upload, and Identity are the exceptions: despite
// Proxmox's own source marking Copy "experimental - do not use", it has
// shipped unchanged across many releases and is the only way to copy or
// move an existing volume (e.g. across nodes) without going through the
// guest config; DownloadURL, OCIRegistryPull, and Upload are the ways to
// get a file into storage without an existing guest — DownloadURL and
// OCIRegistryPull as server-side fetches, Upload as a client-supplied
// multipart file — and all three, like Copy, return a plain UPID and
// fit the envelope model just as well as any other task-returning
// write; Identity is a plain GET returning a small {id, type} object
// and fits the model as directly as Status does. All five are included,
// Copy with its caveat documented on its own method.
package storage

import (
	"context"
	"io"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// Getter is the subset of the root client used by this package. It is
// satisfied by *proxmox.Client, which keeps the storage package decoupled
// from the root package (no import cycle).
type Getter interface {
	Get(ctx context.Context, path string, out any, params map[string]string) error
	Create(ctx context.Context, path string, out any, params map[string]string) error
	Update(ctx context.Context, path string, out any, params map[string]string) error
	Delete(ctx context.Context, path string, out any, params map[string]string) error
	// Upload performs a multipart/form-data POST, used only by Upload.
	Upload(ctx context.Context, path string, out any, fields map[string]string, fieldName, fileName string, file io.Reader) error
}

// Client provides access to the /nodes/{node}/storage resource tree,
// scoped to the node given to New.
type Client struct {
	client Getter
	node   string
}

// New returns a new storage client backed by the given root client,
// scoped to node.
func New(c Getter, node string) *Client {
	return &Client{client: c, node: node}
}

// List retrieves the node's view of every storage via
// GET /nodes/{node}/storage. opts may be nil to request every storage
// the caller has access to, unfiltered.
// This user=>all endpoint has no fixed privilege; results are filtered to
// storage paths with either Datastore.Audit or Datastore.AllocateSpace.
//
// +proxmox:rbac:path=/storage/{storage},method=GET,privs=Datastore.Audit;Datastore.AllocateSpace,match=any
func (c *Client) List(ctx context.Context, opts *ListOptions) ([]Storage, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var storages []Storage
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/storage", &storages, p); err != nil {
		return nil, err
	}

	return storages, nil
}

// Status retrieves a single storage's status on the node via
// GET /nodes/{node}/storage/{storage}/status. The response omits the
// storage id (it's already the URL path segment), so Status fills it in
// manually.
//
// +proxmox:rbac:path=/storage/{storage},method=GET,privs=Datastore.Audit;Datastore.AllocateSpace,match=any
func (c *Client) Status(ctx context.Context, storageID string) (*Storage, error) {
	s := &Storage{}
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/storage/"+storageID+"/status", s, nil); err != nil {
		return nil, err
	}
	s.Storage = storageID

	return s, nil
}

// Identity retrieves a storage instance's plugin-assigned identity via
// GET /nodes/{node}/storage/{storage}/identity. It's meaningful mainly
// for import-capable storage plugins (e.g. "esxi"), which use it to
// recognize the same backing instance across config changes; other
// plugin types may return an ID with no comparable stability guarantee.
//
// +proxmox:rbac:path=/storage/{storage},method=GET,privs=Datastore.Audit;Datastore.AllocateSpace,match=any
func (c *Client) Identity(ctx context.Context, storageID string) (*Identity, error) {
	id := &Identity{}
	if err := c.client.Get(ctx, "/nodes/"+c.node+"/storage/"+storageID+"/identity", id, nil); err != nil {
		return nil, err
	}

	return id, nil
}

// Content returns an accessor for the
// /nodes/{node}/storage/{storage}/content resource tree, scoped to
// storageID: a storage's volumes.
func (c *Client) Content(storageID string) *contentResource {
	return &contentResource{client: c.client, node: c.node, storageID: storageID}
}

// PruneBackups returns an accessor for the
// /nodes/{node}/storage/{storage}/prunebackups resource tree, backup
// retention pruning.
func (c *Client) PruneBackups() *pruneBackupsResource {
	return &pruneBackupsResource{client: c.client, node: c.node}
}
