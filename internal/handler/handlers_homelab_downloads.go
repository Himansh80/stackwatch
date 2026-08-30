// Tier 10 Phase 5 — Download Stats (H5). 3 protected endpoints
// back the per-user download client registry (homelab_download_clients):
//
//	GET    /api/v1/homelab/downloads/clients      — ListHomelabDownloadClients
//	POST   /api/v1/homelab/downloads/clients      — CreateHomelabDownloadClient
//	DELETE /api/v1/homelab/downloads/clients/:id  — DeleteHomelabDownloadClient
//
// The snapshots endpoint (GET /downloads/snapshots) lives in
// handlers_homelab_downloads_snapshots.go (split for file-size
// discipline — keeps each file under 400 LOC).
//
// The DownloadsWorker (internal/homelab/downloads.go) ticks every
// 60s and INSERTs a snapshot row per enabled client. POST /clients
// fires an immediate fire-and-forget poll after create via
// PollClientAndInsertSnapshot so a freshly-pinned client shows
// stats within seconds rather than waiting a full tick.
//
// Per-user (NOT per-tenant): every WHERE clause filters by both
// tenant_id and user_id so user A can never see or modify user B's
// download clients — matches US-8 in the speckit proposal.
package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// Validation helpers (validateDownloadClientName,
// validateDownloadKind, validateDownloadBaseURL,
// validateDownloadCredentials, countDownloadClients) live in
// handlers_homelab_downloads_types.go so this file can stay under
// the 400-LOC cap.

// ------------------------------------------------------------------
// GET /api/v1/homelab/downloads/clients
// ------------------------------------------------------------------

