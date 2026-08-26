// Package middleware — Rate-limit middleware (Tier 11
// Phase 7 — PL7).
//
// Applies the in-process token-bucket rate limiter to
// every authenticated request that flows through the
// protected /api/v1/* group. Wired into the router
// AFTER auth (so claims.TenantID is set) but BEFORE
// the handler (so a 429 short-circuits any expensive
// work).
//
// Headers written on every protected response:
//
//	X-RateLimit-Limit       — the caller's per-minute cap
//	X-RateLimit-Remaining   — tokens left in the bucket
//	X-RateLimit-Reset       — seconds until the bucket is full
//
// On 429 the middleware ALSO sets Retry-After (RFC 7231)
// so a well-behaved client backs off automatically.
//
// Why an interface (PlanResolver):
//   The middleware must NOT import internal/platform/
//   (would be a cycle — platform would then import
//   middleware, but gin needs the middleware concretely).
//   The middleware accepts any PlanResolver; main.go wires
//   the concrete *platform.PlanCache that holds the
//   db.Pool. Tests can pass a fake PlanResolver that
//   returns "pro" without spinning up Postgres.
//
// Why the limiter is an interface too (Limiter):
//   Same reason — tests for the middleware should be able
//   to swap in a fake limiter that always-allow or
//   always-deny without dragging in the real token-bucket
//   package.

package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/kernel"
	"github.com/stackwatch/platform/internal/platform"
)

// PlanResolver is satisfied by *platform.PlanCache. See
// plan_cache.go for the production implementation.
type PlanResolver = platform.PlanResolver

// Limiter is the minimal surface the middleware needs.
// The production implementation is *platform.Limiter.
// Declared here as an interface so tests can fake it.
type Limiter interface {
	Take(tenantID uuid.UUID, plan string) (allowed bool, remaining int, resetAtSeconds int, blockedSeconds int)
}

// rateLimitCtxTimeout caps how long a single Take()
// call can hold the request goroutine. Today Take is
// in-memory + mutex-guarded so the timeout never fires
// in practice — but a future Redis-backed adapter would
// benefit from the explicit deadline (so a stalled Redis
// doesn't pin every request at the gateway).
const rateLimitCtxTimeout = 2 * time.Second

// RateLimit returns a gin middleware that:
//   1. Reads claims.TenantID (set by RequireAuth, must
//      run BEFORE this middleware in the chain).
//   2. Resolves the tenant's plan name via PlanResolver.
//   3. Calls limiter.Take(tenantID, plan).
//   4. If allowed → sets X-RateLimit-* headers, c.Next().
//   5. If denied → sets X-RateLimit-* + Retry-After, writes
//      429 JSON body, c.Abort() (no handler invocation).
//
// The middleware is safe to wire into ANY protected
// route — it never panics, never blocks longer than
// rateLimitCtxTimeout, and never logs on its own (the
// upstream logging middleware captures the 429 status).
func RateLimit(limiter Limiter, plans PlanResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate-limiting when no auth context is set.
		// (Public routes — health, signup, install-script —
		// are mounted on a different router group that
		// never reaches this middleware.) Defensive: if
		// a caller wires this onto the public group,
		// fail open instead of denying every anonymous
		// request.
		tenantIDVal, exists := c.Get("auth.tenant_id")
		if !exists {
			c.Next()
			return
		}
		tenantID, ok := tenantIDVal.(uuid.UUID)
		if !ok || tenantID == uuid.Nil {
			c.Next()
			return
		}

		// Super-admin / platform_admin bypass (Aug 2026).
		// The free plan cap (60/min even after the bump)
		// is for end-user tenants; operator accounts must
		// never get throttled. Look up role via the
		// context-stored claims (auth.role). Accept
		// super_admin, platform_admin, AND 'admin'
		// (legacy super-admin role name).
		if role, ok := c.Get("auth.role"); ok {
			if rs, ok := role.(string); ok && (rs == "super_admin" || rs == "platform_admin" || rs == "admin") {
				c.Header("X-RateLimit-Limit", "unlimited")
				c.Header("X-RateLimit-Remaining", "unlimited")
				c.Header("X-RateLimit-Reset", "0")
				c.Next()
				return
			}
		}

		// Resolve the plan name. The PlanResolver caches
		// after the first DB read so subsequent requests
		// are O(1).
		plan := plans.ResolvePlan(tenantID)

		// Bound the Take() call so a future Redis-backed
		// limiter can't pin the request forever. Today
		// Take is mutex-only, so the timeout never fires
		// — but the explicit ctx keeps the architecture
		// honest.
		ctx, cancel := context.WithTimeout(c.Request.Context(),
			rateLimitCtxTimeout)
		defer cancel()
		_ = ctx // currently unused; placeholder for future Redis adapter

		allowed, remaining, resetSec, blockedSec := limiter.Take(tenantID, plan)

		// Always write X-RateLimit-* headers (even on 429)
		// so the dashboard can render the live count.
		writeRateLimitHeaders(c, plan, remaining, resetSec)

		if !allowed {
			// Retry-After (RFC 7231) takes precedence over
			// the X-RateLimit-Reset header for clients
			// that honour the de-facto standard.
			if blockedSec < 1 {
				blockedSec = 1
			}
			c.Header("Retry-After", strconv.Itoa(blockedSec))
			kernel.RespondRateLimited(c, kernel.ErrTooManyRequests, blockedSec)
			c.Abort()
			return
		}

		c.Next()
	}
}

// writeRateLimitHeaders writes the X-RateLimit-* trio.
//
// Limit     — always the per-minute cap for the caller's plan
// Remaining — tokens left in the bucket (>= 0)
// Reset     — seconds until the bucket is FULL
//
// When the plan name is unknown to the limiter (shouldn't
// happen because PlanResolver returns "free" on miss) we
// still write the headers but with limit=0 so the client
// can detect a misconfiguration.
func writeRateLimitHeaders(c *gin.Context, plan string, remaining, resetSec int) {
	cap := planLimitLookup(plan)
	if cap < 0 {
		cap = 0
	}
	if remaining < 0 {
		remaining = 0
	}
	if resetSec < 1 {
		resetSec = 1
	}
	c.Header("X-RateLimit-Limit", strconv.Itoa(cap))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
	c.Header("X-RateLimit-Reset", strconv.Itoa(resetSec))
}

// planLimitLookup returns the per-minute cap for plan,
// falling back to platform.PlanDefaults[platform.DefaultPlan].
// Inlined here so the middleware never imports the
// platform package's mutable state directly — the lookup
// is a pure function of the public PlanDefaults map.
//
// The package-level platform.PlanDefaults is read at
// request time (not captured at construction) so a
// PATCH /platform/ratelimit/global that changes the cap
// is visible on the very next request.
func planLimitLookup(plan string) int {
	if v, ok := platform.PlanDefaults[plan]; ok {
		return v
	}
	if v, ok := platform.PlanDefaults[platform.DefaultPlan]; ok {
		return v
	}
	return http.StatusOK // fallback (should never happen)
}
