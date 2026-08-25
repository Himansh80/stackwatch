// Tier 10 Phase 4 — Calendar (H4). The events endpoint — split
// out of handlers_homelab_calendars.go to keep both files under
// 400 LOC.
//
//	GET /api/v1/homelab/calendars/events  — ListHomelabCalendarEvents
//
// MUST be registered BEFORE /calendars/:id/* routes — Gin's
// radix tree matches in registration order, so "/events" would
// otherwise be parsed as :id="events" and the static-path
// handler would never fire.
package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// parseTimeBound parses ?from/?to as either RFC3339 or
// datetime-local. Returns the supplied default on empty input;
// returns an error on parse failure so the caller can surface a
// 400.
func parseTimeBound(raw string, def time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse(timeParseLayout, raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("must be RFC3339 or YYYY-MM-DDTHH:MM")
}

// ListHomelabCalendarEvents returns events in [from, to] for the
// caller. Defaults: now → now + 30d (matches the spec's
// "upcoming events next 30d"). Optional ?calendar_id filters to
// one calendar; ?limit caps the response.
func ListHomelabCalendarEvents(pool *db.Pool) gin.HandlerFunc {
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

		var req eventsListReq
		if err := c.ShouldBindQuery(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		now := time.Now().UTC()
		defaultTo := now.Add(time.Duration(eventsDefaultWindowDays) * 24 * time.Hour)
		from, ferr := parseTimeBound(req.From, now)
		if ferr != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "from: "+ferr.Error())
			return
		}
		to, terr := parseTimeBound(req.To, defaultTo)
		if terr != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "to: "+terr.Error())
			return
		}
		if !to.After(from) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "to must be after from")
			return
		}

		limit := eventsListLimit
		if req.Limit > 0 {
			if req.Limit > eventsListLimitMax {
				req.Limit = eventsListLimitMax
			}
			limit = req.Limit
		}

		// Build the WHERE clause dynamically so calendar_id is
		// optional. Base filter on tenant_id + user_id is
		// non-negotiable.
		conds := []string{"tenant_id = $1", "user_id = $2", "starts_at >= $3", "starts_at <= $4"}
		args := []interface{}{tenantID, userID, from, to}
		if calID := strings.TrimSpace(req.CalendarID); calID != "" {
			if _, perr := uuid.Parse(calID); perr != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "calendar_id must be a uuid")
				return
			}
			args = append(args, calID)
			conds = append(conds, fmt.Sprintf("calendar_id = $%d", len(args)))
		}
		args = append(args, limit)
		q := `SELECT id::text, tenant_id::text, user_id::text, calendar_id::text,
		             uid, summary, description, location,
		             starts_at::text, ends_at::text, all_day,
		             created_at::text, updated_at::text
		        FROM homelab_events
		       WHERE ` + joinStrings(conds, " AND ") +
			fmt.Sprintf(" ORDER BY starts_at ASC LIMIT $%d", len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := []eventRow{}
		for rows.Next() {
			var (
				row                          eventRow
				endsAt, description, location *string
				createdAt, updatedAt         string
			)
			if err := rows.Scan(&row.ID, &row.TenantID, &row.UserID, &row.CalendarID,
				&row.UID, &row.Summary, &description, &location,
				&row.StartsAt, &endsAt, &row.AllDay,
				&createdAt, &updatedAt); err != nil {
				kernel.RespondError(c, err)
				return
			}
			if description != nil {
				row.Description = *description
			}
			if location != nil {
				row.Location = *location
			}
			if endsAt != nil {
				row.EndsAt = *endsAt
			}
			row.CreatedAt = createdAt
			row.UpdatedAt = updatedAt
			out = append(out, row)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"events": out,
			"count":  len(out),
			"from":   from.Format(time.RFC3339),
			"to":     to.Format(time.RFC3339),
			"limit":  limit,
		})
	}
}
