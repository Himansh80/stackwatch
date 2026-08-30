// Tier 11 Phase 2 — Usage Metering (PL2).
//
// JSON row shapes + request bodies for the 5 metering endpoints.
//
// The handlers themselves live in handlers_platform_usage.go:
//
//	RecordUsageEvent        — POST /platform/usage-events
//	GetUsageCurrent         — GET  /platform/usage/current
//	GetUsageHistory         — GET  /platform/usage/history
//	GetUsageSummary         — GET  /platform/usage/summary (super_admin)
//	GetUsageExport          — GET  /platform/usage/export
//
// The hourly aggregation worker lives at
// internal/platform/usage_meter.go; it reads from
// platform_usage_events and UPSERTs into
// platform_usage_aggregates.
//
// Splitting types out keeps handlers_platform_usage.go under the
// 400-LOC cap (it would push ~360 lines as a single file with all
// types inlined — comfortable but the split keeps the handler file
// focused on HTTP-boundary logic: claim extraction, validation,
// error mapping, streaming CSV).
package handler

import "time"

// ---------------------------------------------------------------------------
// Allowed event kinds (server allowlist)
// ---------------------------------------------------------------------------
//
// Per spec PL2 §"PL2 — Usage Metering" the event_kind must be in a
// fixed allowlist so callers can't invent arbitrary billable
// dimensions. The list maps to the platforms we already have data
// for (homelab servers, alerts, dashboards) plus a few cross-cutting
// events (API calls, logins). Phase 5 (Backup/Restore) and Phase 7
// (Rate Limit) will extend this list as those surfaces ship.
//
// The map is value-less; we use it as a membership probe (looking
// up the kind returns a constant true/false). Keeping it as a map
// rather than a slice lets the membership check stay O(1) and lets
// future phases add kinds without reordering anything.

var allowedEventKinds = map[string]struct{}{
	// Homelab surfacing (Tier 10 already records these)
	"server.created": {},
	"server.deleted": {},
	"alert.fired":    {},
	"alert.resolved": {},
	// API surface
	"api.call": {},
	// Storage
	"storage.gb.hour": {},
	// Dashboard / observability
	"dashboard.panel.rendered": {},
	// Auth
	"login.success": {},
	"login.failure": {},
}

// allowedPeriods restricts the `period` query param on
// current/history to a known set so the SQL we generate downstream
// doesn't have to whitelist new values blindly.
var allowedPeriods = map[string]struct{}{
	"day":   {},
	"week":  {},
	"month": {},
}

// isAllowedEventKind is a tiny predicate the handlers use in two
// places (the POST body validation + the summary filter). Lifting
// it out keeps the handler file narrow.
func isAllowedEventKind(kind string) bool {
	_, ok := allowedEventKinds[kind]
	return ok
}

// isAllowedPeriod same idea for the period selector.
func isAllowedPeriod(p string) bool {
	_, ok := allowedPeriods[p]
	return ok
}

// defaultPeriod is what /usage/current uses when ?period is absent.
// Matches the spec proposal's example value.
const defaultPeriod = "month"

// defaultHistoryPeriods is how many bucket rows /usage/history
// returns when ?periods= is absent.
const defaultHistoryPeriods = 12

// ---------------------------------------------------------------------------
// Request body
// ---------------------------------------------------------------------------

// usageEventReq is the JSON body for POST /platform/usage-events.
// Every field except event_kind is optional; the validateEventReq
// helper in handlers_platform_usage.go enforces the allowlist on
// event_kind and bounds the metadata blob.
type usageEventReq struct {
	EventKind  string         `json:"event_kind" binding:"required"`
	Quantity   float64        `json:"quantity"   binding:"omitempty"`
	Unit       string         `json:"unit"       binding:"omitempty,max=32"`
	ResourceID string         `json:"resource_id" binding:"omitempty,max=256"`
	Metadata   map[string]any `json:"metadata"   binding:"omitempty"`
}

