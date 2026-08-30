// Tier 11 Phase 7 — Rate Limiting (PL7).
//
// ratelimit.go — in-memory token-bucket rate limiter.
//
// The Limiter holds one token bucket per tenant (keyed by
// tenant_id) plus a per-plan limit map. Every authenticated
// request flows through middleware.RateLimit which calls
// Limiter.Take(tenantID, plan) and either:
//   - lets the request through (writes the
//     X-RateLimit-Limit / X-RateLimit-Remaining /
//     X-RateLimit-Reset response headers), OR
//   - rejects with 429 + Retry-After (writes the
//     same X-RateLimit-* headers so the dashboard renders
//     a live "0 remaining" indicator).
//
// Why token bucket (not a fixed-window counter):
//   Token bucket allows BURSTS up to the bucket capacity, then
//   refills at the steady-state rate. A free user with
//   limit=10/min gets a full 10-token burst on the first
//   request and then refills at 1 token every 6s — way more
//   forgiving than a fixed-window counter that punishes
//   legitimate "open dashboard" bursts. Token bucket is also
//   what GitHub, Stripe, and Datadog all use for their
//   X-RateLimit-* surfaces.
//
// Why per-tenant buckets:
//   Per-IP buckets punish users behind a corporate NAT
//   (200 analysts share one egress IP — only one request
//   per second!). Per-tenant buckets key the limit to the
//   JWT subject so the limit applies per-customer, not
//   per-machine. (The signup endpoint is the one exception —
//   it rate-limits per-IP because at signup time there is no
//   tenant yet; that path lives in PL3 Phase 3 handlers.)
//
// Why in-memory (not Redis):
//   Single-node api-gateway today. The sync.RWMutex keeps
//   concurrent reads cheap (the hot path is Take, which
//   only takes a write lock briefly per request). A future
//   Redis adapter can implement the same Limiter interface
//   so the middleware + handler shape is unchanged when the
//   platform moves to multi-region. Phase 8 PL8 may add
//   Redis; PL7 ships the in-memory default.
//
// Why PlanDefaults = {free:10, starter:60, pro:100,
// enterprise:1000}:
//   Matches the seed_plans.go api_calls_per_minute column
//   (PL4) for the 4 built-in plans. Keeping the two in
//   sync means a tenant's PL7 bucket matches their PL4
//   plan card. A future "allow override per tenant" feature
//   would land on top of the per-plan defaults; today the
//   only knob is the per-plan map.

package platform

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Limiter is the in-memory token-bucket manager.
// One instance per api-gateway process; constructed in
// main.go and threaded through the router. The struct
// holds no DB refs — entirely in-memory — so a future
// Redis-backed implementation can swap in without
// touching callers.
type Limiter struct {
	mu      sync.RWMutex
	buckets map[uuid.UUID]*tokenBucket
	limits  map[string]int // per-plan requests-per-minute
}

// NewLimiter returns a fresh Limiter seeded with
// PlanDefaults. Callers can immediately invoke Take()
// without bootstrapping anything else.
func NewLimiter() *Limiter {
	return &Limiter{
		buckets: make(map[uuid.UUID]*tokenBucket),
		limits:  copyPlanDefaults(),
	}
}

// copyPlanDefaults snapshots the package-level map so a
// caller that mutates limits via SetGlobalLimits does
// not corrupt the seed data for future NewLimiter() calls.
func copyPlanDefaults() map[string]int {
	out := make(map[string]int, len(PlanDefaults))
	for k, v := range PlanDefaults {
		out[k] = v
	}
	return out
}

// getOrCreateBucket returns the bucket for tenantID,
// creating it FULL (tokens == capacity) on first use. The
// read-lock fast-path is for the common "bucket exists"
// case; create-on-miss takes the write lock.
//
// IMPORTANT: a fresh bucket MUST start full so a user's
// FIRST request is never blocked. Starting at zero would
// punish legitimate "just signed up, let me load the
// dashboard" bursts.
func (l *Limiter) getOrCreateBucket(tenantID uuid.UUID, limitPerMin int) *tokenBucket {
	l.mu.RLock()
	if b, ok := l.buckets[tenantID]; ok {
		l.mu.RUnlock()
		return b
	}
	l.mu.RUnlock()

	l.mu.Lock()
	defer l.mu.Unlock()
	if b, ok := l.buckets[tenantID]; ok {
		return b // double-checked: another goroutine created it
	}
	b := &tokenBucket{
		tokens:     float64(limitPerMin),
		capacity:   limitPerMin,
		lastRefill: time.Now(),
	}
	l.buckets[tenantID] = b
	return b
}

// refill advances the bucket's tokens by the elapsed-time
// * rate (limit / 60 per second), capped at capacity.
// Must be called with the bucket's write lock held (or
// with no other goroutine touching the bucket — we wrap
// every Take in the Limiter's write lock so a single
// goroutine sees a consistent view).
func (b *tokenBucket) refill(now time.Time, limitPerMin int) {
	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}
	refillPerSec := float64(limitPerMin) / 60.0
	b.tokens += elapsed * refillPerSec
	cap := float64(b.capacity)
	if b.tokens > cap {
		b.tokens = cap
	}
	b.lastRefill = now
}

