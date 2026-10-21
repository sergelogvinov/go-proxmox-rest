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
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sergelogvinov/go-proxmox-rest/internal/apierr"
	"resty.dev/v3"
)

// APIError represents a Proxmox API error response. It is defined in
// internal/apierr, alongside the classification behind the Is* helpers
// below, so packages that cannot import this one classify errors the
// same way.
type APIError = apierr.APIError

// newAPIError builds an APIError from a resty response.
func newAPIError(res *resty.Response) *APIError {
	e := &APIError{StatusCode: res.StatusCode(), Message: res.Status()}

	// A reverse proxy rewrites pveproxy's reason phrase to the standard
	// text, so prefer the envelope's own message when the body carries one.
	var env struct {
		Errors  map[string]string `json:"errors"`
		Message string            `json:"message"`
	}
	if json.Unmarshal(res.Bytes(), &env) == nil {
		e.Errors = env.Errors
		if len(e.Errors) > 0 {
			e.Message = http.StatusText(e.StatusCode)
		} else if msg := strings.TrimSpace(env.Message); msg != "" {
			e.Message = msg
		}
	}

	return e
}

// IsNotFound reports whether the error indicates that the requested
// resource (or a required Proxmox binary) was not found: a 404, a 500
// whose message or per-field errors say so (Proxmox reports most missing
// guests and configs that way), or a missing task.
func IsNotFound(err error) bool {
	switch apierr.Classify(err) { //nolint:exhaustive
	case apierr.NotFound, apierr.NoSuchTask:
		return true
	default:
		return false
	}
}

// IsRateLimited reports whether the error is a 429 or a Proxmox
// "SlowDown" response.
func IsRateLimited(err error) bool {
	return apierr.Classify(err) == apierr.RateLimited
}

// IsUnexpected reports whether the error is a 5xx from the Proxmox API.
func IsUnexpected(err error) bool {
	var apiErr *APIError
	if asAPIError(err, &apiErr) {
		return apiErr.StatusCode >= http.StatusInternalServerError
	}
	return false
}

// asAPIError is a small helper to avoid importing errors in multiple places.
func asAPIError(err error, target **APIError) bool {
	for err != nil {
		if e, ok := err.(*APIError); ok { //nolint:errorlint // direct type assertion is fine here
			*target = e
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
