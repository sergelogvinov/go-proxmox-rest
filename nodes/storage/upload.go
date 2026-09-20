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
	"io"

	"github.com/sergelogvinov/go-proxmox-rest/internal/params"
)

// UploadOptions holds the parameters for Client.Upload
// (POST /nodes/{node}/storage/{storage}/upload).
type UploadOptions struct {
	// Content restricts the destination content type to "iso",
	// "vztmpl", or "import", matching DownloadURLOptions.Content.
	// Required.
	Content string `url:"content"`
	// Filename is the destination file name; Proxmox derives it from
	// the upload's own Content-Disposition rather than a separate
	// parameter, so it is carried as File's multipart filename, not a
	// form field. Required.
	Filename string
	// File is the file content to upload. Required. Upload does not
	// close it; the caller retains ownership.
	File io.Reader
	// Checksum is the expected checksum of the uploaded file, in the
	// algorithm named by ChecksumAlgorithm.
	Checksum string `url:"checksum,omitempty"`
	// ChecksumAlgorithm is the hash algorithm Checksum is expressed in.
	// Setting it requires Checksum to be set too.
	ChecksumAlgorithm ChecksumAlgorithm `url:"checksum-algorithm,omitempty"`
}

// Upload uploads a file directly into storage via multipart POST
// /nodes/{node}/storage/{storage}/upload, streaming opts.File from the
// caller — unlike DownloadURL, which fetches server-side from a URL.
// Returns the upload task's UPID.
//
// +proxmox:rbac:path=/storage/{storage},method=POST,privs=Datastore.AllocateTemplate,match=all
func (c *Client) Upload(ctx context.Context, storageID string, opts *UploadOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("storage: upload options are required")
	}
	if opts.Content == "" {
		return "", fmt.Errorf("storage: upload content is required")
	}
	if opts.Filename == "" {
		return "", fmt.Errorf("storage: upload filename is required")
	}
	if opts.File == nil {
		return "", fmt.Errorf("storage: upload file is required")
	}

	fields, err := params.Encode(opts)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.client.Upload(ctx, "/nodes/"+c.node+"/storage/"+storageID+"/upload", &upid, fields, "filename", opts.Filename, opts.File); err != nil {
		return "", err
	}

	return upid, nil
}