// ---------------------------------------------------------------------------
// Response rows
// ---------------------------------------------------------------------------

// usageEventRow is the JSON shape returned by:
//   - POST /platform/usage-events (the create response — exactly
//     the row that was just written)
//   - GET  /platform/usage/export (one CSV row per platform_usage_events
//     row streamed back to the caller)
type usageEventRow struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	UserID     *string        `json:"user_id,omitempty"`
	EventKind  string         `json:"event_kind"`
	Quantity   float64        `json:"quantity"`
	Unit       string         `json:"unit"`
	ResourceID *string        `json:"resource_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// usageAggregateRow is the JSON shape of one row in
// platform_usage_aggregates. Used inside usageCurrentResp +
// usageHistoryResp so the dashboard can pivot by event_kind.
type usageAggregateRow struct {
	EventKind   string    `json:"event_kind"`
	BucketTS    time.Time `json:"bucket_ts"`
	SumQuantity float64   `json:"sum_quantity"`
	Count       int       `json:"count"`
	MinQuantity *float64  `json:"min_quantity,omitempty"`
	MaxQuantity *float64  `json:"max_quantity,omitempty"`
}

// usageCurrentResp is the JSON returned by GET /platform/usage/current.
// One row per (event_kind) with the period total + a top-kind
// pointer so the KPI strip can label the "Top event" badge
// without re-querying.
type usageCurrentResp struct {
	Period        string             `json:"period"`
	RangeStart    time.Time          `json:"range_start"`
	RangeEnd      time.Time          `json:"range_end"`
	TotalCount    int                `json:"total_count"`
	TopEventKind  string             `json:"top_event_kind"`
	TopEventCount int                `json:"top_event_count"`
	EstimatedCost float64            `json:"estimated_cost"`
	Kinds         []usageKindSummary `json:"kinds"`
}

type usageKindSummary struct {
	EventKind   string  `json:"event_kind"`
	Count       int     `json:"count"`
	SumQuantity float64 `json:"sum_quantity"`
	CostUSD     float64 `json:"cost_usd"`
}

// usageHistoryResp is the JSON returned by GET
// /platform/usage/history. `series` is grouped by event_kind so the
// chart on the frontend can render one line per kind without
// re-grouping client-side.
type usageHistoryResp struct {
	Periods int                         `json:"periods"`
	Buckets []time.Time                 `json:"buckets"`
	Series  map[string][]usageBucketVal `json:"series"`
}

type usageBucketVal struct {
	BucketTS    time.Time `json:"bucket_ts"`
	SumQuantity float64   `json:"sum_quantity"`
	Count       int       `json:"count"`
}

// usageSummaryResp is the JSON returned by GET
// /platform/usage/summary (super_admin only). One row per tenant
// so the platform-admin billing table can show all customers in one
// round-trip.
type usageSummaryResp struct {
	Tenants []usageTenantSummary `json:"tenants"`
}

type usageTenantSummary struct {
	TenantID    string  `json:"tenant_id"`
	TenantName  string  `json:"tenant_name"`
	Plan        string  `json:"plan"`
	TotalCount  int     `json:"total_count"`
	SumQuantity float64 `json:"sum_quantity"`
	TopKind     string  `json:"top_kind"`
}

// usagePeriodBounds is a tiny helper kept here so it sits next to
// the response shapes that consume it. Returns the [start, end)
// window for the requested period. end is always "now" (caller can
// override for backfills). day = last 24h, week = last 7d, month
// = last 30d. Bucket count matches the history default.
func usagePeriodBounds(period string, now time.Time) (time.Time, time.Time, int) {
	switch period {
	case "day":
		return now.Add(-24 * time.Hour), now, 24
	case "week":
		return now.Add(-7 * 24 * time.Hour), now, 7
	default:
		// "month" and unknown both fall back to 30 daily
		// buckets — keeps the dashboard deterministic when a
		// caller mistypes ?period=months.
		return now.Add(-30 * 24 * time.Hour), now, 30
	}
}
