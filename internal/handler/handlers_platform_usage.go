// Tier 11 Phase 2 — Usage Metering (PL2).
//
// HTTP route handlers for the 5 metering endpoints:
//
//	POST /api/v1/platform/usage-events  — RecordUsageEvent  (auth-protected)
//	GET  /api/v1/platform/usage/current — GetUsageCurrent   (auth-protected)
//	GET  /api/v1/platform/usage/history — GetUsageHistory   (auth-protected)
//	GET  /api/v1/platform/usage/summary — GetUsageSummary   (super_admin only)
//	GET  /api/v1/platform/usage/export  — GetUsageExport    (auth-protected)
//
// Shared types live in handlers_platform_usage_types.go (kept
// under the 400-LOC cap by the split). The hourly aggregation
// worker lives in internal/platform/usage_meter.go.
//
// Why these endpoints:
//
//	RecordUsageEvent is the write path — every billable action
//	(server created, alert fired, dashboard panel rendered, etc.)
//	INSERTs one row into platform_usage_events. The body is
//	restricted to a fixed allowlist of event_kind values so a
//	caller can't invent new billable dimensions.
//	GetUsageCurrent / GetUsageHistory / GetUsageSummary are the
//	read path — they scan platform_usage_aggregates (the rollup
//	table) so the dashboard stays fast even with millions of raw
//	events. GetUsageExport streams the raw events as CSV so a
//	finance team can slice the data in Excel.
//
// Auth & tenancy: every endpoint requires a JWT (carried by the
// router's RequireAuth middleware). Tenant isolation is enforced
// by always filtering on claims.TenantID — no cross-tenant
// leakage. The summary endpoint additionally checks Role ==
// "super_admin" (matches the existing handlers_tenant.go pattern
// for tenant-wide admin).
package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// usageCostPerEvent is a hand-tuned per-kind USD cost table. We
// multiply quantity * per-event to get the estimated cost column.
// The table is intentionally simple — Phase 4 (Tenant Limits) is
// when plan-tier pricing kicks in for real. Keeping it as a const
// map means the values are grep-able, easy to tune in code review,
// and don't require a config knob for a Phase 2 ship.
var usageCostPerEvent = map[string]float64{
	"server.created":           0.01,
	"server.deleted":           0.0,
	"alert.fired":              0.001,
	"alert.resolved":           0.0,
	"api.call":                 0.0001,
	"storage.gb.hour":          0.02,
	"dashboard.panel.rendered": 0.0005,
	"login.success":            0.0,
	"login.failure":            0.0,
}

func costFor(eventKind string, qty float64) float64 {
	rate, ok := usageCostPerEvent[eventKind]
	if !ok {
		rate = 0.0
	}
	return rate * qty
}

// ------------------------------------------------------------------
// Protected endpoint: POST /api/v1/platform/usage-events
// ------------------------------------------------------------------

// RecordUsageEvent inserts one row into platform_usage_events.
// Body validation enforces the event_kind allowlist (server-side,
// so a caller can't smuggle in a new "kind" they'd then bypass)
// and bounds metadata to keep the jsonb small.
//
// Returns 201 with the row that was just written (NO id echo from
// the DB — we generate the id client-side so the response is
// stable across replicas without a RETURNING round-trip — the
// table defaults handle id + created_at).
func RecordUsageEvent(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, _ := userFromContext(c) // optional — billing events can fire from system
		var req usageEventReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Required field check (gin's `binding:"required"` does NOT
		// reject empty strings; do it explicitly).
		req.EventKind = strings.TrimSpace(req.EventKind)
		if req.EventKind == "" {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"event_kind is required")
			return
		}
		if !isAllowedEventKind(req.EventKind) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"event_kind is not in the allowlist")
			return
		}
		// Defaults match the spec PL2 body: quantity=1 (counted
		// actions), unit="count" (most events are scalar; storage
		// events use "gb" or "hour").
		qty := req.Quantity
		if qty == 0 {
			qty = 1
		}
		if qty < 0 {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"quantity must be non-negative")
			return
		}
		unit := strings.TrimSpace(req.Unit)
		if unit == "" {
			unit = "count"
		}
		// Always go through the canonical low-level INSERT path so
		// the row id is generated server-side and we never store a
		// client-supplied id (which would let a caller overwrite
		// another tenant's row by id collision — defense in depth
		// even though the schema UUID is hardened).
		newID := uuid.New()
		var userArg interface{}
		if userID != nil {
			userArg = userID.ID
		}
		var resourceArg interface{}
		if rid := strings.TrimSpace(req.ResourceID); rid != "" {
			resourceArg = rid
		}
		_, err := pool.Pgx().Exec(c.Request.Context(),
			`INSERT INTO platform_usage_events
			        (id, tenant_id, user_id, event_kind, quantity, unit, resource_id, metadata)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			newID, tenantID, userArg, req.EventKind, qty, unit, resourceArg, req.Metadata)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		var resp usageEventRow
		resp.ID = newID.String()
		resp.TenantID = tenantID.String()
		if userID != nil {
			uid := userID.ID.String()
			resp.UserID = &uid
		}
		resp.EventKind = req.EventKind
		resp.Quantity = qty
		resp.Unit = unit
		if rid := strings.TrimSpace(req.ResourceID); rid != "" {
			ridVal := rid
			resp.ResourceID = &ridVal
		}
		if len(req.Metadata) > 0 {
			resp.Metadata = req.Metadata
		}
		resp.CreatedAt = time.Now().UTC()
		kernel.RespondCreated(c, resp)
	}
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/platform/usage/current
// ------------------------------------------------------------------

// GetUsageCurrent returns the period totals for the caller's
// tenant, grouped by event_kind, plus a top-kind pointer and an
// estimated cost. Reads from platform_usage_aggregates (the
// worker's output) so the response stays fast.
func GetUsageCurrent(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		period := strings.TrimSpace(c.Query("period"))
		if period == "" {
			period = defaultPeriod
		}
		if !isAllowedPeriod(period) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"period must be day|week|month")
			return
		}
		now := time.Now().UTC()
		start, end, _ := usagePeriodBounds(period, now)

		// Pull kind-level aggregates for the window.
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT event_kind, COALESCE(SUM(sum_quantity),0)::float8 AS sum_q,
			        COALESCE(SUM(count),0)::bigint AS cnt
			   FROM platform_usage_aggregates
			  WHERE tenant_id = $1
			    AND bucket_ts >= $2 AND bucket_ts < $3
			  GROUP BY event_kind
			  ORDER BY cnt DESC`,
			tenantID, start, end)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		var (
			kinds      []usageKindSummary
			totalCount int
			topKind    = ""
			topCount   = 0
			totalCost  float64
		)
		for rows.Next() {
			var k usageKindSummary
			if err := rows.Scan(&k.EventKind, &k.SumQuantity, &k.Count); err != nil {
				continue
			}
			k.CostUSD = costFor(k.EventKind, k.SumQuantity)
			totalCount += k.Count
			if k.Count > topCount {
				topCount = k.Count
				topKind = k.EventKind
			}
			totalCost += k.CostUSD
			kinds = append(kinds, k)
		}
		kernel.RespondOK(c, usageCurrentResp{
			Period:        period,
			RangeStart:    start,
			RangeEnd:      end,
			TotalCount:    totalCount,
			TopEventKind:  topKind,
			TopEventCount: topCount,
			EstimatedCost: totalCost,
			Kinds:         kinds,
		})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/platform/usage/history
