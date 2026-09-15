// Package storage provides access to the Proxmox VE per-node storage API
// (endpoints under /nodes/{node}/storage): each storage's status on that
// node, its content (volumes), and backup retention pruning.
//
// This intentionally stops short of the full endpoint surface under
// /nodes/{node}/storage/{storage}/*: rrd/rrddata (graph data meant for the
// web UI), upload (multipart file upload), download-url and
// oci-registry-pull (server-side fetch of external content into a
// storage), file-restore (single-file restore from a backup, a binary
// download), import-metadata/identity (import-specific introspection),
// and content's copy (Proxmox's own source marks it "experimental - do
// not use"). None of these fit this client's JSON-envelope
// request/response model as cleanly as the rest of the API, or are worth
// the surface area yet.
package storage

import (
	"context"

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
}

// Client provides access to the /nodes/{node}/storage resource tree.
// Every method/accessor takes the target node's name as a call argument,
// since this package has no persistent per-node scope of its own.
type Client struct {
	client Getter
}

// New returns a new storage client backed by the given root client.
func New(c Getter) *Client {
	return &Client{client: c}
}

// List retrieves the given node's view of every storage via
// GET /nodes/{node}/storage. opts may be nil to request every storage
// the caller has access to, unfiltered.
func (c *Client) List(ctx context.Context, node string, opts *ListOptions) ([]Storage, error) {
	var p map[string]string
	if opts != nil {
		var err error
		p, err = params.Encode(opts)
		if err != nil {
			return nil, err
		}
	}

	var storages []Storage
	if err := c.client.Get(ctx, "/nodes/"+node+"/storage", &storages, p); err != nil {
		return nil, err
	}

	return storages, nil
}

// Status retrieves a single storage's status on the given node via
// GET /nodes/{node}/storage/{storage}/status. The response omits the
// storage id (it's already the URL path segment), so Status fills it in
// manually.
func (c *Client) Status(ctx context.Context, node, storageID string) (*Storage, error) {
	s := &Storage{}
	if err := c.client.Get(ctx, "/nodes/"+node+"/storage/"+storageID+"/status", s, nil); err != nil {
		return nil, err
	}
	s.Storage = storageID

	return s, nil
}

// Content returns an accessor for the
// /nodes/{node}/storage/{storage}/content resource tree, a storage's
// volumes.
func (c *Client) Content() *contentResource {
	return &contentResource{client: c.client}
}

// PruneBackups returns an accessor for the
// /nodes/{node}/storage/{storage}/prunebackups resource tree, backup
// retention pruning.
func (c *Client) PruneBackups() *pruneBackupsResource {
	return &pruneBackupsResource{client: c.client}
}
