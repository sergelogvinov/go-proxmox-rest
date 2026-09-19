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

package storage

import (
	"context"
	"fmt"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// ChecksumAlgorithm identifies the hash algorithm used to verify a
// DownloadURLOptions.Checksum.
type ChecksumAlgorithm string

const (
	ChecksumAlgorithmMD5    ChecksumAlgorithm = "md5"
	ChecksumAlgorithmSHA1   ChecksumAlgorithm = "sha1"
	ChecksumAlgorithmSHA224 ChecksumAlgorithm = "sha224"
	ChecksumAlgorithmSHA256 ChecksumAlgorithm = "sha256"
	ChecksumAlgorithmSHA384 ChecksumAlgorithm = "sha384"
	ChecksumAlgorithmSHA512 ChecksumAlgorithm = "sha512"
)

// DownloadURLOptions holds the parameters for Client.DownloadURL
// (POST /nodes/{node}/storage/{storage}/download-url).
type DownloadURLOptions struct {
	// Content restricts the destination content type to "iso",
	// "vztmpl", or "import" — narrower than the general storage
	// content types (see ContentListOptions.Content). Required.
	Content string `url:"content"`
	// Filename is the name of the file to create; Proxmox normalizes
	// it. Required.
	Filename string `url:"filename"`
	// URL is the address to download the file from. Required.
	URL string `url:"url"`
	// Checksum is the expected checksum of the downloaded file, in the
	// algorithm named by ChecksumAlgorithm.
	Checksum string `url:"checksum,omitempty"`
	// ChecksumAlgorithm is the hash algorithm Checksum is expressed in.
	// Setting it requires Checksum to be set too.
	ChecksumAlgorithm ChecksumAlgorithm `url:"checksum-algorithm,omitempty"`
	// Compression decompresses the downloaded file with the named
	// algorithm, e.g. "gz", "lzo", "zst".
	Compression string `url:"compression,omitempty"`
	// VerifyCertificates disables TLS certificate verification for the
	// download when set to a false value. Nil leaves Proxmox's own
	// default (verify) in effect.
	VerifyCertificates *bool `url:"verify-certificates"`
}

// DownloadURL fetches a file from a URL directly into storage via
// POST /nodes/{node}/storage/{storage}/download-url, without routing the
// content through the client. Returns the download task's UPID.
// Besides Datastore.AllocateTemplate on the storage, Proxmox also treats
// this as a (local!) network probe: it additionally requires either
// Sys.AccessNetwork on the node or, for backwards compatibility,
// Sys.Audit/Sys.Modify on "/". The marker records the storage-privilege
// branch; the node-network alternative is documented here since the
// annotation format has no way to express a second, ANDed path.
//
// +proxmox:rbac:path=/storage/{storage},method=POST,privs=Datastore.AllocateTemplate,match=all
func (c *Client) DownloadURL(ctx context.Context, storageID string, opts *DownloadURLOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("storage: download-url options are required")
	}
	if opts.Content == "" {
		return "", fmt.Errorf("storage: download-url content is required")
	}
	if opts.Filename == "" {
		return "", fmt.Errorf("storage: download-url filename is required")
	}
	if opts.URL == "" {
		return "", fmt.Errorf("storage: download-url url is required")
	}

	p, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Create(ctx, "/nodes/"+c.node+"/storage/"+storageID+"/download-url", &upid, p); err != nil {
		return "", err
	}

	return upid, nil
}
