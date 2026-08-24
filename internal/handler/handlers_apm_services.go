// Tier 7 — APM (D2) — Service endpoints.
//
//	POST /api/v1/apm/services      — register a service (idempotent)
//	GET  /api/v1/apm/services      — list services for the caller's tenant
//	GET  /api/v1/apm/services/:id  — service detail + live metrics (req/s, err %, p95)
//
// Every query honors tenant_id from the JWT — no cross-tenant data ever
// crosses the wire.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// apmServiceReq is the JSON shape for creating a service.
type apmServiceReq struct {
	Name        string `json:"name" binding:"required,min=1,max=128"`
	Language    string `json:"language" binding:"max=64"`
	Framework   string `json:"framework" binding:"max=64"`
	Environment string `json:"environment" binding:"max=64"`
}

// apmService is the JSON shape returned by the service endpoints.
type apmService struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Language    string `json:"language"`
	Framework   string `json:"framework"`
	Environment string `json:"environment"`
	CreatedAt   string `json:"created_at"`
}

// apmServiceDetail wraps a service with its computed live metrics.
type apmServiceDetail struct {
	apmService
	RequestRate float64 `json:"request_rate"` // req/sec, last 5m
	ErrorRate   float64 `json:"error_rate"`   // 0..1
	P95Latency  float64 `json:"p95_latency"`  // ms
}

// RegisterAPMService creates a service record. Idempotent on
// (tenant_id, name): if a service with the same name already exists in
// this tenant we return the existing row (200) instead of failing.
func RegisterAPMService(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r apmServiceReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		r.Name = strings.TrimSpace(r.Name)
		r.Language = strings.TrimSpace(r.Language)
		r.Framework = strings.TrimSpace(r.Framework)
		r.Environment = strings.TrimSpace(r.Environment)
		// Upsert: ON CONFLICT DO NOTHING then SELECT to return either
		// the freshly inserted row or the pre-existing one.
		var svc apmService
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO apm_services (tenant_id, name, language, framework, environment)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (tenant_id, name) DO NOTHING
			 RETURNING id, name, COALESCE(language, ''), COALESCE(framework, ''),
			           COALESCE(environment, ''), created_at::text`,
			tenantID, r.Name, r.Language, r.Framework, r.Environment,
		).Scan(&svc.ID, &svc.Name, &svc.Language, &svc.Framework, &svc.Environment, &svc.CreatedAt)
		if err != nil {
			// No row returned — already exists. Fetch it.
			err = pool.Pgx().QueryRow(c.Request.Context(),
				`SELECT id, name, COALESCE(language, ''), COALESCE(framework, ''),
				        COALESCE(environment, ''), created_at::text
				 FROM apm_services WHERE tenant_id = $1 AND name = $2`,
				tenantID, r.Name,
			).Scan(&svc.ID, &svc.Name, &svc.Language, &svc.Framework, &svc.Environment, &svc.CreatedAt)
			if err != nil {
				kernel.RespondError(c, err)
				return
			}
			kernel.RespondOK(c, gin.H{"service": svc, "created": false})
			return
		}
		kernel.RespondCreated(c, gin.H{"service": svc, "created": true})
	}
}

// ListAPMServices returns all services for the caller's tenant.
func ListAPMServices(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id, name, COALESCE(language, ''), COALESCE(framework, ''),
			        COALESCE(environment, ''), created_at::text
			 FROM apm_services
			 WHERE tenant_id = $1
			 ORDER BY created_at DESC`,
			tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []apmService{}
		for rows.Next() {
			var s apmService
			if err := rows.Scan(&s.ID, &s.Name, &s.Language, &s.Framework, &s.Environment, &s.CreatedAt); err != nil {
				continue
			}
			out = append(out, s)
		}
		kernel.RespondOK(c, gin.H{"services": out, "total": len(out)})
	}
}

// GetAPMService returns one service plus its computed live metrics
// (request rate / error rate / p95 latency) over the last 5 minutes.
func GetAPMService(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var svc apmService
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id, name, COALESCE(language, ''), COALESCE(framework, ''),
			        COALESCE(environment, ''), created_at::text
			 FROM apm_services WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&svc.ID, &svc.Name, &svc.Language, &svc.Framework, &svc.Environment, &svc.CreatedAt)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		// Compute metrics over the last 5m window from apm_spans.
		// request_rate = COUNT(spans) / 300s
		// error_rate   = COUNT(spans WHERE status='error') / COUNT(spans)
		// p95_latency  = percentile_cont(0.95) of duration_us (converted to ms)
		var requestRate, errorRate, p95 float64
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT
			   COALESCE(COUNT(*)::float / 300.0, 0)                                       AS request_rate,
			   COALESCE(COUNT(*) FILTER (WHERE status = 'error')::float
			            / NULLIF(COUNT(*)::float, 0), 0)                                AS error_rate,
			   COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_us) / 1000.0, 0)
			                                                                             AS p95_ms
			 FROM apm_spans
			 WHERE tenant_id = $1 AND service_id = $2
			   AND started_at > NOW() - INTERVAL '5 minutes'`,
			tenantID, id,
		).Scan(&requestRate, &errorRate, &p95)
		if err != nil {
			// No data or query failed — return service with zero metrics.
			requestRate, errorRate, p95 = 0, 0, 0
		}
		kernel.RespondOK(c, apmServiceDetail{
			apmService:  svc,
			RequestRate: requestRate,
			ErrorRate:   errorRate,
			P95Latency:  p95,
		})
	}
}
