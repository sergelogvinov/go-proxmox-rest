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
	"crypto/tls"
	"errors"
	"net"

	"resty.dev/v3"
)

// safePOSTRetryCondition is the sole resty.RetryConditionFunc installed
// (via Request.SetRetryConditions, replacing rather than adding to the
// client's own default) on a POST to a node-scoped path when
// WithSafePOSTFailover is enabled. It allows a retry — and, via
// maxDirectAttempts, a fallback to the delegate balancer — only when err
// proves the request never reached the server. Client.retryCondition (5xx,
// SlowDown, any transport error) is not safe to reuse here: a non-idempotent
// method must not be retried just because *some* error occurred.
func safePOSTRetryCondition(_ *resty.Response, err error) bool {
	return errIsUnsent(err)
}

// errIsUnsent reports whether err proves a request never reached the
// server:
//   - the TCP connection could not be established at all — refused, no
//     route to host, DNS failure — which net/http always surfaces as a
//     *net.OpError with Op "dial", regardless of which of those it was;
//   - the TLS handshake failed — a *tls.CertificateVerificationError,
//     which Go's transport returns for every handshake-time certificate
//     failure (wrapping the more specific x509 error underneath).
//
// Both leave the server having received nothing, so retrying a POST is
// safe. Anything else — a write that got partway, a timeout waiting for a
// response, a 5xx — might mean the request already reached the server and
// must not be retried.
func errIsUnsent(err error) bool {
	if err == nil {
		return false
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return true
	}

	var certErr *tls.CertificateVerificationError

	return errors.As(err, &certErr)
}