// Take is the hot-path rate-limit check.
//
// Returns:
//
//	allowed            — false → caller MUST reject with 429
//	remaining          — tokens left after this request (>= 0)
//	resetAtSeconds     — when the bucket will be FULL again
//	blockedSeconds     — when an already-blocked tenant can retry
//
// Algorithm (token bucket):
//  1. If the bucket is in its blocked window, reject
//     WITHOUT touching the tokens (so retries don't
//     extend the block — same UX lesson as the login
//     rate-limit in handler/rate_limit.go).
//  2. Refill by elapsed * (limit / 60) tokens, capped at
//     capacity.
//  3. If tokens >= 1, consume 1, allow.
//  4. Else, set blockedTill = now + (1 / refillPerSec)
//     and reject.
//
// All 4 cases write back the bucket so a concurrent Take
// sees the new token count (the Limiter's mutex serialises
// the whole call so no further locking is needed).
func (l *Limiter) Take(tenantID uuid.UUID, plan string) (allowed bool, remaining int, resetAtSeconds int, blockedSeconds int) {
	limit := l.limitFor(plan)
	b := l.getOrCreateBucket(tenantID, limit)

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// Step 1: already-blocked window?
	if !b.blockedTill.IsZero() && now.Before(b.blockedTill) {
		retry := int(b.blockedTill.Sub(now).Seconds())
		if retry < 1 {
			retry = 1
		}
		// Tokens during a block window stay at 0 — DO NOT
		// consume them so the eventual unblock has a fresh
		// "wait until first token is available" experience.
		// resetAtSeconds tracks when the bucket will FULL again,
		// which is the same instant blockedTill lifts (we set
		// blockedTill = now + (1/refillPerSec) below so tokens
		// become available at exactly the unblock time).
		return false, 0, retry, retry
	}
	// Clear an expired block so the bucket can recover.
	b.blockedTill = time.Time{}

	// Step 2: refill.
	b.refill(now, limit)

	// Step 3: consume a token if available.
	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		rem := int(b.tokens)
		// resetAtSeconds = ceil(time until bucket is FULL).
		// At full, (capacity - tokens) tokens to add, rate =
		// limit/60 per second, so ceil(((cap-tokens) / rate) sec).
		ratePerSec := float64(limit) / 60.0
		var secsToFull int
		if ratePerSec > 0 && float64(b.capacity)-b.tokens > 0 {
			secsToFull = int((float64(b.capacity)-b.tokens)/ratePerSec + 0.999)
		} else {
			secsToFull = 60
		}
		if secsToFull < 1 {
			secsToFull = 1
		}
		return true, rem, secsToFull, 0
	}

	// Step 4: tokens < 1 → block until 1 token regenerates.
	ratePerSec := float64(limit) / 60.0
	secsToNext := 1
	if ratePerSec > 0 {
		secsToNext = int((1.0-b.tokens)/ratePerSec + 0.999)
	}
	if secsToNext < 1 {
		secsToNext = 1
	}
	b.blockedTill = now.Add(time.Duration(secsToNext) * time.Second)
	return false, 0, secsToNext, secsToNext
}

// limitFor returns the per-plan limit, falling back to
// free's limit when the plan is unknown (defensive — the
// caller is supposed to pass one of AllowedPlans). The
// mutex order matches Take: callers already hold l.mu so
// we don't lock again here.
func (l *Limiter) limitFor(plan string) int {
	if v, ok := l.limits[plan]; ok && v > 0 {
		return v
	}
	return l.limits[DefaultPlan]
}

// GetGlobalLimits returns a snapshot of the per-plan
// limits map (free/starter/pro/enterprise). The HTTP
// handler GET /platform/ratelimit/global uses this to
// surface the current caps to the dashboard.
func (l *Limiter) GetGlobalLimits() map[string]int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make(map[string]int, len(l.limits))
	for k, v := range l.limits {
		out[k] = v
	}
	return out
}

// SetGlobalLimits replaces the per-plan limits map. The
// HTTP handler PATCH /platform/ratelimit/global uses
// this. We do NOT clear buckets on a cap change — the
// existing tokens are kept, the new cap takes effect on
// the next refill. (If a free user is at 10 tokens and
// the operator drops free to 5/min, the next refill caps
// tokens at 5, not 10.)
//
// Returns an error if any value is out of range (1..100000)
// so a malformed PATCH body surfaces as 400 instead of
// silently zeroing out a plan's limit.
func (l *Limiter) SetGlobalLimits(limits map[string]int) error {
	for k, v := range limits {
		if v < 1 || v > 100000 {
			return ErrBadLimitRange
		}
		// Only accept allowed plan keys (defensive — a
		// curl caller sending {"hacker": 99999} shouldn't
		// be able to inject a new bucket class).
		if !isAllowedPlan(k) {
			return ErrUnknownPlan
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	// Merge: keep keys the operator didn't touch, replace
	// the ones they did. A PATCH with only {"free": 5}
	// should NOT wipe out starter/pro/enterprise.
	for _, p := range AllowedPlans {
		if v, ok := limits[p]; ok {
			l.limits[p] = v
		}
	}
	return nil
}

// ResetToDefaults restores PlanDefaults for all 4 plans.
// Wired to the dashboard "Reset to defaults" button (PATCH
// /platform/ratelimit/global with empty body — Phase 7
// handler special-cases that).
func (l *Limiter) ResetToDefaults() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.limits = copyPlanDefaults()
}

// isAllowedPlan reports whether name is one of the 4
// canonical plan names. Mirrors the handler-side
// allowedPlan check (defence-in-depth).
func isAllowedPlan(name string) bool {
	for _, p := range AllowedPlans {
		if p == name {
			return true
		}
	}
	return false
}