// ListHomelabDownloadClients returns every download client the
// caller has pinned, joined with the latest snapshot so the
// dashboard can render the per-client stat row in one round-trip.
//
// LEFT JOIN to the latest snapshot via DISTINCT ON — same pattern
// as ListHomelabServices's latest-health join. api_key / username
// / password are intentionally NOT echoed back to the frontend.
func ListHomelabDownloadClients(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT c.id::text, c.tenant_id::text, c.user_id::text, c.name,
			        c.kind, c.base_url, c.enabled,
			        c.last_polled_at::text, c.last_poll_status, c.last_poll_error,
			        c.created_at::text, c.updated_at::text,
			        s.queue_count, s.queue_size_bytes,
			        s.download_speed_bytes_per_sec, s.upload_speed_bytes_per_sec,
			        s.today_downloaded_bytes, s.today_uploaded_bytes,
			        s.polled_at::text
			   FROM homelab_download_clients c
			   LEFT JOIN (
			       SELECT DISTINCT ON (client_id)
			              client_id, queue_count, queue_size_bytes,
			              download_speed_bytes_per_sec, upload_speed_bytes_per_sec,
			              today_downloaded_bytes, today_uploaded_bytes,
			              polled_at
			         FROM homelab_download_snapshots
			        ORDER BY client_id, polled_at DESC
			   ) s ON s.client_id = c.id
			  WHERE c.tenant_id = $1 AND c.user_id = $2
			  ORDER BY c.created_at DESC`,
			tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := []downloadClientRow{}
		for rows.Next() {
			var (
				r        downloadClientRow
				lastPol  *string
				lastStat *string
				lastErr  *string
				// LEFT JOIN may return NULLs for the snapshot cols;
				// use sql.Null types via *int / *int64 / *string
				// pointers so we can detect "no snapshot yet".
				queueCount   *int
				queueBytes   *int64
				downSpeed    *int64
				upSpeed      *int64
				todayDown    *int64
				todayUp      *int64
				snapPolledAt *string
			)
			if err := rows.Scan(&r.ID, &r.TenantID, &r.UserID, &r.Name,
				&r.Kind, &r.BaseURL, &r.Enabled,
				&lastPol, &lastStat, &lastErr,
				&r.CreatedAt, &r.UpdatedAt,
				&queueCount, &queueBytes,
				&downSpeed, &upSpeed,
				&todayDown, &todayUp,
				&snapPolledAt); err != nil {
				kernel.RespondError(c, err)
				return
			}
			if lastPol != nil {
				r.LastPolledAt = *lastPol
			}
			if lastStat != nil {
				r.LastPollStatus = *lastStat
			}
			if lastErr != nil {
				r.LastPollError = *lastErr
			}
			if queueCount != nil {
				r.LatestSnapshot = &downloadSnapshotBlock{
					ClientID:                 r.ID,
					QueueCount:               *queueCount,
					QueueSizeBytes:           *queueBytes,
					DownloadSpeedBytesPerSec: *downSpeed,
					UploadSpeedBytesPerSec:   *upSpeed,
					TodayDownloadedBytes:     *todayDown,
					TodayUploadedBytes:       *todayUp,
					PolledAt:                 *snapPolledAt,
				}
			}
			out = append(out, r)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"clients": out,
			"count":   len(out),
		})
	}
}

// ------------------------------------------------------------------
// POST /api/v1/homelab/downloads/clients
// ------------------------------------------------------------------

// CreateHomelabDownloadClient inserts a new download client for
// the caller. Fires an immediate fire-and-forget poll after
// INSERT via PollClientAndInsertSnapshot so a freshly-pinned
// client shows stats within seconds rather than waiting a full
// 60s tick.
//
// Validates name (non-empty, ≤ maxDownloadNameLen), kind
// (allowedDownloadClientKinds whitelist), base_url (http/https
// parseable), and credentials (api_key OR username+password
// per the kind).
//
// Returns 201 with the inserted row (no secret echo). Honors the
// per-user soft cap (maxDownloadClientsPerUser) — returns 400
// rather than 409 so the frontend can surface the message
// uniformly.
func CreateHomelabDownloadClient(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		var req downloadClientReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.Kind = strings.ToLower(strings.TrimSpace(req.Kind))
		req.BaseURL = strings.TrimSpace(req.BaseURL)
		req.APIKey = strings.TrimSpace(req.APIKey)
		req.Username = strings.TrimSpace(req.Username)
		_ = req.Password // don't trim — leading/trailing spaces in a password are intentional

		if err := validateDownloadClientName(req.Name); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if err := validateDownloadKind(req.Kind); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if err := validateDownloadBaseURL(req.BaseURL); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if err := validateDownloadCredentials(req.Kind, req.APIKey, req.Username, req.Password); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}

		// Soft cap on per-user pinned clients. Counted BEFORE
		// the INSERT so a rejected user sees "limit reached" rather
		// than "conflict" on a unique-violation.
		current, cerr := countDownloadClients(c.Request.Context(), pool, tenantID, userID)
		if cerr != nil {
			kernel.RespondError(c, cerr)
			return
		}
		if current >= maxDownloadClientsPerUser {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "limit_reached",
				fmt.Sprintf("pin limit reached (%d). remove a client before adding another.", maxDownloadClientsPerUser))
			return
		}

		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}

		var (
			id, createdAt, updatedAt string
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO homelab_download_clients
			        (tenant_id, user_id, name, kind, base_url,
			         api_key, username, password, enabled)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 RETURNING id::text, created_at::text, updated_at::text`,
			tenantID, userID, req.Name, req.Kind, req.BaseURL,
			nullableString(req.APIKey), nullableString(req.Username), nullableString(req.Password),
			enabled,
		).Scan(&id, &createdAt, &updatedAt)
		if err != nil {
			// UNIQUE (tenant_id, user_id, name) — surface as 409.
			if strings.Contains(err.Error(), "homelab_download_clients_tenant_id_user_id_name_key") {
				kernel.RespondErrorWithCode(c, http.StatusConflict, "conflict",
					"a client with that name already exists")
				return
			}
			kernel.RespondError(c, err)
			return
		}

		// Fire-and-forget immediate poll so the dashboard shows
		// real stats within seconds. PollClientAndInsertSnapshot
		// is the shared code path with the DownloadsWorker
		// (DRY pattern — see handlers_homelab_downloads_snapshots.go).
		newID, perr := uuid.Parse(id)
		if perr == nil && enabled {
			// Use a background context so the request's short
			// timeout doesn't kill the poll mid-flight. The
			// poll itself uses its own 30s context.
			go PollClientAndInsertSnapshot(context.Background(), pool, newID)
		}

		kernel.RespondCreated(c, downloadClientRow{
			ID:        id,
			TenantID:  tenantID.String(),
			UserID:    userID.String(),
			Name:      req.Name,
			Kind:      req.Kind,
			BaseURL:   req.BaseURL,
			Enabled:   enabled,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
}

// ------------------------------------------------------------------
// DELETE /api/v1/homelab/downloads/clients/:id
// ------------------------------------------------------------------

// DeleteHomelabDownloadClient removes a download client (and its
// snapshots via ON DELETE CASCADE FK). Idempotent: 200 with
// {deleted: 0} when the row doesn't exist or isn't the caller's.
func DeleteHomelabDownloadClient(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "id must be a uuid")
			return
		}

		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM homelab_download_clients
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondOK(c, gin.H{
			"deleted":       tag.RowsAffected(),
			"client_id":     id.String(),
			"idempotent_ok": true,
		})
	}
}
