// Tier 7 Phase 2 — Synthetics Full (D5) — Test partial update + delete.
//
//	PUT    /api/v1/synthetics/tests-full/:test_id  — partial update
//	DELETE /api/v1/synthetics/tests-full/:test_id  — delete (cascades to test_runs)
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// UpdateSynthTest applies a partial update. Returns 404 if missing.
func UpdateSynthTest(pool *db.Pool) gin.HandlerFunc {
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
		var r synthTestUpdateReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		sets := []string{}
		args := []any{}
		idx := 1
		if r.Name != nil {
			args = append(args, strings.TrimSpace(*r.Name))
			sets = append(sets, "name = $"+itoa(idx))
			idx++
		}
		if r.Type != nil {
			if !allowedSyntheticTypes[*r.Type] {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
			args = append(args, *r.Type)
			sets = append(sets, "type = $"+itoa(idx))
			idx++
		}
		if r.URL != nil {
			args = append(args, *r.URL)
			sets = append(sets, "url = $"+itoa(idx))
			idx++
		}
		if r.Method != nil {
			args = append(args, strings.ToUpper(strings.TrimSpace(*r.Method)))
			sets = append(sets, "method = $"+itoa(idx))
			idx++
		}
		if r.Body != nil {
			args = append(args, nullIfEmpty(*r.Body))
			sets = append(sets, "body = $"+itoa(idx))
			idx++
		}
		if r.Headers != nil && string(r.Headers) != "null" {
			args = append(args, string(r.Headers))
			sets = append(sets, "headers = $"+itoa(idx)+"::jsonb")
			idx++
		}
		if r.Assertions != nil && string(r.Assertions) != "null" {
			args = append(args, string(r.Assertions))
			sets = append(sets, "assertions = $"+itoa(idx)+"::jsonb")
			idx++
		}
		if r.IntervalSeconds != nil {
			args = append(args, *r.IntervalSeconds)
			sets = append(sets, "interval_seconds = $"+itoa(idx))
			idx++
		}
		if r.TimeoutMs != nil {
			args = append(args, *r.TimeoutMs)
			sets = append(sets, "timeout_ms = $"+itoa(idx))
			idx++
		}
		if r.Locations != nil {
			args = append(args, *r.Locations)
			sets = append(sets, "locations = $"+itoa(idx))
			idx++
		}
		if r.SLAUptimePct != nil {
			args = append(args, *r.SLAUptimePct)
			sets = append(sets, "sla_uptime_pct = $"+itoa(idx))
			idx++
		}
		if r.SLAResponseMs != nil {
			args = append(args, *r.SLAResponseMs)
			sets = append(sets, "sla_response_ms = $"+itoa(idx))
			idx++
		}
		if r.Enabled != nil {
			args = append(args, *r.Enabled)
			sets = append(sets, "enabled = $"+itoa(idx))
			idx++
		}
		if len(sets) == 0 {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		args = append(args, tenantID, id)
		q := "UPDATE synthetics_tests SET " + strings.Join(sets, ", ") +
			" WHERE tenant_id = $" + itoa(idx) + " AND id = $" + itoa(idx+1)
		tag, err := pool.Pgx().Exec(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		// Return refreshed row.
		var t synthTestRow
		var enabled bool
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id::text, name, type, url, COALESCE(method, 'GET'),
			        interval_seconds, timeout_ms, enabled,
			        COALESCE(sla_uptime_pct, 99.9)::float8, COALESCE(sla_response_ms, 1000),
			        created_at::text
			 FROM synthetics_tests
			 WHERE tenant_id = $1 AND id = $2`,
			tenantID, id,
		).Scan(&t.ID, &t.Name, &t.Type, &t.URL, &t.Method,
			&t.IntervalSeconds, &t.TimeoutMs, &enabled,
			&t.SLAUptimePct, &t.SLAResponseMs, &t.CreatedAt)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		t.Enabled = enabled
		kernel.RespondOK(c, t)
	}
}

// DeleteSynthTest removes a test. CASCADEs to test_runs.
func DeleteSynthTest(pool *db.Pool) gin.HandlerFunc {
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
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM synthetics_tests WHERE tenant_id = $1 AND id = $2`,
			tenantID, id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondOK(c, gin.H{"deleted": id.String()})
	}
}