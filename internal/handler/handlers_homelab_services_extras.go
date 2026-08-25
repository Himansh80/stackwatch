// Tier 10 Phase 2 — Service Status (H2). The remaining 3 service
// handlers (PATCH, DELETE, aggregate) split out of
// handlers_homelab_services.go to keep both files under 400 LOC.
//
//	PATCH  /api/v1/homelab/services/:id      — PatchHomelabService
//	DELETE /api/v1/homelab/services/:id      — DeleteHomelabService
//	GET    /api/v1/homelab/services/aggregate — GetHomelabServicesAggregate
//
// All three honor tenant_id + user_id from the JWT — never touches
// another user's pins.
package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// PATCH /api/v1/homelab/services/:id
// ------------------------------------------------------------------

// PatchHomelabService updates a pin. All fields optional — the
// handler builds a dynamic UPDATE with only the supplied fields,
// matching the dashboards PATCH pattern.
//
// 404 when the row does not belong to the caller (or doesn't exist).
func PatchHomelabService(pool *db.Pool) gin.HandlerFunc {
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

		var req pinnedServicePatchReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		sets := []string{}
		args := []interface{}{}
		if req.Name != nil {
			nm := strings.TrimSpace(*req.Name)
			if nm == "" {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "name cannot be empty")
				return
			}
			args = append(args, nm)
			sets = append(sets, fmt.Sprintf("name = $%d", len(args)))
		}
		if req.URL != nil {
			// We need the kind to validate URL shape. Pull the
			// current kind from the row first when URL is being
			// changed — single SELECT, cheap.
			var curKind string
			if err := pool.Pgx().QueryRow(c.Request.Context(),
				`SELECT kind FROM homelab_pinned_services
				  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
				id, tenantID, userID,
			).Scan(&curKind); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					kernel.RespondError(c, kernel.ErrNotFound)
					return
				}
				kernel.RespondError(c, err)
				return
			}
			newKind := curKind
			if req.Kind != nil {
				newKind = *req.Kind
			}
			if err := validateServiceURL(*req.URL, newKind); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			args = append(args, *req.URL)
			sets = append(sets, fmt.Sprintf("url = $%d", len(args)))
		}
		if req.Kind != nil {
			if !allowedServiceKinds[*req.Kind] {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
					"kind must be one of: http, https, tcp, icmp")
				return
			}
			args = append(args, *req.Kind)
			sets = append(sets, fmt.Sprintf("kind = $%d", len(args)))
		}
		if req.Icon != nil {
			args = append(args, *req.Icon)
			sets = append(sets, fmt.Sprintf("icon = $%d", len(args)))
		}
		if req.Enabled != nil {
			args = append(args, *req.Enabled)
			sets = append(sets, fmt.Sprintf("enabled = $%d", len(args)))
		}
		if len(sets) == 0 {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "no fields to update")
			return
		}
		sets = append(sets, "updated_at = NOW()")
		args = append(args, id, tenantID, userID)
		q := "UPDATE homelab_pinned_services SET " + joinStrings(sets, ", ") +
			" WHERE id = $" + fmt.Sprint(len(args)-2) +
			" AND tenant_id = $" + fmt.Sprint(len(args)-1) +
			" AND user_id = $" + fmt.Sprint(len(args))

		tag, err := pool.Pgx().Exec(c.Request.Context(), q, args...)
		if err != nil {
			if strings.Contains(err.Error(), "23505") {
				kernel.RespondErrorWithCode(c, http.StatusConflict, "conflict",
					"a service with this name is already pinned")
				return
			}
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}

		// Return the refreshed row so the UI doesn't have to refetch.
		var (
			rowID, tid, uid, name, urlStr, kindStr, createdAt, updatedAt string
			icon                                                        *string
			enabled                                                     bool
		)
		if err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id::text, tenant_id::text, user_id::text, name, url, kind,
			        icon, enabled, created_at::text, updated_at::text
			   FROM homelab_pinned_services
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		).Scan(&rowID, &tid, &uid, &name, &urlStr, &kindStr, &icon, &enabled, &createdAt, &updatedAt); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, pinnedServiceRow{
			ID: rowID, TenantID: tid, UserID: uid,
			Name: name, URL: urlStr, Kind: kindStr,
			Icon: icon, Enabled: enabled,
			CreatedAt: createdAt, UpdatedAt: updatedAt,
		})
	}
}

// ------------------------------------------------------------------
// DELETE /api/v1/homelab/services/:id
// ------------------------------------------------------------------

// DeleteHomelabService unpins a service for the caller. Idempotent:
// 200 with `{deleted: 0}` when the row doesn't exist (or isn't the
// caller's) — the UI treats "it doesn't exist" and "I deleted it"
// as the same outcome.
//
// We also best-effort delete the corresponding homelab_service_health
// rows so the table doesn't accumulate orphan history rows. The
// worker prunes by tenant_id every tick so this is a hygiene step,
// not a correctness requirement.
func DeleteHomelabService(pool *db.Pool) gin.HandlerFunc {
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
			`DELETE FROM homelab_pinned_services
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Best-effort cleanup of stale health rows. Failure is
		// non-fatal — the worker prunes by tenant_id every tick.
		_, _ = pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM homelab_service_health
			  WHERE service_id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		)

		kernel.RespondOK(c, gin.H{
			"deleted":       tag.RowsAffected(),
			"service_id":    id.String(),
			"idempotent_ok": true,
		})
	}
}

// ------------------------------------------------------------------
// GET /api/v1/homelab/services/aggregate
// ------------------------------------------------------------------

// GetHomelabServicesAggregate returns a {up, degraded, down, unknown,
// total} summary for the caller's pinned services, computed from
// the LATEST health row per service. Powers the KPI strip +
// dashboard hero numbers.
//
// Implementation: a single LATERAL join per pinned service —
// LIMIT 1 in the lateral picks the newest row in O(1) thanks to
// the (service_id, checked_at DESC) index. Cheaper than fetching
// all rows and bucketing in Go.
func GetHomelabServicesAggregate(pool *db.Pool) gin.HandlerFunc {
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

		// Bucketing in one round-trip. LEFT JOIN LATERAL so a
		// freshly-pinned service with no probes yet counts as
		// 'unknown' rather than being skipped.
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT
			       ps.id::text AS service_id,
			       COALESCE(h.status, 'unknown') AS status
			   FROM homelab_pinned_services ps
			   LEFT JOIN LATERAL (
			       SELECT status FROM homelab_service_health
			        WHERE service_id = ps.id
			        ORDER BY checked_at DESC
			        LIMIT 1
			   ) h ON true
			  WHERE ps.tenant_id = $1 AND ps.user_id = $2`,
			tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		var up, degraded, down, unknown int
		for rows.Next() {
			var sid, status string
			if err := rows.Scan(&sid, &status); err != nil {
				kernel.RespondError(c, err)
				return
			}
			switch status {
			case "up":
				up++
			case "degraded":
				degraded++
			case "down":
				down++
			default:
				unknown++
			}
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		total := up + degraded + down + unknown

		kernel.RespondOK(c, gin.H{
			"up":       up,
			"degraded": degraded,
			"down":     down,
			"unknown":  unknown,
			"total":    total,
		})
	}
}