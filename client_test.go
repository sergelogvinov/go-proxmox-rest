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

package proxmox

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestUploadNoChunkedEncoding guards against a regression where Upload's
// multipart body was streamed through resty's io.Pipe-based multipart
// writer with no Content-Length, causing Transfer-Encoding: chunked —
// which pveproxy rejects with "501 chunked transfer encoding not
// supported" on the storage upload endpoint. Upload must always send a
// known Content-Length, for both a seekable file (*bytes.Reader here,
// standing in for *os.File) and a plain, non-seekable io.Reader.
func TestUploadNoChunkedEncoding(t *testing.T) {
	tests := []struct {
		name string
		file io.Reader
	}{
		{name: "seekable", file: bytes.NewReader([]byte("fake iso content"))},
		{name: "non-seekable", file: strings.NewReader("fake iso content")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				gotTransferEncoding []string
				gotContentLength    int64
				gotField            string
				gotFileName         string
				gotFileContent      []byte
			)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotTransferEncoding = r.TransferEncoding
				gotContentLength = r.ContentLength

				_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if err != nil {
					t.Fatalf("ParseMediaType() error = %v", err)
				}

				mr := multipart.NewReader(r.Body, params["boundary"])
				for {
					part, err := mr.NextPart()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatalf("NextPart() error = %v", err)
					}

					if part.FormName() == "content" {
						gotField, _ = readAllString(part)
						continue
					}

					gotFileName = part.FileName()
					gotFileContent, _ = io.ReadAll(part)
				}

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":"UPID:node:upload"}`))
			}))
			defer srv.Close()

			c, err := New(ClientConfig{}, WithURL(srv.URL), WithRetryCount(0))
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			defer c.Close()

			var upid string
			if err := c.Upload(context.Background(), "/upload", &upid,
				map[string]string{"content": "iso"}, "filename", "cloud-init.iso", tt.file); err != nil {
				t.Fatalf("Upload() error = %v", err)
			}

			for _, te := range gotTransferEncoding {
				if te == "chunked" {
					t.Fatalf("request sent with Transfer-Encoding: chunked, want a fixed Content-Length")
				}
			}
			if gotContentLength <= 0 {
				t.Fatalf("server saw ContentLength = %d, want > 0", gotContentLength)
			}
			if gotField != "iso" {
				t.Fatalf("field %q = %q, want %q", "content", gotField, "iso")
			}
			if gotFileName != "cloud-init.iso" {
				t.Fatalf("file part filename = %q, want %q", gotFileName, "cloud-init.iso")
			}
			if string(gotFileContent) != "fake iso content" {
				t.Fatalf("file part content = %q, want %q", gotFileContent, "fake iso content")
			}
		})
	}
}

func readAllString(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	return string(b), err
}
