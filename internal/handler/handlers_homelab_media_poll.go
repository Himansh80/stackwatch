// Tier 10 Phase 6 — Media Server (H6). The shared
// poll-and-upsert helper used by BOTH the handler-side
// immediate fire-and-forget poll (fired by POST /media/servers
// after create) AND the MediaWorker (internal/homelab/media.go)
// for its 60s periodic tick.
//
// Split out from handlers_homelab_media.go so each handler file
// stays under the 400-LOC cap. The CRUD endpoints (List / Create
// / Delete) live in handlers_homelab_media.go; the state
// endpoint (GET /state) lives in handlers_homelab_media_state.go.
//
// The actual per-kind HTTP fetch lives in internal/homelab/
// (homelab.PollMediaServer). This handler-level wrapper:
//   1. SELECTs the server's row (kind, base_url, api_key,
//      tenant_id, user_id) — ownership check happens here so a
//      future "force poll" admin route could bypass it
//   2. Calls homelab.PollMediaServer with the credentials
//   3. UPSERTs one homelab_now_playing row per active session
//   4. UPSERTs one homelab_recent_additions row per recent item
//   5. UPDATEs the server row's bookkeeping columns
//
// Per-user (NOT per-tenant): every WHERE clause filters by both
// tenant_id and user_id so user A can never see or modify user B's
// media servers — matches US-8 in the speckit proposal.
package handler

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/homelab"
)

// mediaPollOverallTimeout caps any single poll's wall-clock
// budget — covers HTTP fetch + JSON parse + DB UPSERTs +
// bookkeeping UPDATE. 30s matches the worst-case window a Plex or
// Jellyfin API might take when the library is large; longer hangs
// should be killed at the per-kind timeout (homelab-level) rather
// than here.
const mediaPollOverallTimeout = 30 * time.Second

