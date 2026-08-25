// Tier 11 Phase 7 — Rate Limiting (PL7).
//
// ratelimit_snapshot.go — read-only snapshot helpers for
// the rate limiter.
//
// Why split from ratelimit.go:
//   The hot-path code (Take + refill + block-window) lives
//   in ratelimit.go so it stays focused on the per-request
//   path. The snapshot helpers here are dashboard-facing
//   reads that never consume a token; extracting them keeps
//   ratelimit.go under the 400-LOC cap while preserving a
//   single "Limiter" surface for callers.
//
//   Both helpers take the read lock first (or write lock
//   when they need to refill) so they compose safely with
//   concurrent Take() calls. They DO NOT mutate buckets
//   beyond advancing the clock via refill() — a no-op for
//   "is this bucket full" but necessary to compute
//   "tokens remaining now".

package platform

import (
	"sort"
	"time"

	"github.com/google/uuid"
)

// GetTenantUsage returns the live usage row for a tenant.
// Used by GET /platform/ratelimit/me. Does NOT consume a
// token — it's a read-only peek. Returns zero usage when
// the tenant has no bucket yet (no requests this process
// lifetime → no bucket → report a full-bucket "0 of N used").
func (l *Limiter) GetTenantUsage(tenantID uuid.UUID, plan string) tenantUsage {
	limit := l.limitFor(plan)
	l.mu.RLock()
	b, ok := l.buckets[tenantID]
	l.mu.RUnlock()

	now := time.Now()
	out := tenantUsage{
		TenantID:       tenantID,
		Plan:           plan,
		LimitPerMinute: limit,
	}

	if !ok {
		// No bucket yet → report a fresh "0 of N used".
		out.Remaining = limit
		out.ResetAtSeconds = 60
		return out
	}

	// Refill under the write lock so the read is
	// consistent with what Take would see.
	l.mu.Lock()
	b.refill(now, limit)
	used := int(float64(b.capacity) - b.tokens)
	if used < 0 {
		used = 0
	}
	if used > b.capacity {
		used = b.capacity
	}
	rem := int(b.tokens)
	if rem < 0 {
		rem = 0
	}
	blockedSec := 0
	if !b.blockedTill.IsZero() && now.Before(b.blockedTill) {
		blockedSec = int(b.blockedTill.Sub(now).Seconds())
		if blockedSec < 1 {
			blockedSec = 1
		}
	}
	// resetAtSeconds: how long until the bucket is FULL.
	ratePerSec := float64(limit) / 60.0
	secsToFull := 60
	if ratePerSec > 0 && float64(b.capacity)-b.tokens > 0 {
		secsToFull = int((float64(b.capacity)-b.tokens)/ratePerSec + 0.999)
	}
	if secsToFull < 1 {
		secsToFull = 1
	}
	l.mu.Unlock()

	out.CurrentWindowCount = used
	out.Remaining = rem
	out.ResetAtSeconds = secsToFull
	out.BlockedUntilSeconds = blockedSec
	return out
}

// GetBlockedTenants returns every tenant currently in its
// blocked window. Used by GET /platform/ratelimit/blocked
// (super_admin only). Sorted by blocked-until DESC so the
// dashboard shows the longest-blocked tenants first.
//
// Read-only: does not mutate buckets.
func (l *Limiter) GetBlockedTenants() []blockedTenant {
	l.mu.RLock()
	defer l.mu.RUnlock()
	now := time.Now()
	var out []blockedTenant
	for tid, b := range l.buckets {
		if b.blockedTill.IsZero() || !now.Before(b.blockedTill) {
			continue
		}
		secs := int(b.blockedTill.Sub(now).Seconds())
		if secs < 1 {
			secs = 1
		}
		// Identify the plan by looking at the bucket's
		// capacity (which is set from the plan at
		// getOrCreateBucket time). Walk the limits map
		// for a match.
		plan := planForCapacity(b.capacity, l.limits)
		used := int(float64(b.capacity) - b.tokens)
		if used < 0 {
			used = 0
		}
		out = append(out, blockedTenant{
			TenantID:            tid,
			Plan:                plan,
			CurrentCount:        used,
			LimitPerMinute:      b.capacity,
			BlockedUntilSeconds: secs,
		})
	}
	// Deterministic order: longest-blocked first.
	sort.Slice(out, func(i, j int) bool {
		return out[i].BlockedUntilSeconds > out[j].BlockedUntilSeconds
	})
	return out
}

// planForCapacity finds the plan name whose limit equals
// capacity. Falls back to "free" when no exact match
// (defensive — a SetGlobalLimits call could create an
// orphan capacity that doesn't match any plan).
func planForCapacity(capacity int, limits map[string]int) string {
	for plan, lim := range limits {
		if lim == capacity {
			return plan
		}
	}
	return DefaultPlan
}
