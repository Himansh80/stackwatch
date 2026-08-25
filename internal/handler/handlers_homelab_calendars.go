// Tier 10 Phase 4 — Calendar (H4). 5 protected endpoints back
// the per-user iCal subscription surface (homelab_calendars +
// homelab_events).
//
//	GET    /api/v1/homelab/calendars              — ListHomelabCalendars
//	POST   /api/v1/homelab/calendars              — CreateHomelabCalendar
//	POST   /api/v1/homelab/calendars/:id/refresh  — RefreshHomelabCalendar
//	DELETE /api/v1/homelab/calendars/:id          — DeleteHomelabCalendar
//
// The events endpoint (GET /calendars/events) lives in
// handlers_homelab_calendars_events.go (split for file-size
// discipline — keeps each file under 400 LOC).
//
// The CalendarWorker (internal/homelab/calendar.go) ticks every
// 30min and upserts events; POST /calendars fires an immediate
// sync after create so a freshly-pinned feed shows events within
// seconds rather than waiting a full tick. POST /:id/refresh
// blocks until the sync finishes and returns the new
// last_synced_at so the frontend can refresh the chip color.
//
// Per-user (NOT per-tenant): every WHERE clause filters by both
// tenant_id and user_id so user A can never see or modify user
// B's calendars — matches US-8 in the speckit proposal.
package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/homelab"
	"github.com/stackwatch/platform/internal/kernel"
)

// validateCalendarURL rejects anything that isn't http/https.
// Same check the worker relies on, so we surface the error at
// insert time rather than at first sync.
func validateCalendarURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("ical_url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("ical_url is not parseable: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("ical_url must be http:// or https://")
	}
	if u.Host == "" {
		return fmt.Errorf("ical_url is missing host")
	}
	return nil
}

// validateCalendarName enforces a sane title. Empty / 200+ chars
// both rejected — the dashboard's calendar chip uses the name as
// its label.
func validateCalendarName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > 200 {
		return fmt.Errorf("name must be 200 characters or fewer")
	}
	return nil
}

// normalizeCalendarColor returns the color the handler will
// INSERT. Falls back to defaultCalendarColor when the input is
// empty or doesn't match the allowed palette.
func normalizeCalendarColor(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultCalendarColor
	}
	if _, ok := allowedCalendarColors[raw]; ok {
		return raw
	}
	return defaultCalendarColor
}

// parseCalendarColorPicker accepts any of the allowed hex codes
// plus a case-insensitive "default" sentinel that maps to
// defaultCalendarColor. The frontend's color picker emits these
// exact values.
func parseCalendarColorPicker(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.EqualFold(raw, "default") || raw == "" {
		return defaultCalendarColor
	}
	return normalizeCalendarColor(raw)
}

// scanCalendarRow scans one row of the standard calendars SELECT
// into a calendarRow. Helper shared by List + Create so the
// Scan signature stays in one place.
func scanCalendarRow(scan func(...any) error) (calendarRow, error) {
	var (
		row        calendarRow
		lastSynced *string
		lastStatus *string
		lastError  *string
		eventCount int
		createdAt  string
		updatedAt  string
	)
	if err := scan(&row.ID, &row.TenantID, &row.UserID, &row.Name,
		&row.ICalURL, &row.Color, &row.Enabled,
		&lastSynced, &lastStatus, &lastError,
		&eventCount, &createdAt, &updatedAt); err != nil {
		return calendarRow{}, err
	}
	if lastSynced != nil {
		row.LastSyncedAt = *lastSynced
	}
	if lastStatus != nil {
		row.LastSyncStatus = *lastStatus
	}
	if lastError != nil {
		row.LastSyncError = *lastError
	}
	row.EventCount = eventCount
	row.CreatedAt = createdAt
	row.UpdatedAt = updatedAt
	return row, nil
}

// ------------------------------------------------------------------
// GET /api/v1/homelab/calendars
// ------------------------------------------------------------------

