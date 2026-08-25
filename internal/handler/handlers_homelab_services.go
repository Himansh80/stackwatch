// Tier 10 Phase 2 — Service Status (H2). CRUD endpoints for the
// per-user pinboard (homelab_pinned_services) plus the aggregate
// health summary used by the KPI strip and the service tile grid.
//
//	GET    /api/v1/homelab/services          — ListHomelabServices
//	POST   /api/v1/homelab/services          — CreateHomelabService
//
// PatchHomelabService / DeleteHomelabService /
// GetHomelabServicesAggregate live in handlers_homelab_services_extras.go
// (same Phase 2, split for file-size discipline). Probe endpoints
// (POST /:id/probe, GET /:id/history, POST /probe-all) live in
// handlers_homelab_services_probe.go.
//
// Storage: homelab_pinned_services (CRUD) + homelab_service_health
// (latest-per-service join for the GET /aggregate response).
//
// Per-user (NOT per-tenant): every WHERE clause filters by both
// tenant_id and user_id so user A can never see user B's pins.
package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// Validation helpers — kept small so the file stays under 400 LOC.
// ------------------------------------------------------------------

// validateServiceURL does a soft check on the URL the user pinned:
// must be parseable AND must carry a scheme we know how to probe
// (http, https, tcp, icmp). We don't reach out here — that's what
// the probe endpoints are for.
func validateServiceURL(raw, kind string) error {
	if raw == "" {
		return fmt.Errorf("url is required")
	}
	switch kind {
	case "http", "https":
		u, err := url.Parse(raw)
		if err != nil {
			return fmt.Errorf("url is not parseable: %w", err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("http/https urls must start with http:// or https://")
		}
		if u.Host == "" {
			return fmt.Errorf("url is missing host")
		}
	case "tcp":
		// Expect host:port. We don't validate port range strictly
		// (1..65535) — url.Parse handles whitespace and gives us
		// a host:port pair to feed into net.Dial.
		if _, _, err := splitHostPort(raw); err != nil {
			return fmt.Errorf("tcp urls must be host:port (e.g. 10.0.0.5:22)")
		}
	case "icmp":
		// Bare hostname or IP — ICMP doesn't carry a port.
		if strings.ContainsAny(raw, " \t\n/") {
			return fmt.Errorf("icmp targets must be a hostname or IP, no scheme")
		}
	default:
		return fmt.Errorf("kind %q is not supported", kind)
	}
	return nil
}

// splitHostPort is a thin wrapper around net.SplitHostPort so the
// imports list stays tight (we already need `net/url` for http).
func splitHostPort(addr string) (host, port string, err error) {
	// Use net.SplitHostPort inline so we don't add another import.
	colon := strings.LastIndex(addr, ":")
	if colon < 0 {
		return "", "", fmt.Errorf("missing :port")
	}
	return addr[:colon], addr[colon+1:], nil
}

// ------------------------------------------------------------------
// Internal helper — read the latest health row for one service.
//
// Latency budget: O(rows-for-this-service) thanks to the
// (service_id, checked_at DESC) index. We use a single SELECT
// DISTINCT ON query rather than a window function for clarity.
// ------------------------------------------------------------------

// loadLatestHealthPerService returns a map[serviceID]pinnedHealthBlock
// of the most-recent health row per service in the supplied slice.
// Used by ListHomelabServices to enrich the response with status
// pills (no extra round-trip per service from the frontend).
//
// Returns an empty map if no health rows exist yet (first-run UX
// is "unknown" pills — the worker fills them in within 60s).
func loadLatestHealthPerService(ctx context.Context, pool *db.Pool, tenantID, userID uuid.UUID, serviceIDs []uuid.UUID) (map[string]pinnedHealthBlock, error) {
	out := map[string]pinnedHealthBlock{}
	if len(serviceIDs) == 0 {
		return out, nil
	}
	// Build the IN list ($1 = tenantID, $2 = userID, $3..$N = ids).
	args := []interface{}{tenantID, userID}
	for _, sid := range serviceIDs {
		args = append(args, sid)
	}
	// DISTINCT ON (service_id) gives us the newest row per service
	// in one query. Ordered by (service_id, checked_at DESC) so the
	// planner uses the (service_id, checked_at DESC) index directly.
	placeholders := "$3"
	for i := 4; i < 3+len(serviceIDs); i++ {
		placeholders += fmt.Sprintf(",$%d", i)
	}
	query := fmt.Sprintf(`
		SELECT DISTINCT ON (service_id)
		       service_id::text, status, latency_ms, status_code, error_message, checked_at::text
		  FROM homelab_service_health
		 WHERE tenant_id = $1 AND user_id = $2
		   AND service_id IN (%s)
		 ORDER BY service_id, checked_at DESC`, placeholders)

	rows, err := pool.Pgx().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			sid, status, checkedAt string
			latencyMS              *int
			statusCode             *int
			errorMsg               *string
		)
		if err := rows.Scan(&sid, &status, &latencyMS, &statusCode, &errorMsg, &checkedAt); err != nil {
			return nil, err
		}
		out[sid] = pinnedHealthBlock{
			Status:       status,
			LatencyMS:    latencyMS,
			StatusCode:   statusCode,
			ErrorMessage: errorMsg,
			CheckedAt:    checkedAt,
		}
	}
	return out, rows.Err()
}