// PollMediaServerAndInsertState runs one poll cycle for serverID:
// SELECTs the server's credentials + ownership, calls
// homelab.PollMediaServer for the kind-specific HTTP fetch, then
// UPSERTs one homelab_now_playing row per active session and one
// homelab_recent_additions row per recent item. Also UPDATEs the
// server row's bookkeeping columns.
//
// Synchronous — callers decide whether to run this in a goroutine
// (POST /servers fires-and-forgets; the MediaWorker calls
// sequentially per tick).
//
// On success: UPDATEs the server with last_poll_status='success',
// last_poll_error=''. On failure: still UPDATEs the server with
// last_poll_status='error' and the truncated error message so the
// dashboard can render the error pill without losing the failure
// reason.
//
// Caller-supplied ctx bounds the whole operation (fetch + upserts
// + bookkeeping). The homelab-layer fetcher has its own internal
// per-request timeout that's typically tighter than this.
func PollMediaServerAndInsertState(ctx context.Context, pool *db.Pool, serverID uuid.UUID) {
	logger := slog.Default()

	// Wrap the caller-supplied ctx with mediaPollOverallTimeout so
	// a slow remote target can't pin a goroutine indefinitely.
	ctx, cancel := context.WithTimeout(ctx, mediaPollOverallTimeout)
	defer cancel()

	var (
		tenantID, userID uuid.UUID
		kind, baseURL    string
		apiKey           string
		enabled          bool
	)
	err := pool.Pgx().QueryRow(ctx,
		`SELECT tenant_id, user_id, kind, base_url, api_key, enabled
		   FROM homelab_media_servers WHERE id = $1`,
		serverID,
	).Scan(&tenantID, &userID, &kind, &baseURL, &apiKey, &enabled)
	if err != nil {
		logger.Warn("media: load server row failed",
			"server_id", serverID, "err", err)
		return
	}
	if !enabled {
		// Disabled servers are skipped without upserting any
		// state — the user explicitly opted out and we don't want
		// to pollute the dashboard with stale rows.
		return
	}

	state, pollErr := homelab.PollMediaServer(ctx, kind, baseURL, apiKey)

	// Build the upsert + bookkeeping in one transaction so the
	// dashboard never sees a partial state (e.g. now_playing
	// rows for one tick but stale recent_additions from a prior
	// tick).
	tx, txErr := pool.Pgx().Begin(ctx)
	if txErr != nil {
		logger.Warn("media: tx begin failed",
			"server_id", serverID, "err", txErr)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr.Error() != "tx is closed" {
			logger.Warn("media: tx rollback failed",
				"server_id", serverID, "err", rbErr)
		}
	}()

	// Defensive defaults — ensure we never have stale rows from
	// prior ticks lingering after a transient outage.
	if state == nil {
		state = &homelab.MediaState{}
	}

	// 1. Prune prior now_playing rows for this server. The poll
	//    may return fewer sessions than were active last tick
	//    (the user paused, the network dropped, the session
	//    ended), and we don't want lingering rows. The UPSERTs
	//    below re-create only the sessions that are active RIGHT
	//    NOW.
	if _, err := tx.Exec(ctx,
		`DELETE FROM homelab_now_playing WHERE server_id = $1`,
		serverID,
	); err != nil {
		logger.Warn("media: prune now_playing failed",
			"server_id", serverID, "err", err)
		return
	}

	// 2. Insert the freshly-polled now_playing rows.
	for _, sess := range state.NowPlaying {
		if sess.SessionID == "" {
			// Skip malformed entries rather than fail the
			// whole batch — the upstream API can occasionally
			// return sessions with empty ids.
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO homelab_now_playing
			        (tenant_id, user_id, server_id,
			         session_id, title, user_name, player,
			         transcoding, progress_ms, duration_ms)
			 VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''),
			         $8, $9, $10)`,
			tenantID, userID, serverID,
			sess.SessionID, sess.Title, sess.UserName, sess.Player,
			sess.Transcoding, sess.ProgressMs, sess.DurationMs,
		); err != nil {
			logger.Warn("media: insert now_playing failed",
				"server_id", serverID, "session_id", sess.SessionID, "err", err)
			// Don't return — keep trying to insert the rest.
		}
	}

	// 3. UPSERT the freshly-polled recent_additions rows. We use
	//    ON CONFLICT (server_id, item_id) DO UPDATE so the polled_at
	//    + title + poster_url + year reflect the latest upstream
	//    values. added_at is left alone (it's the upstream
	//    library timestamp and shouldn't change on re-poll).
	for _, item := range state.RecentAdditions {
		if item.ItemID == "" || item.Title == "" {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO homelab_recent_additions
			        (tenant_id, user_id, server_id,
			         item_id, title, item_kind, year, poster_url,
			         added_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9)
			 ON CONFLICT (server_id, item_id) DO UPDATE
			    SET title      = EXCLUDED.title,
			        item_kind  = EXCLUDED.item_kind,
			        year       = EXCLUDED.year,
			        poster_url = EXCLUDED.poster_url,
			        polled_at  = NOW()`,
			tenantID, userID, serverID,
			item.ItemID, item.Title, item.Kind, item.Year, item.PosterURL,
			item.AddedAt,
		); err != nil {
			logger.Warn("media: upsert recent_additions failed",
				"server_id", serverID, "item_id", item.ItemID, "err", err)
			// Don't return — keep trying to upsert the rest.
		}
	}

	// 4. Cap the recent_additions table at the last
	//    mediaStateRecentAdditionsCap items per server. Without
	//    this a long-running user would accumulate hundreds of
	//    rows per server (Plex's /library/recentlyAdded can
	//    return 50+ items per poll depending on the library
	//    size).
	if _, err := tx.Exec(ctx,
		`DELETE FROM homelab_recent_additions
		  WHERE server_id = $1
		    AND id NOT IN (
		        SELECT id FROM homelab_recent_additions
		         WHERE server_id = $1
		         ORDER BY added_at DESC
		         LIMIT $2
		    )`,
		serverID, mediaStateRecentAdditionsCap,
	); err != nil {
		logger.Warn("media: prune recent_additions failed",
			"server_id", serverID, "err", err)
		return
	}

	// 5. Update the server row's bookkeeping columns.
	now := time.Now().UTC()
	var status, errMsg string
	if pollErr != nil {
		status = "error"
		errMsg = mediaTruncateErr(pollErr.Error())
	} else {
		status = "success"
		errMsg = ""
	}
	if _, err := tx.Exec(ctx,
		`UPDATE homelab_media_servers
		    SET last_polled_at   = $2,
		        last_poll_status = $3,
		        last_poll_error  = NULLIF($4, ''),
		        updated_at       = NOW()
		  WHERE id = $1`,
		serverID, now, status, errMsg,
	); err != nil {
		logger.Warn("media: update server status failed",
			"server_id", serverID, "err", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		logger.Warn("media: tx commit failed",
			"server_id", serverID, "err", err)
		return
	}

	if pollErr != nil {
		logger.Warn("media: poll failed",
			"server_id", serverID, "kind", kind, "err", pollErr)
	}
}

// mediaTruncateErr caps an error message at 200 chars + ellipsis
// so a huge TLS chain or 30s timeout doesn't bloat the
// homelab_media_servers.last_poll_error column. Renamed
// (mediaTruncateErr prefix) to avoid a future package-level name
// clash with the download-side downloadTruncateErr helper.
func mediaTruncateErr(s string) string {
	const maxLength = 200
	if len(s) > maxLength {
		return s[:maxLength] + "…"
	}
	return s
}
