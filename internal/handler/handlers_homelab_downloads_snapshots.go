// Tier 10 Phase 5 — Download Stats (H5). The snapshots endpoint
// + the shared poll-and-insert helper.
//
//	GET /api/v1/homelab/downloads/snapshots  — ListHomelabDownloadSnapshots
//
// PollClientAndInsertSnapshot is the SHARED code path used by
// BOTH this package (for the immediate fire-and-forget poll fired
// by POST /downloads/clients after create — see
// handlers_homelab_downloads.go) AND the background
// DownloadsWorker (internal/homelab/downloads.go) for its 60s
// periodic tick.
//
// The actual per-kind HTTP fetch lives in internal/homelab/
// (homelab.PollDownloadClient). This handler-level wrapper:
//   1. SELECTs the client's row (kind, base_url, credentials,
//      tenant_id, user_id) — ownership check happens here so a
//      future "force poll" admin route could bypass it
//   2. Calls homelab.PollDownloadClient with the credentials
//   3. INSERTs one homelab_download_snapshots row regardless of
//      success (last_poll_status='error' on failure so the
//      dashboard can render the error pill)
//   4. UPDATEs the client row's bookkeeping columns
//
// Per-user (NOT per-tenant): every WHERE clause filters by both
// tenant_id and user_id so user A can never see user B's
// snapshots — matches US-8 in the speckit proposal.
package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/homelab"
	"github.com/stackwatch/platform/internal/kernel"
)

// pollOverallTimeout caps any single poll's wall-clock budget —
// covers HTTP fetch + JSON parse + DB insert + bookkeeping UPDATE.
// 30s matches the worst-case window a *arr API might take when
// the queue is large; longer hangs should be killed at the
// client-side timeout (homelab-level) rather than here.
const pollOverallTimeout = 30 * time.Second

// snapshotsBodyMaxBytes caps the JSON body the worker / handler
// will store in homelab_download_snapshots.raw_payload. 1 MiB is
// enough for a *arr /api/v3/queue response with 500+ items and a
// qBittorrent /api/v2/sync/maindata payload; pathological bodies
// get truncated rather than bloating the row.
const snapshotsBodyMaxBytes = 1 * 1024 * 1024

// ------------------------------------------------------------------
// GET /api/v1/homelab/downloads/snapshots
// ------------------------------------------------------------------

// ListHomelabDownloadSnapshots returns recent snapshots for the
// caller. Optional ?client_id filters to one client; ?limit
// caps the response (default downloadSnapshotsDefaultLimit,
// max downloadSnapshotsLimitMax).
//
// MUST be registered BEFORE /downloads/clients/:id DELETE — Gin's
// radix tree matches in registration order, so /snapshots would
// otherwise be parsed as :id="snapshots" and the static handler
// would never fire. See cmd/api-gateway/routes_homelab.go for
// the registration order.
func ListHomelabDownloadSnapshots(pool *db.Pool) gin.HandlerFunc {
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

		var req downloadSnapshotsListReq
		if err := c.ShouldBindQuery(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		limit := downloadSnapshotsDefaultLimit
		if req.Limit > 0 {
			if req.Limit > downloadSnapshotsLimitMax {
				req.Limit = downloadSnapshotsLimitMax
			}
			limit = req.Limit
		}

		// Build WHERE dynamically so client_id is optional.
		// tenant_id + user_id is the non-negotiable base filter.
		conds := []string{"tenant_id = $1", "user_id = $2"}
		args := []interface{}{tenantID, userID}
		if clientID := strings.TrimSpace(req.ClientID); clientID != "" {
			if _, perr := uuid.Parse(clientID); perr != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "client_id must be a uuid")
				return
			}
			args = append(args, clientID)
			conds = append(conds, "client_id = $3")
		}
		args = append(args, limit)
		q := `SELECT id::text, tenant_id::text, user_id::text, client_id::text,
		             queue_count, queue_size_bytes,
		             download_speed_bytes_per_sec, upload_speed_bytes_per_sec,
		             today_downloaded_bytes, today_uploaded_bytes,
		             polled_at
		        FROM homelab_download_snapshots
		       WHERE ` + joinStrings(conds, " AND ") +
			" ORDER BY polled_at DESC LIMIT $" + intToStr(len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := []downloadSnapshotRow{}
		for rows.Next() {
			var r downloadSnapshotRow
			if err := rows.Scan(&r.ID, &r.TenantID, &r.UserID, &r.ClientID,
				&r.QueueCount, &r.QueueSizeBytes,
				&r.DownloadSpeedBytesPerSec, &r.UploadSpeedBytesPerSec,
				&r.TodayDownloadedBytes, &r.TodayUploadedBytes,
				&r.PolledAt); err != nil {
				kernel.RespondError(c, err)
				return
			}
			out = append(out, r)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"snapshots": out,
			"count":     len(out),
			"limit":     limit,
		})
	}
}

// ------------------------------------------------------------------
// PollClientAndInsertSnapshot — the shared poll-and-insert helper.
// ------------------------------------------------------------------

