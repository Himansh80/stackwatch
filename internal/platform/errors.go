// Package platform — Sentinel errors shared across the Tier 11
// (Platform & Commerce) code. Putting them in their own file
// keeps internal/platform/deploy.go under the 400-LOC cap while
// still letting the HTTP handler layer (handlers_platform_deploy.go)
// map each error to a specific HTTP status code.
//
// Errors here are returned by the helper functions in
// internal/platform/deploy.go (LookupInstallToken, etc.). The
// handler layer uses errors.Is to decide on the response status:
//   ErrTokenNotFound → 410 Gone  (already used or never existed)
//   ErrTokenExpired  → 410 Gone  (lifetime exceeded)
//   anything else    → 500 Internal Server Error (with logging)
package platform

import "errors"

// ErrTokenNotFound is returned by LookupInstallToken when no
// un-used, un-expired row matches the supplied plaintext. The
// handler maps this to HTTP 410 Gone ("the install token has
// already been used or never existed") rather than 404 because
// the URL is well-formed — the resource existed at some point
// but is no longer available.
var ErrTokenNotFound = errors.New("install token not found")

// ErrTokenExpired is returned when a token's expires_at is in
// the past. Today LookupInstallToken filters out expired rows
// in its WHERE clause so this error is reserved for a future
// code path (e.g. an admin "show me all expired tokens" view).
var ErrTokenExpired = errors.New("install token expired")