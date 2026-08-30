// Tier 7 — Log Management Full (D3) — Archive + rehydration endpoints.
//
//	POST /api/v1/logs/archives                  — create an archive destination
//	GET  /api/v1/logs/archives                  — list archives for the tenant
//	POST /api/v1/logs/archives/:id/rehydrate    — queue a rehydration job
//	GET  /api/v1/logs/rehydrations              — list rehydrations (filter ?archive_id=)
//
// Archives describe where logs get sent for long-term cold storage.
// Rehydration is the reverse flow: a tenant asks "re-pull my logs from
// archive X between time A and time B back into the live index", and
// we queue a job whose status can be polled.
//
// The actual archive / rehydrate workers are out of scope for Phase 2
// (the surface is storage + the UI). This file owns the HTTP boundary
// only.
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

// allowedArchiveDestinationTypes is the set of destination_type values
// we'll accept on archive create. Matches spec.md §"Migration 031":
// 's3' | 'gcs' | 'azure' | 'local'.
var allowedArchiveDestinationTypes = map[string]bool{
	"s3":    true,
	"gcs":   true,
	"azure": true,
	"local": true,
}

// logArchiveReq is the JSON body for creating an archive destination.
//
// destination_config is a free-form object whose shape depends on the
// destination_type (e.g. bucket + region for s3, path for local).
// We persist it as JSONB so the UI can render the right fields later.
type logArchiveReq struct {
	Name              string          `json:"name" binding:"required,min=1,max=128"`
	DestinationType   string          `json:"destination_type" binding:"required,max=32"`
	DestinationConfig json.RawMessage `json:"destination_config"`
	Enabled           *bool           `json:"enabled"`
}

// logArchiveRow is the canonical archive shape returned to clients.
type logArchiveRow struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	DestinationType   string          `json:"destination_type"`
	DestinationConfig json.RawMessage `json:"destination_config"`
	Enabled           bool            `json:"enabled"`
	CreatedAt         string          `json:"created_at"`
}

// CreateLogArchive registers a new archive destination.
func CreateLogArchive(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r logArchiveReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		r.Name = strings.TrimSpace(r.Name)
		r.DestinationType = strings.ToLower(strings.TrimSpace(r.DestinationType))
		if !allowedArchiveDestinationTypes[r.DestinationType] {
			kernel.RespondErrorWithCode(c, 400, "bad_destination_type",
				"destination_type must be one of: s3, gcs, azure, local")
			return
		}
		cfg := r.DestinationConfig
		if len(cfg) == 0 || string(cfg) == "null" {
			cfg = json.RawMessage(`{}`)
		}
		enabled := true
		if r.Enabled != nil {
			enabled = *r.Enabled
		}
		var id uuid.UUID
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO log_archives
			   (tenant_id, name, destination_type, destination_config, enabled)
			 VALUES ($1, $2, $3, $4::jsonb, $5)
			 RETURNING id`,
			tenantID, r.Name, r.DestinationType, string(cfg), enabled,
		).Scan(&id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, logArchiveRow{
			ID:                id.String(),
			Name:              r.Name,
			DestinationType:   r.DestinationType,
			DestinationConfig: cfg,
			Enabled:           enabled,
			CreatedAt:         time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// ListLogArchives returns all archive destinations for the tenant.
func ListLogArchives(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, name, destination_type, destination_config::text, enabled, created_at::text
			 FROM log_archives
			 WHERE tenant_id = $1
			 ORDER BY created_at DESC`,
			tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []logArchiveRow{}
		for rows.Next() {
			var a logArchiveRow
			var cfgStr string
			var enabled bool
			if err := rows.Scan(&a.ID, &a.Name, &a.DestinationType, &cfgStr, &enabled, &a.CreatedAt); err != nil {
				continue
			}
			a.Enabled = enabled
			a.DestinationConfig = json.RawMessage(cfgStr)
			out = append(out, a)
		}
		kernel.RespondOK(c, gin.H{"archives": out, "total": len(out)})
	}
}

// rehydrateReq is the JSON body for POST /logs/archives/:id/rehydrate.
//
// start_time and end_time are RFC3339 timestamps; we accept any
// timezone and rely on Postgres to normalize.
type rehydrateReq struct {
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`
}

// logRehydrationRow is the canonical rehydration shape.
type logRehydrationRow struct {
	ID          string    `json:"id"`
	ArchiveID   string    `json:"archive_id"`
	ArchiveName string    `json:"archive_name"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Status      string    `json:"status"`
	ProgressPct int       `json:"progress_pct"`
	CreatedAt   string    `json:"created_at"`
}

// TriggerLogRehydration queues a rehydration job against the named
// archive. Returns 201 with the new rehydration_id and status='queued'.
// The caller can poll GET /api/v1/logs/rehydrations to see progress.
func TriggerLogRehydration(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		archiveID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var r rehydrateReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if !r.EndTime.After(r.StartTime) {
			kernel.RespondErrorWithCode(c, 400, "bad_time_range",
				"end_time must be after start_time")
			return
		}
		// Confirm archive belongs to tenant before we accept the job.
		var archiveName string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT name FROM log_archives WHERE id = $1 AND tenant_id = $2`,
			archiveID, tenantID,
		).Scan(&archiveName)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		var id uuid.UUID
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO log_rehydrations
			   (tenant_id, archive_id, start_time, end_time, status, progress_pct)
			 VALUES ($1, $2, $3, $4, 'queued', 0)
			 RETURNING id`,
			tenantID, archiveID, r.StartTime, r.EndTime,
		).Scan(&id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, gin.H{
			"rehydration_id": id.String(),
			"archive_id":     archiveID.String(),
			"archive_name":   archiveName,
			"start_time":     r.StartTime.UTC().Format(time.RFC3339),
			"end_time":       r.EndTime.UTC().Format(time.RFC3339),
			"status":         "queued",
			"progress_pct":   0,
		})
	}
}

// ListLogRehydrations returns rehydration jobs for the tenant, newest
// first. Optional ?archive_id=<uuid> filter narrows to one archive.
func ListLogRehydrations(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		args := []any{tenantID}
		q := `SELECT r.id::text, r.archive_id::text, a.name,
		             r.start_time, r.end_time, r.status, r.progress_pct, r.created_at::text
		      FROM log_rehydrations r
		      JOIN log_archives a ON a.id = r.archive_id
		      WHERE r.tenant_id = $1`
		if arcID := c.Query("archive_id"); arcID != "" {
			if id, err := uuid.Parse(arcID); err == nil {
				args = append(args, id)
				q += " AND r.archive_id = $2"
			}
		}
		q += " ORDER BY r.created_at DESC LIMIT 200"
		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []logRehydrationRow{}
		for rows.Next() {
			var r logRehydrationRow
			if err := rows.Scan(&r.ID, &r.ArchiveID, &r.ArchiveName,
				&r.StartTime, &r.EndTime, &r.Status, &r.ProgressPct, &r.CreatedAt); err != nil {
				continue
			}
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{"rehydrations": out, "total": len(out)})
	}
}