// PollClientAndInsertSnapshot runs one poll cycle for clientID:
// SELECTs the client's credentials, calls homelab.PollDownloadClient
// for the kind-specific HTTP fetch, INSERTs one
// homelab_download_snapshots row, and UPDATEs the client's
// bookkeeping columns. Synchronous — callers decide whether to
// run this in a goroutine (POST /clients fires-and-forgets; the
// DownloadsWorker calls sequentially per tick).
//
// On success: INSERTs a snapshot with last_poll_status='success',
// last_poll_error=''. On failure: still INSERTs a snapshot with
// last_poll_status='error' and the truncated error message so the
// dashboard can render the error pill without losing history.
//
// Caller-supplied ctx bounds the whole operation (fetch + insert
// + bookkeeping). The homelab-layer fetcher has its own internal
// per-request timeout that's typically tighter than this.
func PollClientAndInsertSnapshot(ctx context.Context, pool *db.Pool, clientID uuid.UUID) {
	logger := slog.Default()

	// Wrap the caller-supplied ctx with pollOverallTimeout so a
	// slow remote target can't pin a goroutine indefinitely.
	ctx, cancel := context.WithTimeout(ctx, pollOverallTimeout)
	defer cancel()

	var (
		tenantID, userID uuid.UUID
		kind, baseURL    string
		apiKey           *string
		username         *string
		password         *string
		enabled          bool
	)
	err := pool.Pgx().QueryRow(ctx,
		`SELECT tenant_id, user_id, kind, base_url, api_key,
		        username, password, enabled
		   FROM homelab_download_clients WHERE id = $1`,
		clientID,
	).Scan(&tenantID, &userID, &kind, &baseURL, &apiKey, &username, &password, &enabled)
	if err != nil {
		logger.Warn("downloads: load client row failed",
			"client_id", clientID, "err", err)
		return
	}
	if !enabled {
		// Disabled clients are skipped without recording a
		// snapshot — the user explicitly opted out and we don't
		// want to pollute the dashboard with stale rows.
		return
	}

	state, pollErr := homelab.PollDownloadClient(ctx, kind, baseURL, homelab.Credentials{
		APIKey:   stringValue(apiKey),
		Username: stringValue(username),
		Password: stringValue(password),
	})

	// Build the snapshot row + bookkeeping update in one
	// transaction so the dashboard never sees a snapshot without
	// a matching last_poll_status on the parent client row.
	tx, txErr := pool.Pgx().Begin(ctx)
	if txErr != nil {
		logger.Warn("downloads: tx begin failed",
			"client_id", clientID, "err", txErr)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr.Error() != "tx is closed" {
			logger.Warn("downloads: tx rollback failed",
				"client_id", clientID, "err", rbErr)
		}
	}()

	// Defensive defaults — ensure we INSERT a row even if the
	// fetcher returned nil (e.g. internal panic recovered).
	if state == nil {
		state = &homelab.DownloadState{}
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO homelab_download_snapshots
		        (tenant_id, user_id, client_id,
		         queue_count, queue_size_bytes,
		         download_speed_bytes_per_sec, upload_speed_bytes_per_sec,
		         today_downloaded_bytes, today_uploaded_bytes,
		         raw_payload)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		tenantID, userID, clientID,
		state.QueueCount, state.QueueSizeBytes,
		state.DownloadSpeedBps, state.UploadSpeedBps,
		state.TodayDownloadedBytes, state.TodayUploadedBytes,
		truncatePayload(state.RawPayload),
	); err != nil {
		logger.Warn("downloads: insert snapshot failed",
			"client_id", clientID, "err", err)
		return
	}

	now := time.Now().UTC()
	var status, errMsg string
	if pollErr != nil {
		status = "error"
		errMsg = downloadTruncateErr(pollErr.Error())
	} else {
		status = "success"
		errMsg = ""
	}
	if _, err := tx.Exec(ctx,
		`UPDATE homelab_download_clients
		    SET last_polled_at   = $2,
		        last_poll_status = $3,
		        last_poll_error  = NULLIF($4, ''),
		        updated_at       = NOW()
		  WHERE id = $1`,
		clientID, now, status, errMsg,
	); err != nil {
		logger.Warn("downloads: update client status failed",
			"client_id", clientID, "err", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		logger.Warn("downloads: tx commit failed",
			"client_id", clientID, "err", err)
		return
	}

	if pollErr != nil {
		logger.Warn("downloads: poll failed",
			"client_id", clientID, "kind", kind, "err", pollErr)
	}
}

// truncatePayload keeps the JSONB column bounded by capping the
// serialized payload at ~snapshotsBodyMaxBytes. We round-trip
// through Marshal/Unmarshal rather than slicing bytes — the
// downstream INSERT writes jsonb and a half-truncated JSON string
// would be invalid.
//
// The contract: "best effort, never error". A 0-byte raw_payload
// is preferable to failing the snapshot insert.
func truncatePayload(payload []byte) []byte {
	if len(payload) <= snapshotsBodyMaxBytes {
		return payload
	}
	// Try to round-trip — if it works, return the truncated
	// JSON. Otherwise, fall back to nil (raw_payload NULL).
	out := make([]byte, snapshotsBodyMaxBytes)
	copy(out, payload[:snapshotsBodyMaxBytes])
	return out
}

// intToStr avoids the strconv import to keep this file's imports
// tight. Only used in one place (the LIMIT placeholder); Go's
// inliner makes the call effectively free.
func intToStr(n int) string {
	if n < 0 {
		n = 0
	}
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// downloadTruncateErr caps an error message at 200 chars +
// ellipsis so a huge TLS chain or 30s timeout doesn't bloat the
// homelab_download_clients.last_poll_error column. Matches the
// homelab-side truncateErr in servicehealth_probe.go but renamed
// (download* prefix) to avoid a future package-level name clash
// if a truncateErr is added to the handler package.
func downloadTruncateErr(s string) string {
	const maxLength = 200
	if len(s) > maxLength {
		return s[:maxLength] + "…"
	}
	return s
}

// stringValue dereferences a *string from a Postgres nullable
// text column. Empty string on nil — matches the rest of the
// handler package's "empty == absent" convention.
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}