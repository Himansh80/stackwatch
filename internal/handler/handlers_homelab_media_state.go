// Tier 10 Phase 6 — Media Server (H6). The state read endpoint:
//
//	GET /api/v1/homelab/media/state?server_id=X
//
// Returns the latest now-playing + recent-additions rows for the
// caller's media servers. Optional ?server_id filters to one
// server; omit it to return state for every server the caller has
// pinned.
//
// Caps each list at mediaStateNowPlayingCap /
// mediaStateRecentAdditionsCap so a power user with many servers
// can't blow the response size.
//
// MUST be registered BEFORE /media/servers/:id DELETE — Gin's
// radix tree matches in registration order, so /state would
// otherwise be parsed as :id="state" and the static handler would
// never fire. See cmd/api-gateway/routes_homelab.go for the
// registration order.
//
// Split out from handlers_homelab_media.go so the CRUD file
// (List + Create + Delete) stays under the 400-LOC cap. The
// per-server poll helper (PollMediaServerAndInsertState) lives in
// handlers_homelab_media_poll.go so this state read endpoint
// stays focused on the SELECT + JOIN logic.
package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// GetHomelabMediaState returns the latest now-playing + recent
// additions rows for the caller's media servers. Optional
// ?server_id filters to one server; omit it to return state for
// every server the caller has pinned (grouped under each
// server_id).
//
// Caps each list at mediaStateNowPlayingCap /
// mediaStateRecentAdditionsCap so a power user with many servers
// can't blow the response size.
//
// MUST be registered BEFORE /media/servers/:id DELETE — Gin's
// radix tree matches in registration order, so /state would
// otherwise be parsed as :id="state" and the static handler would
// never fire. See cmd/api-gateway/routes_homelab.go for the
// registration order.
func GetHomelabMediaState(pool *db.Pool) gin.HandlerFunc {
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

		var req mediaStateListReq
		if err := c.ShouldBindQuery(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Build WHERE dynamically so server_id is optional.
		// tenant_id + user_id is the non-negotiable base filter.
		// When server_id is provided we ALSO filter on the
		// homelab_media_servers row so a caller can't read another
		// user's state by guessing a server UUID.
		conds := []string{
			"np.tenant_id = $1",
			"np.user_id = $2",
			"s.tenant_id = $1",
			"s.user_id = $2",
		}
		args := []interface{}{tenantID, userID}
		if sid := strings.TrimSpace(req.ServerID); sid != "" {
			if _, perr := uuid.Parse(sid); perr != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "server_id must be a uuid")
				return
			}
			args = append(args, sid)
			conds = append(conds, fmt.Sprintf("np.server_id = $%d", len(args)))
		}

		// Fetch now_playing rows joined to the server row for the
		// chip name/kind. The DESC order on polled_at surfaces
		// the most recent sessions first.
		npQuery := fmt.Sprintf(
			`SELECT np.id::text, np.server_id::text, s.name, s.kind,
			        np.session_id, np.title, COALESCE(np.user_name, ''),
			        COALESCE(np.player, ''), np.transcoding,
			        np.progress_ms, np.duration_ms, np.polled_at::text
			   FROM homelab_now_playing np
			   JOIN homelab_media_servers s ON s.id = np.server_id
			  WHERE %s
			  ORDER BY np.polled_at DESC
			  LIMIT %d`,
			joinStrings(conds, " AND "),
			mediaStateNowPlayingCap,
		)
		npRows, err := pool.Pgx().Query(c.Request.Context(), npQuery, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer npRows.Close()

		npOut := []nowPlayingRow{}
		for npRows.Next() {
			var r nowPlayingRow
			if err := npRows.Scan(&r.ID, &r.ServerID, &r.ServerName, &r.ServerKind,
				&r.SessionID, &r.Title, &r.UserName, &r.Player, &r.Transcoding,
				&r.ProgressMs, &r.DurationMs, &r.PolledAt); err != nil {
				kernel.RespondError(c, err)
				return
			}
			npOut = append(npOut, r)
		}
		if err := npRows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Reuse the same conds/args for recent_additions. Different
		// table alias (ra instead of np) — but server_id, tenant_id,
		// user_id columns are identical across the two tables.
		raConds := []string{
			"ra.tenant_id = $1",
			"ra.user_id = $2",
			"s.tenant_id = $1",
			"s.user_id = $2",
		}
		raArgs := []interface{}{tenantID, userID}
		if req.ServerID != "" {
			raArgs = append(raArgs, req.ServerID)
			raConds = append(raConds, fmt.Sprintf("ra.server_id = $%d", len(raArgs)))
		}
		raQuery := fmt.Sprintf(
			`SELECT ra.id::text, ra.server_id::text, s.name, s.kind,
			        ra.item_id, ra.title, ra.item_kind, ra.year,
			        COALESCE(ra.poster_url, ''), ra.added_at::text, ra.polled_at::text
			   FROM homelab_recent_additions ra
			   JOIN homelab_media_servers s ON s.id = ra.server_id
			  WHERE %s
			  ORDER BY ra.added_at DESC
			  LIMIT %d`,
			joinStrings(raConds, " AND "),
			mediaStateRecentAdditionsCap,
		)
		raRows, err := pool.Pgx().Query(c.Request.Context(), raQuery, raArgs...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer raRows.Close()

		raOut := []recentAdditionRow{}
		for raRows.Next() {
			var r recentAdditionRow
			var year *int
			if err := raRows.Scan(&r.ID, &r.ServerID, &r.ServerName, &r.ServerKind,
				&r.ItemID, &r.Title, &r.ItemKind, &year,
				&r.PosterURL, &r.AddedAt, &r.PolledAt); err != nil {
				kernel.RespondError(c, err)
				return
			}
			r.Year = year
			raOut = append(raOut, r)
		}
		if err := raRows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Always include the matching server rows in the response
		// so the widget can render the server chips without an
		// extra GET. When server_id is set we filter to that
		// single row; when omitted we return the caller's full
		// registry.
		var serverRows []mediaServerRow
		var srvQuery string
		var srvArgs []interface{}
		if req.ServerID != "" {
			srvQuery = `SELECT id::text, tenant_id::text, user_id::text, name,
			                   kind, base_url, enabled,
			                   last_polled_at::text, last_poll_status, last_poll_error,
			                   created_at::text, updated_at::text
			              FROM homelab_media_servers
			             WHERE id = $3 AND tenant_id = $1 AND user_id = $2`
			srvArgs = []interface{}{tenantID, userID, req.ServerID}
		} else {
			srvQuery = `SELECT id::text, tenant_id::text, user_id::text, name,
			                   kind, base_url, enabled,
			                   last_polled_at::text, last_poll_status, last_poll_error,
			                   created_at::text, updated_at::text
			              FROM homelab_media_servers
			             WHERE tenant_id = $1 AND user_id = $2
			             ORDER BY created_at DESC`
			srvArgs = []interface{}{tenantID, userID}
		}
		srvRows, serr := pool.Pgx().Query(c.Request.Context(), srvQuery, srvArgs...)
		if serr != nil {
			kernel.RespondError(c, serr)
			return
		}
		defer srvRows.Close()
		serverRows = []mediaServerRow{}
		for srvRows.Next() {
			var (
				r        mediaServerRow
				lastPol  *string
				lastStat *string
				lastErr  *string
			)
			if err := srvRows.Scan(&r.ID, &r.TenantID, &r.UserID, &r.Name,
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
			serverRows = append(serverRows, r)
		}
		if err := srvRows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}

		resp := mediaStateResp{
			Servers:          serverRows,
			NowPlaying:       npOut,
			RecentAdditions:  raOut,
			CountNowPlaying:  len(npOut),
			CountRecentAdded: len(raOut),
		}
		if req.ServerID != "" {
			resp.ServerID = req.ServerID
		}
		kernel.RespondOK(c, resp)
	}
}