// ListHomelabCalendars returns every calendar the caller has
// pinned. Each row carries event_count (computed via a subquery)
// and the worker bookkeeping fields so the frontend can render
// the chip + status pill without a follow-up call.
func ListHomelabCalendars(pool *db.Pool) gin.HandlerFunc {
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
			        c.ical_url, c.color, c.enabled,
			        c.last_synced_at::text, c.last_sync_status, c.last_sync_error,
			        COALESCE(e.cnt, 0),
			        c.created_at::text, c.updated_at::text
			   FROM homelab_calendars c
			   LEFT JOIN (
			       SELECT calendar_id, COUNT(*)::int AS cnt
			         FROM homelab_events
			        GROUP BY calendar_id
			   ) e ON e.calendar_id = c.id
			  WHERE c.tenant_id = $1 AND c.user_id = $2
			  ORDER BY c.created_at DESC`,
			tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := []calendarRow{}
		for rows.Next() {
			r, serr := scanCalendarRow(rows.Scan)
			if serr != nil {
				kernel.RespondError(c, serr)
				return
			}
			out = append(out, r)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"calendars": out,
			"count":     len(out),
		})
	}
}

// ------------------------------------------------------------------
// POST /api/v1/homelab/calendars
// ------------------------------------------------------------------

// CreateHomelabCalendar inserts a new calendar for the caller.
// Fires an immediate sync after the INSERT (fire-and-forget) so
// a freshly-pinned feed shows events within seconds.
//
// Validates name (non-empty, ≤ 200 chars), ical_url (http/https
// parseable), and color (allowed palette or default).
// Returns 201 with the inserted row.
func CreateHomelabCalendar(pool *db.Pool) gin.HandlerFunc {
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

		var req calendarReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if err := validateCalendarName(req.Name); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		req.ICalURL = strings.TrimSpace(req.ICalURL)
		if err := validateCalendarURL(req.ICalURL); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		color := parseCalendarColorPicker(req.Color)
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}

		var (
			id, createdAt, updatedAt string
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO homelab_calendars
			        (tenant_id, user_id, name, ical_url, color, enabled)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id::text, created_at::text, updated_at::text`,
			tenantID, userID, req.Name, req.ICalURL, color, enabled,
		).Scan(&id, &createdAt, &updatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Fire-and-forget immediate sync so events show within
		// seconds rather than waiting a full worker tick. The
		// goroutine recovers from any panic in homelab package.
		newID, perr := uuid.Parse(id)
		if perr == nil {
			homelab.SyncCalendarByID(pool, newID, slog.Default())
		}

		kernel.RespondCreated(c, calendarRow{
			ID:        id,
			TenantID:  tenantID.String(),
			UserID:    userID.String(),
			Name:      req.Name,
			ICalURL:   req.ICalURL,
			Color:     color,
			Enabled:   enabled,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
}

// ------------------------------------------------------------------
// POST /api/v1/homelab/calendars/:id/refresh
// ------------------------------------------------------------------

// RefreshHomelabCalendar fires a synchronous sync for one
// calendar and returns the new bookkeeping state. Blocks for up
// to 60s — long enough for a slow iCal feed, short enough to
// feel responsive.
//
// MUST be registered BEFORE /calendars/:id DELETE.
func RefreshHomelabCalendar(pool *db.Pool) gin.HandlerFunc {
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

		// Verify the row belongs to the caller BEFORE running
		// the sync — prevents the worker from doing a fetch
		// for another user's calendar.
		var ownerCheck int
		if err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT 1 FROM homelab_calendars WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		).Scan(&ownerCheck); err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}

		syncedAt, status, syncErr := homelab.SyncCalendarByID(pool, id, slog.Default())
		if syncErr != nil {
			// Sync failed — return 200 with the error in the
			// body so the frontend can surface the error
			// message but still update the UI on the new
			// status (last_sync_status='error').
			kernel.RespondOK(c, gin.H{
				"calendar_id":      id.String(),
				"last_synced_at":   syncedAt.Format(time.RFC3339),
				"last_sync_status": status,
				"sync_ok":          false,
				"error":            syncErr.Error(),
			})
			return
		}
		kernel.RespondOK(c, gin.H{
			"calendar_id":      id.String(),
			"last_synced_at":   syncedAt.Format(time.RFC3339),
			"last_sync_status": status,
			"sync_ok":          true,
		})
	}
}

// ------------------------------------------------------------------
// DELETE /api/v1/homelab/calendars/:id
// ------------------------------------------------------------------

// DeleteHomelabCalendar removes a calendar (and its events via
// ON DELETE CASCADE FK). Idempotent: 200 with {deleted: 0} when
// the row doesn't exist or isn't the caller's.
func DeleteHomelabCalendar(pool *db.Pool) gin.HandlerFunc {
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
			`DELETE FROM homelab_calendars
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondOK(c, gin.H{
			"deleted":       tag.RowsAffected(),
			"calendar_id":   id.String(),
			"idempotent_ok": true,
		})
	}
}
