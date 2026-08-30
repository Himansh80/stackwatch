// Tier 7 Phase 2 — Synthetics Full (D5) — Test CRUD (create/list/get).
//
//	GET  /api/v1/synthetics/tests-full          — list tests (filter ?enabled, ?type)
//	POST /api/v1/synthetics/tests-full          — create test
//	GET  /api/v1/synthetics/tests-full/:test_id — test detail
//
// Update + delete live in handlers_synthetics_test_update.go.
// Run-now + SLA + results live in handlers_synthetics_run.go.
//
// Results are persisted in synthetics_test_runs (NOT synthetics_results —
// that name is owned by the Tier 6 v2 schema and holds 13k+ legacy rows).
package handler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// allowedSyntheticTypes — defense in depth at the API edge. The spec
// defines these five values; anything else falls back to "http" so the
// column never holds garbage.
var allowedSyntheticTypes = map[string]bool{
	"http": true, "tcp": true, "icmp": true, "browser": true, "multi_step": true,
}

// synthTestReq is the JSON shape for POST /tests-full.
// All fields optional except name + type + url.
type synthTestReq struct {
	Name            string          `json:"name" binding:"required,min=1,max=128"`
	Type            string          `json:"type" binding:"required,oneof=http tcp icmp browser multi_step"`
	URL             string          `json:"url" binding:"required,min=1,max=2048"`
	Method          string          `json:"method" binding:"max=16"`
	Body            string          `json:"body"`
	Headers         json.RawMessage `json:"headers"`
	Assertions      json.RawMessage `json:"assertions"`
	IntervalSeconds int             `json:"interval_seconds" binding:"min=10,max=86400"`
	TimeoutMs       int             `json:"timeout_ms" binding:"min=100,max=120000"`
	Locations       []string        `json:"locations"`
	SLAUptimePct    float64         `json:"sla_uptime_pct"`
	SLAResponseMs   int             `json:"sla_response_ms"`
	Enabled         *bool           `json:"enabled"`
}

// synthTestUpdateReq is the JSON shape for PUT /tests-full/:id.
// All fields optional (PATCH semantics).
type synthTestUpdateReq struct {
	Name            *string         `json:"name"`
	Type            *string         `json:"type"`
	URL             *string         `json:"url"`
	Method          *string         `json:"method"`
	Body            *string         `json:"body"`
	Headers         json.RawMessage `json:"headers"`
	Assertions      json.RawMessage `json:"assertions"`
	IntervalSeconds *int            `json:"interval_seconds"`
	TimeoutMs       *int            `json:"timeout_ms"`
	Locations       *[]string       `json:"locations"`
	SLAUptimePct    *float64        `json:"sla_uptime_pct"`
	SLAResponseMs   *int            `json:"sla_response_ms"`
	Enabled         *bool           `json:"enabled"`
}

