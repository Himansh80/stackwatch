// Tier 11 Phase 7 — Rate Limiting (PL7).
//
// plan_cache.go — per-tenant plan name cache used by the
// rate-limit middleware.
//
// Why a cache:
//   The middleware hot path runs on EVERY authenticated
//   request. Reading tenants.plan from Postgres every time
//   would double-or-triple the per-request DB load for
//   zero benefit (plans change rarely). A sync.Map keyed
//   by tenant_id gives O(1) lookup with a single SELECT
//   per tenant per process lifetime.
//
// Why an interface (not a concrete *db.Pool):
//   The middleware lives in internal/middleware/ and must
//   not depend on internal/platform/ (would be a cycle).
//   An interface lets main.go wire the concrete *platform.PlanCache
//   (which holds the *db.Pool) into the middleware at
//   build time without creating an import cycle.
//
// How staleness is handled:
//   Today: never expire. A tenant's plan change is
//   infrequent enough that the api-gateway can be
//   restarted when a plan change happens (the operator
//   runs `systemctl restart stackwatch-api-gateway` after
//   the migration). Phase 8 PL8 will add a TTL so the
//   cache self-invalidates; for now, "restart on plan
//   change" is documented in the operator runbook.
//
// Why a sync.Map (not a map + RWMutex):
//   sync.Map is optimised for the "write once, read many"
//   access pattern (a tenant's plan is looked up thousands
//   of times after one DB read). The standard
//   map+RWMutex would acquire the read lock on every
//   request — sync.Map's atomic Load avoids that.

package platform

import (
	"sync"

	"github.com/google/uuid"
)

// PlanResolver is the interface the middleware depends on.
// Anything that can map a tenantID → plan name works.
// Internal package implements it via the *PlanCache type
// below; tests can swap in a fake.
type PlanResolver interface {
	ResolvePlan(tenantID uuid.UUID) string
}

// PlanCache is the in-process tenants.plan cache. One
// instance per api-gateway process; constructed in main.go
// alongside the Limiter and threaded into the middleware.
type PlanCache struct {
	mu    sync.RWMutex
	cache map[uuid.UUID]string
	// Lookup is the actual DB query. Kept as a field
	// (not a method) so tests can swap it for a stub.
	Lookup func(tenantID uuid.UUID) string
}

// NewPlanCache returns a fresh PlanCache with a DB-backed
// default Lookup. The default Lookup falls back to
// "free" when the tenant row is missing (defensive —
// a tenant created before PL7 launched won't have a
// plan row until Phase 8 retroactively fills it).
func NewPlanCache(lookup func(tenantID uuid.UUID) string) *PlanCache {
	return &PlanCache{
		cache:  make(map[uuid.UUID]string),
		Lookup: lookup,
	}
}

// ResolvePlan returns the cached plan name for tenantID,
// doing at most one DB read on cache miss. On error the
// caller gets "free" — the safe default (a tenant with an
// unreadable plan gets the most restrictive bucket).
//
// Implements PlanResolver.
func (c *PlanCache) ResolvePlan(tenantID uuid.UUID) string {
	if tenantID == uuid.Nil {
		return DefaultPlan
	}
	// Fast path: cache hit.
	c.mu.RLock()
	if p, ok := c.cache[tenantID]; ok {
		c.mu.RUnlock()
		return p
	}
	c.mu.RUnlock()

	// Slow path: cache miss → DB read once, then cache
	// forever (until process restart). Holding the write
	// lock while reading prevents two concurrent misses
	// from both firing the DB query.
	c.mu.Lock()
	// Double-check after acquiring the write lock — another
	// goroutine might have populated the entry between our
	// RUnlock and Lock.
	if p, ok := c.cache[tenantID]; ok {
		c.mu.Unlock()
		return p
	}
	plan := DefaultPlan
	if c.Lookup != nil {
		if got := c.Lookup(tenantID); got != "" {
			plan = got
		}
	}
	c.cache[tenantID] = plan
	c.mu.Unlock()
	return plan
}

// Invalidate drops a single tenant's cache entry. Wired
// to PATCH /platform/limits/me so a tenant that just
// changed plan picks up the new bucket on the very next
// request without a process restart.
func (c *PlanCache) Invalidate(tenantID uuid.UUID) {
	c.mu.Lock()
	delete(c.cache, tenantID)
	c.mu.Unlock()
}

// Size returns the number of cached entries. Used by
// the diagnostics endpoint (PL8) and by tests.
func (c *PlanCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}
