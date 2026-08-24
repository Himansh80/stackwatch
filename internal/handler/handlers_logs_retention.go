// Tier 7 — Log Management Full (D3) — Retention policy endpoints.
//
//	POST /api/v1/logs/retention   — set (or upsert) a retention policy
//	                                 for a given service. Idempotent on
//	                                 (tenant_id, service).
//	GET  /api/v1/logs/retention   — list retention policies for the tenant.
//
// Retention policies are the hot/cold storage windows per service:
//   - hot_days:  how many days logs stay in the fast / expensive tier
//   - cold_days: how many additional days they stay in the slow / cheap
//                tier before being purged
//
// One row per (tenant_id, service). Setting a policy for a service that
// already has one UPDATES the existing row in place (no duplicate row).
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// logRetentionReq is the JSON body for setting a retention policy.
type logRetentionReq struct {
	Service  string `json:"service" binding:"required,min=1,max=128"`
	HotDays  int    `json:"hot_days" binding:"min=1,max=3650"`
	ColdDays int    `json:"cold_days" binding:"min=1,max=3650"`
	Enabled  *bool  `json:"enabled"`
}

// logRetentionRow is the canonical retention shape.
type logRetentionRow struct {
	ID        string `json:"id"`
	Service   string `json:"service"`
	HotDays   int    `json:"hot_days"`
	ColdDays  int    `json:"cold_days"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

// SetLogRetention upserts a retention policy. ON CONFLICT (tenant_id,
// service) DO UPDATE so repeated POSTs simply refresh the windows.
func SetLogRetention(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r logRetentionReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		r.Service = strings.TrimSpace(r.Service)
		if r.HotDays == 0 {
			r.HotDays = 7
		}
		if r.ColdDays == 0 {
			r.ColdDays = 90
		}
		// cold_days must be >= hot_days — otherwise there's no cold tier.
		if r.ColdDays < r.HotDays {
			kernel.RespondErrorWithCode(c, 400, "bad_retention_window",
				"cold_days must be >= hot_days")
			return
		}
		enabled := true
		if r.Enabled != nil {
			enabled = *r.Enabled
		}
		// ON CONFLICT ... DO UPDATE returns the resulting row so we can
		// echo it back without a second SELECT.
		var (
			id        string
			createdAt string
			gotEnabled bool
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO log_retention_policies
			   (tenant_id, service, hot_days, cold_days, enabled)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (tenant_id, service) DO UPDATE
			 SET hot_days = EXCLUDED.hot_days,
			     cold_days = EXCLUDED.cold_days,
			     enabled = EXCLUDED.enabled
			 RETURNING id::text, created_at::text, enabled`,
			tenantID, r.Service, r.HotDays, r.ColdDays, enabled,
		).Scan(&id, &createdAt, &gotEnabled)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, logRetentionRow{
			ID:        id,
			Service:   r.Service,
			HotDays:   r.HotDays,
			ColdDays:  r.ColdDays,
			Enabled:   gotEnabled,
			CreatedAt: createdAt,
		})
	}
}

// ListLogRetention returns all retention policies for the tenant.
func ListLogRetention(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, service, hot_days, cold_days, enabled, created_at::text
			 FROM log_retention_policies
			 WHERE tenant_id = $1
			 ORDER BY service ASC`,
			tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []logRetentionRow{}
		for rows.Next() {
			var p logRetentionRow
			var enabled bool
			if err := rows.Scan(&p.ID, &p.Service, &p.HotDays, &p.ColdDays, &enabled, &p.CreatedAt); err != nil {
				continue
			}
			p.Enabled = enabled
			out = append(out, p)
		}
		kernel.RespondOK(c, gin.H{"policies": out, "total": len(out)})
	}
}
