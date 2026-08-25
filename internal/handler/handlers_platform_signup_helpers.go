// Tier 11 Phase 3 — Self-Service Signup (PL3) — helpers.
//
// Two pieces live here, both small enough to keep the main
// handlers file (handlers_platform_signup.go) under the
// 400-LOC cap:
//
//  1. In-memory per-IP rate limit (signupRateLimit) — used by
//     CreateSignup to enforce the 10/day cap from spec §"PL3 —
//     Self-Service Signup". Replaced by the Redis-backed
//     token-bucket in PL7 (Phase 7).
//
//  2. Random verification-token generator (newVerificationToken).
//     Used by CreateSignup (initial token) + ResendSignup
//     (rotation). crypto/rand only — math/rand would be a
//     CVE magnet for an endpoint whose entire job is to mint
//     credentials.
package handler

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// =====================================================================
// Per-IP rate limit — Phase 3 (replaced by PL7 in Phase 7)
// =====================================================================
//
// signupRateLimit holds one rolling-window counter per source IP.
// The map is package-level (singleton) so every CreateSignup call
// mutates the SAME state — process-local, but that's fine for a
// single api-gateway instance today.
//
// Why not middleware: the rate limit only applies to one of the
// three PL3 routes (CreateSignup); the other two (verify, resend)
// don't need a per-IP gate because they don't create new
// platform_signups rows. A targeted handler-side check keeps the
// middleware surface clean.
//
// Why not a database-backed counter: a per-IP counter belongs in
// Redis (PL7) once multi-replica lands. Phase 7 ships the proper
// implementation; today this map is sufficient for the single-
// instance deploy on .115.
//
// Concurrency: all mutations are guarded by signupRateMu. The
// hot path is one map lookup + one map update per request.
const (
	signupRateLimitPerDay = 10
	signupRateWindow      = 24 * time.Hour
)

type signupBucket struct {
	firstAt time.Time
	count   int
}

var (
	signupRateMu sync.Mutex
	signupRate   = map[string]*signupBucket{}
)

// signupAllowed reports whether `ip` may attempt another signup.
// It returns false when the IP has hit the 10/day limit; the
// caller should 429 with the seconds-until-reset as Retry-After.
//
// The bucket expires when the window rolls over — we reset
// count to 0 and firstAt to now() so a sustained attacker
// gets exactly 10 attempts per 24h no matter how they spread
// them, while a legitimate operator who occasionally signs up
// (Phase 4 might allow tenant expansion) never hits the cap.
func signupAllowed(ip string) (allowed bool, retryAfterSeconds int) {
	if ip == "" {
		// Unknown IP — let it through; the upstream proxy
		// should always set ClientIP. Worst case we lose
		// one limit, not enforce false-positives.
		return true, 0
	}
	signupRateMu.Lock()
	defer signupRateMu.Unlock()

	now := time.Now().UTC()
	b, ok := signupRate[ip]
	if !ok || now.Sub(b.firstAt) >= signupRateWindow {
		signupRate[ip] = &signupBucket{firstAt: now, count: 1}
		return true, 0
	}
	if b.count >= signupRateLimitPerDay {
		// Retry-After: seconds remaining in the rolling window.
		// Cast to int (truncates fractions — RFC 7231 allows it).
		remaining := signupRateWindow - now.Sub(b.firstAt)
		return false, int(remaining.Seconds())
	}
	b.count++
	return true, 0
}

// =====================================================================
// Random verification-token generator
// =====================================================================

// newVerificationToken returns a 64-char hex string (32 bytes
// of random data). crypto/rand is the only acceptable source —
// math/rand would be a CVE magnet for a security-claim endpoint.
func newVerificationToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand only fails if the OS RNG is broken; we
		// can't recover, so return a recognisably-broken
		// value (all zeros) — the handler will fail to
		// /verify against it and the user will retry. Better
		// than panicking in production.
		for i := range b {
			b[i] = 0
		}
	}
	return hex.EncodeToString(b)
}