// ------------------------------------------------------------------
// GET /api/v1/homelab/services
// ------------------------------------------------------------------

// ListHomelabServices returns all pinned services for the caller,
// joined with the latest health snapshot (if any) so the UI can
// render status pills in a single round-trip.
//
// Honors tenant_id + user_id from the JWT — never returns another
// user's pins even if a row id leaks through.
func ListHomelabServices(pool *db.Pool) gin.HandlerFunc {
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
			`SELECT id::text, tenant_id::text, user_id::text, name, url, kind,
			        icon, enabled, created_at::text, updated_at::text
			   FROM homelab_pinned_services
			  WHERE tenant_id = $1 AND user_id = $2
			  ORDER BY created_at ASC`,
			tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		pinRows := []pinnedServiceRow{}
		serviceIDs := []uuid.UUID{}
		for rows.Next() {
			var (
				id, tid, uid, name, url, kind, createdAt, updatedAt string
				icon                                                 *string
				enabled                                              bool
			)
			if err := rows.Scan(&id, &tid, &uid, &name, &url, &kind, &icon, &enabled, &createdAt, &updatedAt); err != nil {
				kernel.RespondError(c, err)
				return
			}
			parsedID, _ := uuid.Parse(id)
			pinRows = append(pinRows, pinnedServiceRow{
				ID: id, TenantID: tid, UserID: uid,
				Name: name, URL: url, Kind: kind,
				Icon: icon, Enabled: enabled,
				CreatedAt: createdAt, UpdatedAt: updatedAt,
			})
			serviceIDs = append(serviceIDs, parsedID)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Enrich with the latest health row per service (single
		// batched SELECT, no N+1).
		healthByService, herr := loadLatestHealthPerService(c.Request.Context(), pool, tenantID, userID, serviceIDs)
		if herr != nil {
			kernel.RespondError(c, herr)
			return
		}
		for i := range pinRows {
			if h, ok := healthByService[pinRows[i].ID]; ok {
				pinRows[i].Health = &h
			}
		}

		kernel.RespondOK(c, gin.H{
			"services": pinRows,
			"count":    len(pinRows),
		})
	}
}

// ------------------------------------------------------------------
// POST /api/v1/homelab/services
// ------------------------------------------------------------------

// CreateHomelabService pins a new service for the caller. Validates:
//
//   - all required fields present
//   - kind is in allowedServiceKinds
//   - url parses and matches the kind (http → http(s) scheme, etc.)
//   - caller has not exceeded maxPinnedServicesPerUser
//
// Returns 409 on duplicate (tenant_id, user_id, name) — same user
// pinning the same name twice is rejected so the dashboard never
// shows duplicate tiles. Re-pinning under a different name is fine.
func CreateHomelabService(pool *db.Pool) gin.HandlerFunc {
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

		var req pinnedServiceReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "name is required")
			return
		}
		if !allowedServiceKinds[req.Kind] {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"kind must be one of: http, https, tcp, icmp")
			return
		}
		if err := validateServiceURL(req.URL, req.Kind); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}

		// Per-user cap — count before INSERT. Cheap query on the
		// (user_id) index.
		var currentCount int
		if err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT COUNT(*) FROM homelab_pinned_services
			  WHERE tenant_id = $1 AND user_id = $2`,
			tenantID, userID,
		).Scan(&currentCount); err != nil {
			kernel.RespondError(c, err)
			return
		}
		if currentCount >= maxPinnedServicesPerUser {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "limit_reached",
				fmt.Sprintf("pin cap of %d reached; unpin a service before adding another", maxPinnedServicesPerUser))
			return
		}

		var (
			id, createdAt, updatedAt string
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO homelab_pinned_services
			        (tenant_id, user_id, name, url, kind, icon)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id::text, created_at::text, updated_at::text`,
			tenantID, userID, req.Name, req.URL, req.Kind, req.Icon,
		).Scan(&id, &createdAt, &updatedAt)
		if err != nil {
			// 23505 = unique_violation (duplicate name for this user)
			if strings.Contains(err.Error(), "23505") {
				kernel.RespondErrorWithCode(c, http.StatusConflict, "conflict",
					"a service with this name is already pinned")
				return
			}
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondCreated(c, pinnedServiceRow{
			ID:        id,
			TenantID:  tenantID.String(),
			UserID:    userID.String(),
			Name:      req.Name,
			URL:       req.URL,
			Kind:      req.Kind,
			Icon:      req.Icon,
			Enabled:   true,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
}