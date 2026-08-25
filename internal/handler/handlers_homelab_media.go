// Tier 10 Phase 6 — Media Server (H6). 3 protected endpoints back
// the per-user media-server registry (homelab_media_servers) CRUD:
//
//	GET    /api/v1/homelab/media/servers      — ListHomelabMediaServers
//	POST   /api/v1/homelab/media/servers      — CreateHomelabMediaServer
//	DELETE /api/v1/homelab/media/servers/:id  — DeleteHomelabMediaServer
//
// The state endpoint (GET /media/state) lives in
// handlers_homelab_media_state.go (split for file-size discipline
// — keeps each file under 400 LOC).
//
// The PollMediaServerAndInsertState helper (used by POST
// /media/servers for the immediate fire-and-forget poll fired
// after create) lives in handlers_homelab_media_poll.go. The
// MediaWorker (internal/homelab/media.go) ticks every 60s and
// calls the same homelab.PollMediaServer dispatch — DRY pattern
// (no twin fetch implementations to keep in sync).
//
// Per-user (NOT per-tenant): every WHERE clause filters by both
// tenant_id and user_id so user A can never see or modify user B's
// media servers — matches US-8 in the speckit proposal.
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

// ------------------------------------------------------------------
// GET /api/v1/homelab/media/servers
// ------------------------------------------------------------------

// ListHomelabMediaServers returns every media server the caller
// has pinned, joined with the latest poll's bookkeeping so the
// dashboard can render the per-server chip in one round-trip.
//
// api_key is intentionally NOT echoed back to the frontend — same
// contract as ListHomelabDownloadClients.
func ListHomelabMediaServers(pool *db.Pool) gin.HandlerFunc {
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
			`SELECT id::text, tenant_id::text, user_id::text, name,
			        kind, base_url, enabled,
			        last_polled_at::text, last_poll_status, last_poll_error,
			        created_at::text, updated_at::text
			   FROM homelab_media_servers
			  WHERE tenant_id = $1 AND user_id = $2
			  ORDER BY created_at DESC`,
			tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := []mediaServerRow{}
		for rows.Next() {
			var (
				r        mediaServerRow
				lastPol  *string
				lastStat *string
				lastErr  *string
			)
			if err := rows.Scan(&r.ID, &r.TenantID, &r.UserID, &r.Name,
				&r.Kind, &r.BaseURL, &r.Enabled,
				&lastPol, &lastStat, &lastErr,
				&r.CreatedAt, &r.UpdatedAt); err != nil {
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
			out = append(out, r)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"servers": out,
			"count":   len(out),
		})
	}
}



// ------------------------------------------------------------------
// POST /api/v1/homelab/media/servers
// ------------------------------------------------------------------

// CreateHomelabMediaServer inserts a new media server for the
// caller. Fires an immediate fire-and-forget poll after INSERT
// via PollMediaServerAndInsertState so a freshly-pinned server
// shows now-playing + recent additions within seconds rather than
// waiting a full 60s tick.
//
// Validates name (non-empty, ≤ maxMediaServerNameLen), kind
// (allowedMediaServerKinds whitelist), base_url (http/https
// parseable), and api_key (non-empty trimmed — all three kinds
// use token-based auth with no username/password fallback).
//
// Returns 201 with the inserted row (no api_key echo). Honors the
// per-user soft cap (maxMediaServersPerUser) — returns 400 rather
// than 409 so the frontend can surface the message uniformly.
func CreateHomelabMediaServer(pool *db.Pool) gin.HandlerFunc {
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

		var req mediaServerReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.Kind = strings.ToLower(strings.TrimSpace(req.Kind))
		req.BaseURL = strings.TrimSpace(req.BaseURL)
		req.APIKey = strings.TrimSpace(req.APIKey)

		if err := validateMediaServerName(req.Name); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if err := validateMediaServerKind(req.Kind); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if err := validateMediaServerBaseURL(req.BaseURL); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if err := validateMediaServerAPIKey(req.APIKey); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}

		// Soft cap on per-user pinned servers. Counted BEFORE
		// the INSERT so a rejected user sees "limit reached" rather
		// than "conflict" on a unique-violation.
		current, cerr := countMediaServers(c.Request.Context(), pool, tenantID, userID)
		if cerr != nil {
			kernel.RespondError(c, cerr)
			return
		}
		if current >= maxMediaServersPerUser {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "limit_reached",
				fmt.Sprintf("pin limit reached (%d). remove a server before adding another.", maxMediaServersPerUser))
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
			`INSERT INTO homelab_media_servers
			        (tenant_id, user_id, name, kind, base_url,
			         api_key, enabled)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING id::text, created_at::text, updated_at::text`,
			tenantID, userID, req.Name, req.Kind, req.BaseURL,
			req.APIKey, enabled,
		).Scan(&id, &createdAt, &updatedAt)
		if err != nil {
			// UNIQUE (tenant_id, user_id, name) — surface as 409.
			if strings.Contains(err.Error(), "homelab_media_servers_tenant_id_user_id_name_key") {
				kernel.RespondErrorWithCode(c, http.StatusConflict, "conflict",
					"a server with that name already exists")
				return
			}
			kernel.RespondError(c, err)
			return
		}

		// Fire-and-forget immediate poll so the dashboard shows
		// real state within seconds. PollMediaServerAndInsertState
		// is the shared code path with the MediaWorker
		// (DRY pattern — see PollMediaServerAndInsertState below).
		newID, perr := uuid.Parse(id)
		if perr == nil && enabled {
			// Use a background context so the request's short
			// timeout doesn't kill the poll mid-flight. The
			// poll itself uses its own 30s context.
			go PollMediaServerAndInsertState(context.Background(), pool, newID)
		}

		kernel.RespondCreated(c, mediaServerRow{
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
// DELETE /api/v1/homelab/media/servers/:id
// ------------------------------------------------------------------

// DeleteHomelabMediaServer removes a media server (and its
// now_playing + recent_additions rows via ON DELETE CASCADE FK).
// Idempotent: 200 with {deleted: 0} when the row doesn't exist
// or isn't the caller's.
func DeleteHomelabMediaServer(pool *db.Pool) gin.HandlerFunc {
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
			`DELETE FROM homelab_media_servers
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondOK(c, gin.H{
			"deleted":       tag.RowsAffected(),
			"server_id":     id.String(),
			"idempotent_ok": true,
		})
	}
}

// ------------------------------------------------------------------