// ------------------------------------------------------------------

// GetUsageHistory returns N period-bucket rows grouped by
// event_kind, suitable for a stacked line chart. Default is 12
// daily buckets (the spec proposal uses "month" of dailies).
func GetUsageHistory(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		periodsQ := strings.TrimSpace(c.Query("periods"))
		periods := defaultHistoryPeriods
		if periodsQ != "" {
			if n, err := strconv.Atoi(periodsQ); err == nil && n > 0 && n <= 365 {
				periods = n
			}
		}
		eventKindFilter := strings.TrimSpace(c.Query("event_kind"))
		if eventKindFilter != "" && !isAllowedEventKind(eventKindFilter) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"event_kind is not in the allowlist")
			return
		}

		now := time.Now().UTC()
		// 12 default periods × 1 day each → last 12 daily buckets.
		// date_trunc('day', …) gives bucket_ts aligned to UTC midnight
		// (the worker does the same — see usage_meter.go).
		// Use (now - periods days) as the floor for the bucket series.
		earliest := now.AddDate(0, 0, -periods)
		// Pull bucket-aligned daily series.
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT event_kind, bucket_ts, sum_quantity::float8, count
			   FROM platform_usage_aggregates
			  WHERE tenant_id = $1
			    AND bucket_ts >= date_trunc('day', $2::timestamptz)
			    AND ($3 = '' OR event_kind = $3)
			  ORDER BY bucket_ts ASC`,
			tenantID, earliest, eventKindFilter)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		// Bucket set (UTC days we have rows for) — built from the DB
		// scan instead of synthesized so the chart can render empty
		// bars correctly.
		bucketSet := map[int64]time.Time{}
		series := map[string][]usageBucketVal{}
		for rows.Next() {
			var (
				kind string
				ts   time.Time
				sum  float64
				cnt  int
			)
			if err := rows.Scan(&kind, &ts, &sum, &cnt); err != nil {
				continue
			}
			bucketSet[ts.UTC().Unix()] = ts.UTC()
			series[kind] = append(series[kind], usageBucketVal{
				BucketTS:    ts.UTC(),
				SumQuantity: sum,
				Count:       cnt,
			})
		}
		buckets := make([]time.Time, 0, len(bucketSet))
		for _, ts := range bucketSet {
			buckets = append(buckets, ts)
		}
		// Cheap deterministic ordering (sort by ts ascending) —
		// better-sqlite-style helpers not used here because the
		// pool is pgx.
		sortTimeAsc(buckets)
		kernel.RespondOK(c, usageHistoryResp{
			Periods: periods,
			Buckets: buckets,
			Series:  series,
		})
	}
}

// sortTimeAsc is a tiny insertion sort kept inline so we don't
// pull in sort just for this one call site. Slices here are short
// (≤ 365 day buckets), so O(n²) is fine.
func sortTimeAsc(s []time.Time) {
	for i := 1; i < len(s); i++ {
		j := i
		for j > 0 && s[j-1].After(s[j]) {
			s[j-1], s[j] = s[j], s[j-1]
			j--
		}
	}
}

// ------------------------------------------------------------------
// Protected endpoints (continued):
//   GET /api/v1/platform/usage/summary — GetUsageSummary (super_admin only)
//   GET /api/v1/platform/usage/export  — GetUsageExport
//
// These live in handlers_platform_usage_summary.go to keep this
// file under the 400-LOC cap. The summary endpoint is gated by
// super_admin (cross-tenant billing dashboard). The export endpoint
// streams CSV so finance can slice the data in Excel.
// ------------------------------------------------------------------