// synthTestRow is the canonical row shape returned to clients.
type synthTestRow struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	URL             string     `json:"url"`
	Method          string     `json:"method"`
	IntervalSeconds int        `json:"interval_seconds"`
	TimeoutMs       int        `json:"timeout_ms"`
	Enabled         bool       `json:"enabled"`
	SLAUptimePct    float64    `json:"sla_uptime_pct"`
	SLAResponseMs   int        `json:"sla_response_ms"`
	CreatedAt       string     `json:"created_at"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	LastStatus      *string    `json:"last_status,omitempty"`
}

// CreateSynthTest inserts a new synthetic test for the caller's tenant.
func CreateSynthTest(pool *db.Pool, runner *SyntheticsRunner) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r synthTestReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if !allowedSyntheticTypes[r.Type] {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		method := strings.ToUpper(strings.TrimSpace(r.Method))
		if method == "" {
			method = "GET"
		}
		if r.IntervalSeconds == 0 {
			r.IntervalSeconds = 300
		}
		if r.TimeoutMs == 0 {
			r.TimeoutMs = 30000
		}
		headers := r.Headers
		if len(headers) == 0 || string(headers) == "null" {
			headers = json.RawMessage(`{}`)
		}
		assertions := r.Assertions
		if len(assertions) == 0 || string(assertions) == "null" {
			assertions = json.RawMessage(`[]`)
		}
		locations := r.Locations
		if locations == nil {
			locations = []string{}
		}
		slaUptime := r.SLAUptimePct
		if slaUptime == 0 {
			slaUptime = 99.9
		}
		slaMs := r.SLAResponseMs
		if slaMs == 0 {
			slaMs = 1000
		}
		enabled := true
		if r.Enabled != nil {
			enabled = *r.Enabled
		}
		var id uuid.UUID
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO synthetics_tests
			   (tenant_id, name, type, url, method, body, headers, assertions,
			    interval_seconds, timeout_ms, locations, sla_uptime_pct,
			    sla_response_ms, enabled)
			 VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb,
			         $9, $10, $11, $12, $13, $14)
			 RETURNING id`,
			tenantID, r.Name, r.Type, r.URL, method, nullIfEmpty(r.Body),
			string(headers), string(assertions),
			r.IntervalSeconds, r.TimeoutMs, locations,
			slaUptime, slaMs, enabled,
		).Scan(&id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, synthTestRow{
			ID:              id.String(),
			Name:            r.Name,
			Type:            r.Type,
			URL:             r.URL,
			Method:          method,
			IntervalSeconds: r.IntervalSeconds,
			TimeoutMs:       r.TimeoutMs,
			Enabled:         enabled,
			SLAUptimePct:    slaUptime,
			SLAResponseMs:   slaMs,
			CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// ListSynthTests returns all synthetic tests for the caller's tenant,
// newest first. Optional filters: ?enabled=true&type=http.
func ListSynthTests(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		enabledFilter := c.Query("enabled")
		typeFilter := c.Query("type")
		args := []any{tenantID}
		q := `SELECT id::text, name, type, url, COALESCE(method, 'GET'),
		             interval_seconds, timeout_ms, enabled,
		             COALESCE(sla_uptime_pct, 99.9)::float8, COALESCE(sla_response_ms, 1000),
		             created_at::text
		      FROM synthetics_tests
		      WHERE tenant_id = $1`
		if enabledFilter == "true" {
			args = append(args, true)
			q += " AND enabled = $" + itoa(len(args))
		} else if enabledFilter == "false" {
			args = append(args, false)
			q += " AND enabled = $" + itoa(len(args))
		}
		if typeFilter != "" && allowedSyntheticTypes[typeFilter] {
			args = append(args, typeFilter)
			q += " AND type = $" + itoa(len(args))
		}
		q += " ORDER BY created_at DESC"
		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []synthTestRow{}
		for rows.Next() {
			var t synthTestRow
			var enabled bool
			if err := rows.Scan(&t.ID, &t.Name, &t.Type, &t.URL, &t.Method,
				&t.IntervalSeconds, &t.TimeoutMs, &enabled,
				&t.SLAUptimePct, &t.SLAResponseMs, &t.CreatedAt); err != nil {
				continue
			}
			t.Enabled = enabled
			out = append(out, t)
		}
		kernel.RespondOK(c, gin.H{"tests": out, "total": len(out)})
	}
}

// GetSynthTest returns one test by id.
func GetSynthTest(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("test_id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var t synthTestRow
		var enabled bool
		var body *string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id::text, name, type, url, COALESCE(method, 'GET'),
			        interval_seconds, timeout_ms, enabled,
			        COALESCE(sla_uptime_pct, 99.9)::float8, COALESCE(sla_response_ms, 1000),
			        created_at::text, body
			 FROM synthetics_tests
			 WHERE tenant_id = $1 AND id = $2`,
			tenantID, id,
		).Scan(&t.ID, &t.Name, &t.Type, &t.URL, &t.Method,
			&t.IntervalSeconds, &t.TimeoutMs, &enabled,
			&t.SLAUptimePct, &t.SLAResponseMs, &t.CreatedAt, &body)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		t.Enabled = enabled
		kernel.RespondOK(c, t)
	}
}
