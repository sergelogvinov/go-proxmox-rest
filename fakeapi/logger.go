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

package fakeapi

import (
	"log"
	"strings"
)

// quietLogger is the resty.Logger Cluster.Client installs. resty warns
// whenever a request carries credentials (a token, here — Cluster.Client
// always sends one) over a plain-HTTP connection; that's real, useful
// advice against a live cluster, but fakeapi's httptest.Server is always
// plain HTTP, so the warning is unconditional noise on every single
// request a fakeapi-backed client makes. This filters out just that one
// message by content, so any other Errorf/Warnf/Debugf resty ever logs
// still reaches the caller.
type quietLogger struct{}

func (quietLogger) Errorf(format string, v ...any) { log.Printf("ERROR RESTY "+format, v...) }

func (quietLogger) Warnf(format string, v ...any) {
	if strings.Contains(format, "sensitive credentials") {
		return
	}
	log.Printf("WARN RESTY "+format, v...)
}

func (quietLogger) Debugf(format string, v ...any) { log.Printf("DEBUG RESTY "+format, v...) }
