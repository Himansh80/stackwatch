// Tier 11 Phase 2 — Usage Metering (PL2).
//
// HTTP route handlers for the 2 platform-wide metering endpoints:
//
//	GET /api/v1/platform/usage/summary — GetUsageSummary (super_admin only)
//	GET /api/v1/platform/usage/export  — GetUsageExport  (auth-protected)
//
// These two are split from handlers_platform_usage.go to keep
// each handler file under the 400-LOC cap. /summary is the
// cross-tenant billing dashboard query (gated by super_admin);
// /export streams raw platform_usage_events rows out as CSV so
// the finance team can slice the data in Excel without holding
// the whole result-set in memory.
//
// Shared types (usageTenantSummary, usageSummaryResp) live in
// handlers_platform_usage_types.go. The hourly aggregation worker
// lives in internal/platform/usage_meter.go.
package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/platform/usage/summary
// ------------------------------------------------------------------

// GetUsageSummary returns per-tenant totals for the platform-wide
// billing dashboard. Guarded by super_admin so an org admin can't
// enumerate every other tenant on the instance.
//
// Window: last 30 days (fixed; dashboard has its own period
// controls as a future polish).
func GetUsageSummary(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsUser, ok := userFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if claimsUser.Role != "super_admin" {
			kernel.RespondErrorWithCode(c, http.StatusForbidden, "forbidden",
				"super_admin role required")
			return
		}
		now := time.Now().UTC()
		since := now.Add(-30 * 24 * time.Hour)

		// One row per tenant for the window. LEFT JOIN keeps
		// tenants with zero usage in the window visible (with
		// zero counts) so the platform admin doesn't have to
		// cross-reference a separate tenant list to know who's
		// on the instance.
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT t.id::text, COALESCE(t.name, ''), COALESCE(t.plan, 'free'),
			        COALESCE(SUM(a.count), 0)::bigint,
			        COALESCE(SUM(a.sum_quantity), 0)::float8,
			        (
			          SELECT a2.event_kind
			            FROM platform_usage_aggregates a2
			           WHERE a2.tenant_id = t.id
			             AND a2.bucket_ts >= $1
			           GROUP BY a2.event_kind
			           ORDER BY SUM(a2.count) DESC
			           LIMIT 1
			        ) AS top_kind
			   FROM tenants t
			   LEFT JOIN platform_usage_aggregates a
			          ON a.tenant_id = t.id
			         AND a.bucket_ts >= $1
			  GROUP BY t.id, t.name, t.plan
			  ORDER BY SUM(a.count) DESC NULLS LAST
			  LIMIT 500`,
			since)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		var tenants []usageTenantSummary
		for rows.Next() {
			var ts usageTenantSummary
			if err := rows.Scan(&ts.TenantID, &ts.TenantName, &ts.Plan,
				&ts.TotalCount, &ts.SumQuantity, &ts.TopKind); err != nil {
				continue
			}
			if ts.TopKind == "" {
				ts.TopKind = "—"
			}
			tenants = append(tenants, ts)
		}
		kernel.RespondOK(c, usageSummaryResp{Tenants: tenants})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/platform/usage/export
// ------------------------------------------------------------------

// GetUsageExport streams the caller's tenant's raw
// platform_usage_events rows as CSV (newline-delimited strings).
// Streams directly to the ResponseWriter so the export never
// buffers more than one row in memory — a tenant with millions of
// events stays one round-trip, no OOM.
//
// Query:
//
//	event_kind  — optional, narrow to one kind
//	format      — default csv; only csv is shipped today (Phase 8
//	              will add ndjson when forecasting arrives)
//	limit       — default 5000, hard cap 100000 to prevent a
//	              runaway export blocking the worker
func GetUsageExport(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		format := strings.ToLower(strings.TrimSpace(c.Query("format")))
		if format == "" {
			format = "csv"
		}
		if format != "csv" {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"only format=csv is supported today")
			return
		}
		eventKind := strings.TrimSpace(c.Query("event_kind"))
		if eventKind != "" && !isAllowedEventKind(eventKind) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"event_kind is not in the allowlist")
			return
		}
		limit := 5000
		if lq := strings.TrimSpace(c.Query("limit")); lq != "" {
			if n, err := strconv.Atoi(lq); err == nil && n > 0 && n <= 100000 {
				limit = n
			}
		}

		// Build the WHERE clause + args incrementally so we never
		// hand-roll SQL strings into log messages. The LIMIT
		// placeholder uses the current arg count so pgx doesn't
		// have to handle a string-typed number.
		where := "tenant_id = $1"
		args := []interface{}{tenantID}
		if eventKind != "" {
			args = append(args, eventKind)
			where += " AND event_kind = $2"
		}
		args = append(args, limit)
		q := fmt.Sprintf(
			`SELECT id::text, COALESCE(user_id::text, ''), event_kind, quantity::float8,
			        unit, COALESCE(resource_id, ''), metadata::text, created_at::text
			   FROM platform_usage_events
			  WHERE %s
			  ORDER BY created_at DESC
			  LIMIT $%d`,
			where, len(args),
		)

		pgxRows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer pgxRows.Close()

		filename := fmt.Sprintf("usage-export-%s.csv",
			time.Now().UTC().Format("2006-01-02T1504"))
		c.Header("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`, filename))
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Status(http.StatusOK)

		w := c.Writer
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{
			"id", "user_id", "event_kind", "quantity", "unit",
			"resource_id", "metadata", "created_at",
		})
		for pgxRows.Next() {
			var (
				id, uid, ek, unit, rid, meta, ts string
				qty                              float64
			)
			if err := pgxRows.Scan(&id, &uid, &ek, &qty, &unit, &rid, &meta, &ts); err != nil {
				continue
			}
			_ = cw.Write([]string{id, uid, ek,
				strconv.FormatFloat(qty, 'f', -1, 64),
				unit, rid, meta, ts,
			})
		}
		cw.Flush()
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}
