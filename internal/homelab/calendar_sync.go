// Sync helpers for the CalendarWorker — split out of calendar.go
// to keep that file under the 400-LOC cap.
//
// These MUST stay in lock-step with the matching fetch+parse+upsert
// path in calendar.go (syncOne + fetchAndParse + upsertEvents) —
// the handler package can't import internal/homelab's unexported
// types, so the handler package calls these exported helpers
// directly. See calendar.go for the sync contract.
package homelab

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// defaultCalendarSyncTimeout caps how long SyncCalendarByID will
// block waiting for a fetch+parse+upsert. 60s covers one slow
// iCal fetch + the bookkeeping UPDATE; longer feeds should be
// retried via the worker's 30min tick instead.
const defaultCalendarSyncTimeout = 60 * time.Second

// SyncCalendarByID fires a single-calendar sync on demand and
// returns the new bookkeeping state. Used by the handler's
// POST /calendars (after-create sync) and POST /calendars/:id/
// refresh (user-triggered manual sync that returns the new
// last_synced_at in the response body).
//
// Synchronous (no goroutine launch) so the caller can include
// the new timestamp in the response. A 60s overall budget covers
// one slow iCal fetch + the bookkeeping UPDATE.
//
// Returns:
//   syncedAt  — last_synced_at after the sync (zero if no row)
//   status    — last_sync_status ('success' | 'error' | '')
//   err       — non-nil when the sync itself failed (the caller
//               can still read status to surface the error to the
//               user; the row is already updated with the error
//               message)
func SyncCalendarByID(pool *db.Pool, calendarID uuid.UUID, logger *slog.Logger) (syncedAt time.Time, status string, err error) {
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultCalendarSyncTimeout)
	defer cancel()
	w := NewCalendarWorker(pool, WithCalendarLogger(logger))
	var c calendarTarget
	if qerr := pool.Pgx().QueryRow(ctx,
		`SELECT tenant_id, user_id, id, name, ical_url
		   FROM homelab_calendars WHERE id = $1`,
		calendarID,
	).Scan(&c.TenantID, &c.UserID, &c.ID, &c.Name, &c.ICalURL); qerr != nil {
		return time.Time{}, "", qerr
	}
	if serr := w.syncOne(ctx, c); serr != nil {
		// Sync itself failed — re-read the bookkeeping row so
		// the response still shows the new last_sync_status
		// + last_sync_error (updateCalendarStatus ran inside
		// syncOne's error path).
		var readErr error
		if readErr = pool.Pgx().QueryRow(ctx,
			`SELECT last_synced_at, last_sync_status
			   FROM homelab_calendars WHERE id = $1`,
			calendarID,
		).Scan(&syncedAt, &status); readErr != nil {
			logger.Warn("calendar: readback after sync failed",
				"calendar_id", calendarID, "err", readErr)
		}
		return syncedAt, status, serr
	}
	// Sync succeeded — read back so the response carries the
	// freshest last_synced_at + last_sync_status. Best-effort:
	// failure here doesn't fail the call.
	if readErr := pool.Pgx().QueryRow(ctx,
		`SELECT last_synced_at, last_sync_status
		   FROM homelab_calendars WHERE id = $1`,
		calendarID,
	).Scan(&syncedAt, &status); readErr != nil {
		logger.Warn("calendar: readback after sync failed",
			"calendar_id", calendarID, "err", readErr)
	}
	return syncedAt, status, nil
}

// SyncAllCalendars fires a full worker tick on demand. Always
// launches a goroutine — callers don't want to block the request
// lifecycle on a multi-calendar sync.
//
// Reserved for a future POST /calendars/refresh-all endpoint;
// nothing currently calls it. Kept here so the pattern is
// available for Phase 5+ without re-plumbing the worker.
func SyncAllCalendars(pool *db.Pool, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("calendar: SyncAllCalendars panic", "err", r)
			}
		}()
		w := NewCalendarWorker(pool, WithCalendarLogger(logger))
		ctx, cancel := context.WithTimeout(context.Background(), 2*defaultCalendarPerSyncBudget)
		defer cancel()
		w.tick(ctx)
	}()
}
