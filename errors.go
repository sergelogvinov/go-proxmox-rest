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
	"fmt"
	"net/http"
	"strings"

	"resty.dev/v3"
)

// APIError represents a Proxmox API error response.
type APIError struct {
	StatusCode int
	Message    string
	Errors     map[string]string // per-field validation errors
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("proxmox API error %d: %s: %v", e.StatusCode, e.Message, e.Errors)
	}
	return fmt.Sprintf("proxmox API error %d: %s", e.StatusCode, e.Message)
}

// newAPIError builds an APIError from a resty response.
func newAPIError(res *resty.Response) error {
	e := &APIError{StatusCode: res.StatusCode(), Message: res.Status()}

	var env struct {
		Errors map[string]string `json:"errors"`
	}
	if json.Unmarshal(res.Bytes(), &env) == nil {
		e.Errors = env.Errors
		if len(e.Errors) > 0 {
			e.Message = http.StatusText(e.StatusCode)
		}
	}

	return e
}

// IsNotFound reports whether the error indicates that the requested resource
// or required Proxmox binary was not found.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if asAPIError(err, &apiErr) {
		if apiErr.StatusCode == http.StatusNotFound {
			return true
		}

		if apiErr.StatusCode == http.StatusInternalServerError {
			if containsErrorMessage(apiErr.Message, apiErr.Errors) {
				return true
			}
		}
	}
	return false
}

func containsErrorMessage(message string, _ map[string]string) bool {
	message = strings.ToLower(message)
	if strings.Contains(message, "does not exist") ||
		strings.Contains(message, "no such resource") ||
		strings.Contains(message, "no such ha") ||
		strings.Contains(message, "no such file or directory") ||
		strings.Contains(message, "binary not installed:") {
		return true
	}

	return false
}

// IsRateLimited reports whether the error maps to a Proxmox "SlowDown"
// response.
func IsRateLimited(err error) bool {
	var apiErr *APIError
	if asAPIError(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests
	}
	return false
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
