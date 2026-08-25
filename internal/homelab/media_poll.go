// Package homelab — MediaWorker's per-server poll-and-upsert
// helper.
//
// Split out from media.go so the worker file (the ticker + the
// per-process singleton + the per-tick loop) stays under the
// 400-LOC cap. mediaPollAndUpsert is the worker-side mirror of
// handler.PollMediaServerAndInsertState — we can't import the
// handler package here (handler imports homelab, not the other
// way around) so the per-server UPSERT logic is duplicated. The
// duplication is small (10-15 LOC) and the alternative —
// extracting the helper to a third package — would add more
// coupling than it saves.
package homelab

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// mediaPollAndUpsert is the worker-side mirror of
// handler.PollMediaServerAndInsertState. We can't import the
// handler package here (handler imports homelab, not the other
// way around) so the per-server UPSERT logic is duplicated.
// The duplication is small (10-15 LOC) and the alternative —
// extracting the helper to a third package — would add more
// coupling than it saves.
//
// We could put the per-server UPSERT into the homelab package
// and have the handler call into it, but that would invert the
// dependency direction. Keeping the worker in homelab and the
// handler-side wrapper in handler matches the DownloadsWorker /
// PollClientAndInsertSnapshot pattern from Phase 5.
func mediaPollAndUpsert(ctx context.Context, pool *db.Pool, serverID uuid.UUID, kind, baseURL, apiKey string, tenantID, userID uuid.UUID, logger *slog.Logger) {
	state, pollErr := PollMediaServer(ctx, kind, baseURL, apiKey)

	tx, txErr := pool.Pgx().Begin(ctx)
	if txErr != nil {
		logger.Warn("media worker: tx begin failed",
			"server_id", serverID, "err", txErr)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr.Error() != "tx is closed" {
			logger.Warn("media worker: tx rollback failed",
				"server_id", serverID, "err", rbErr)
		}
	}()

	if state == nil {
		state = &MediaState{}
	}

	// Prune prior now_playing rows for this server.
	if _, err := tx.Exec(ctx,
		`DELETE FROM homelab_now_playing WHERE server_id = $1`,
		serverID,
	); err != nil {
		logger.Warn("media worker: prune now_playing failed",
			"server_id", serverID, "err", err)
		return
	}

	// Insert the freshly-polled now_playing rows.
	for _, sess := range state.NowPlaying {
		if sess.SessionID == "" {
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
			logger.Warn("media worker: insert now_playing failed",
				"server_id", serverID, "session_id", sess.SessionID, "err", err)
		}
	}

	// UPSERT the freshly-polled recent_additions rows.
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
			logger.Warn("media worker: upsert recent_additions failed",
				"server_id", serverID, "item_id", item.ItemID, "err", err)
		}
	}

	// Cap the recent_additions table at the last 20 items per
	// server.
	if _, err := tx.Exec(ctx,
		`DELETE FROM homelab_recent_additions
		  WHERE server_id = $1
		    AND id NOT IN (
		        SELECT id FROM homelab_recent_additions
		         WHERE server_id = $1
		         ORDER BY added_at DESC
		         LIMIT 20
		    )`,
		serverID,
	); err != nil {
		logger.Warn("media worker: prune recent_additions failed",
			"server_id", serverID, "err", err)
		return
	}

	// Update the server row's bookkeeping columns.
	now := time.Now().UTC()
	var status, errMsg string
	if pollErr != nil {
		status = "error"
		errMsg = mediaTruncateErrHelper(pollErr.Error())
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
		logger.Warn("media worker: update server status failed",
			"server_id", serverID, "err", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		logger.Warn("media worker: tx commit failed",
			"server_id", serverID, "err", err)
		return
	}

	if pollErr != nil {
		logger.Warn("media worker: poll failed",
			"server_id", serverID, "kind", kind, "err", pollErr)
	}
}

// mediaTruncateErrHelper caps an error message at 200 chars +
// ellipsis. Matches the handler-side mediaTruncateErr so the
// dashboard renders both paths identically. Named with the
// "Helper" suffix to avoid a name clash if a future refactor
// exports it from this package.
func mediaTruncateErrHelper(s string) string {
	const maxLength = 200
	if len(s) > maxLength {
		return s[:maxLength] + "…"
	}
	return s
}
