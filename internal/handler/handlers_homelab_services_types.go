// Tier 10 Phase 2 — Service Status (H2).
//
// Shared types for the homelab service-pin + probe surface. Lives
// in its own file so handlers_homelab_services.go (5 routes) and
// handlers_homelab_services_probe.go (3 routes) can both import
// the types without either growing past the 400-LOC cap.
//
// Storage: two tables in migrations/041_homelab.sql:
//
//	homelab_pinned_services — one row per (tenant, user, name) holding
//	                          the pin metadata (url, kind, icon, enabled)
//
//	homelab_service_health  — append-only time-series of probe results
//	                          (status, latency_ms, status_code, error)
//
// Per-user design: every query is filtered by both tenant_id and
// user_id. A user in tenant A cannot see (or probe) another user's
// pins even if they share a tenant — matches US-8 in the speckit
// proposal ("my homelab doesn't change under me when my co-founder
// rearranges theirs").
package handler

import "time"

// ------------------------------------------------------------------
// Pin-row shape — what /homelab/services returns + accepts on POST/PATCH.
// ------------------------------------------------------------------

// pinnedServiceRow is the JSON shape returned by GET /services, POST
// /services, and PATCH /services/:id. Mirrors the columns of
// homelab_pinned_services so the frontend can use field names
// directly without a transformer.
//
// The aggregate health snapshot is joined in from
// homelab_service_health by the GET handler (latest row per service
// in the past 10 minutes). When no recent probe exists, Health is
// nil and the UI renders a "unknown" pill.
type pinnedServiceRow struct {
	ID        string             `json:"id"`
	TenantID  string             `json:"tenant_id"`
	UserID    string             `json:"user_id"`
	Name      string             `json:"name"`
	URL       string             `json:"url"`
	Kind      string             `json:"kind"`
	Icon      *string            `json:"icon,omitempty"`
	Enabled   bool               `json:"enabled"`
	CreatedAt string             `json:"created_at"`
	UpdatedAt string             `json:"updated_at"`
	Health    *pinnedHealthBlock `json:"health,omitempty"`
}

// pinnedHealthBlock is the latest known health snapshot for a
// pinned service. status_code is nullable (only HTTP probes record
// it); latency_ms is nullable when the probe failed before
// establishing a connection. checked_at is the timestamp of the
// underlying homelab_service_health row (latest per service).
type pinnedHealthBlock struct {
	Status      string  `json:"status"`        // "up" | "degraded" | "down" | "unknown"
	LatencyMS   *int    `json:"latency_ms,omitempty"`
	StatusCode  *int    `json:"status_code,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
	CheckedAt   string  `json:"checked_at"`
}

// pinnedServiceReq is the body for POST /services. All fields are
// required at pin time — the handler validates them and returns
// 400 on any missing/invalid field. PATCH uses a separate type so
// fields can be optional.
type pinnedServiceReq struct {
	Name string  `json:"name"  binding:"required"`
	URL  string  `json:"url"   binding:"required"`
	Kind string  `json:"kind"  binding:"required"`
	Icon *string `json:"icon,omitempty"`
}

// pinnedServicePatchReq is the body for PATCH /services/:id. All
// fields are optional — the handler builds a dynamic UPDATE with
// only the fields the caller sent (same pattern as the dashboards
// PATCH handler).
type pinnedServicePatchReq struct {
	Name    *string `json:"name,omitempty"`
	URL     *string `json:"url,omitempty"`
	Kind    *string `json:"kind,omitempty"`
	Icon    *string `json:"icon,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

// ------------------------------------------------------------------
// Probe-result shapes — used by the probe handler + worker.
// ------------------------------------------------------------------

// serviceHealthRow is the JSON shape returned by GET /services/:id/
// history. Mirrors the columns of homelab_service_health. The
// frontend timeline widget renders one row per probe.
//
// All probes (background tick, one-shot, batch) INSERT into the
// table — the history endpoint is the universal read.
type serviceHealthRow struct {
	ID            string    `json:"id"`
	ServiceID     string    `json:"service_id"`
	Status        string    `json:"status"`
	LatencyMS     *int      `json:"latency_ms,omitempty"`
	StatusCode    *int      `json:"status_code,omitempty"`
	ErrorMessage  *string   `json:"error_message,omitempty"`
	CheckedAt     time.Time `json:"checked_at"`
}

// serviceProbeResult is the in-memory shape produced by the probe
// helpers (ProbeHTTP, ProbeTCP, ProbeICMP). The handler converts
// this into a homelab_service_health INSERT.
type serviceProbeResult struct {
	Status        string // "up" | "degraded" | "down" | "unknown"
	LatencyMS     *int
	StatusCode    *int
	ErrorMessage  string
}

// ------------------------------------------------------------------
// Validation enums — enforced on POST/PATCH to reject bad input
// before any DB call. Mirrors the no-CHECK-constraint design note
// in migrations/041_homelab.sql.
// ------------------------------------------------------------------

// allowedServiceKinds is the whitelist for the `kind` enum. Adding
// a new probe type (e.g. 'dns' for DNS lookup) means appending here
// AND implementing the matching probe helper in
// handlers_homelab_services_probe.go AND teaching the worker about
// it — all in one PR so the wire types and backend stay in sync.
var allowedServiceKinds = map[string]bool{
	"http":  true,
	"https": true,
	"tcp":   true,
	"icmp":  true,
}

// allowedServiceStatuses is the whitelist for the `status` enum.
// Mirrors the four UI pill colors (up=green, degraded=amber,
// down=red, unknown=gray).
var allowedServiceStatuses = map[string]bool{
	"up":       true,
	"degraded": true,
	"down":     true,
	"unknown":  true,
}

// maxPinnedServicesPerUser caps how many services a single user
// can pin. Matches the speckit proposal §"Risks" item 4 ("per-user
// pin count cap (50)") — prevents one user from pinning 10,000
// services and DoSing the 60s probe tick.
const maxPinnedServicesPerUser = 50

// historyRowLimit caps the GET /services/:id/history response. The
// frontend timeline widget only needs the most recent N rows; the
// index on (service_id, checked_at DESC) makes the LIMIT cheap.
const historyRowLimit = 100

// serviceHealthRetentionDays is the row lifetime enforced by the
// background worker on every tick. Older rows are pruned to keep
// the table bounded (~2M rows for a power user with 50 pins).
const serviceHealthRetentionDays = 30