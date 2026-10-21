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

// Package apierr is the single place that decides what a Proxmox API
// error means. It holds the APIError type the root client returns
// (re-exported there as proxmox.APIError) and Classify, which maps any
// error to a Kind. The root package's Is* helpers, its retry policy, and
// the packages that cannot import the root package (task, pools, ...)
// all classify through it, so Proxmox's error texts are matched in one
// table.
package apierr

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
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

// Kind is what an error means to a caller.
type Kind int

const (
	// Other is any error not covered by a more specific Kind, including
	// non-API errors (network failures, encoding errors, ...).
	Other Kind = iota
	// NotFound is a missing resource: a 404, a 500 whose message says
	// so (Proxmox reports most missing guests/configs that way), or an
	// error wrapping ErrNotFound.
	NotFound
	// NoSuchTask is Proxmox's "no such task" answer for a UPID, which a
	// just-started task can briefly give before pvedaemon indexes it.
	NoSuchTask
	// RateLimited is a 429 or a Proxmox "SlowDown" response.
	RateLimited
	// ServerError is any other 5xx. Proxmox uses 500 for most
	// application errors (locks, conflicts, failed commands), so this
	// does not imply the request is safe to repeat.
	ServerError
)

// ErrNotFound marks a not-found condition detected client-side (e.g. an
// empty filtered list), so Classify reports NotFound for it just like
// for an HTTP 404. Wrap it: fmt.Errorf("pool %q: %w", id, ErrNotFound).
var ErrNotFound = errors.New("not found")

// notFoundTexts are the message fragments (lowercase) Proxmox uses for a
// missing resource in a 500 response.
var notFoundTexts = []string{
	"does not exist",
	"no such ", // "no such resource", "no such file or directory", ...
	"binary not installed:",
}

// Classify reports what err means. It looks through wrapped errors for
// an *APIError or ErrNotFound; anything else is Other.
func Classify(err error) Kind {
	if err == nil {
		return Other
	}

	if errors.Is(err, ErrNotFound) {
		return NotFound
	}

	var e *APIError
	if !errors.As(err, &e) {
		return Other
	}

	return e.Kind()
}

// Kind classifies e from its status code, message, and per-field errors.
func (e *APIError) Kind() Kind {
	texts := e.texts()

	switch {
	case containsAny(texts, "no such task"):
		return NoSuchTask
	case e.StatusCode == http.StatusTooManyRequests || containsAny(texts, "slowdown"):
		return RateLimited
	case e.StatusCode == http.StatusNotFound:
		return NotFound
	case e.StatusCode == http.StatusInternalServerError && containsAny(texts, notFoundTexts...):
		return NotFound
	case e.StatusCode >= http.StatusInternalServerError:
		return ServerError
	}

	return Other
}

// texts returns e's message and per-field error values, lowercased.
func (e *APIError) texts() []string {
	texts := make([]string, 0, 1+len(e.Errors))
	texts = append(texts, strings.ToLower(e.Message))

	for _, v := range e.Errors {
		texts = append(texts, strings.ToLower(v))
	}

	return texts
}

// containsAny reports whether any of texts contains any of substrs.
func containsAny(texts []string, substrs ...string) bool {
	for _, t := range texts {
		for _, s := range substrs {
			if strings.Contains(t, s) {
				return true
			}
		}
	}

	return false
}
