// Package homelab hosts Tier 10 background workers.
//
// CalendarWorker ticks every 30min, fetches every enabled iCal
// URL in homelab_calendars across all tenants + users, parses the
// response with github.com/arran4/golang-ical, and upserts the
// contained VEVENTs into homelab_events.
//
// Worker must not crash on bad feeds. Each fetch+parse runs inside
// its own goroutine + recovers from any panic so a hung HTTP
// connection or a malformed iCal body can never take the worker
// down.
//
// Per-user (NOT per-tenant): the tick iterates over ALL (tenant, user)
// pairs in the table. The (tenant_id, enabled) partial index makes
// the SELECT cheap on the (typically small) enabled-row set.
//
// Sync helpers (SyncCalendarByID / SyncAllCalendars) live in
// calendar_sync.go. Parse helpers (readStart / propertyValue /
// parsedEvent) live in calendar_parse.go. The split keeps each
// file under the 400-LOC cap.
package homelab

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

const (
	defaultCalendarTickInterval  = 30 * time.Minute
	defaultCalendarMaxPar        = 8
	defaultCalendarFetchTimeout  = 30 * time.Second
	defaultCalendarMaxEvents     = 1000
	defaultCalendarPerSyncBudget = 90 * time.Second
)

// CalendarWorker is the per-process singleton. Start() launches
// the goroutine; the goroutine respects ctx cancellation for
// graceful shutdown.
type CalendarWorker struct {
	pool         *db.Pool
	tickInterval time.Duration
	maxPar       int
	fetchTimeout time.Duration
	maxEvents    int
	perSyncTime  time.Duration
	logger       *slog.Logger
}

// CalendarOption mutates the worker at construction time.
type CalendarOption func(*CalendarWorker)

// WithCalendarTickInterval overrides the default 30min cadence.
func WithCalendarTickInterval(d time.Duration) CalendarOption {
	return func(w *CalendarWorker) { w.tickInterval = d }
}

// WithCalendarMaxPar overrides the parallel-fetch cap.
func WithCalendarMaxPar(n int) CalendarOption {
	return func(w *CalendarWorker) { w.maxPar = n }
}

// WithCalendarLogger overrides the default slog logger.
func WithCalendarLogger(l *slog.Logger) CalendarOption {
	return func(w *CalendarWorker) { w.logger = l }
}

// NewCalendarWorker builds the worker. Call once on api-gateway
// boot, then Start() in a goroutine.
func NewCalendarWorker(pool *db.Pool, opts ...CalendarOption) *CalendarWorker {
	w := &CalendarWorker{
		pool:         pool,
		tickInterval: defaultCalendarTickInterval,
		maxPar:       defaultCalendarMaxPar,
		fetchTimeout: defaultCalendarFetchTimeout,
		maxEvents:    defaultCalendarMaxEvents,
		perSyncTime:  defaultCalendarPerSyncBudget,
		logger:       slog.Default(),
	}
	for _, o := range opts {
		o(w)
	}
	return w
}

// Start launches the worker goroutine. Returns immediately; the
// goroutine exits when ctx is cancelled. The first tick fires
// immediately so a freshly-restarted gateway repopulates events
// within seconds rather than waiting a full 30min.
func (w *CalendarWorker) Start(ctx context.Context) {
	w.logger.Info("calendar worker: starting",
		"tick", w.tickInterval,
		"max_par", w.maxPar,
		"max_events_per_calendar", w.maxEvents)

	go func() {
		// Best-effort startup tick — don't crash if it fails.
		func() {
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("calendar worker: startup tick panic", "err", r)
				}
			}()
			w.tick(ctx)
		}()

		t := time.NewTicker(w.tickInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("calendar worker: stopping")
				return
			case <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							w.logger.Error("calendar worker: tick panic", "err", r)
						}
					}()
					w.tick(ctx)
				}()
			}
		}
	}()
}

// calendarTarget is the small payload the worker pulls for each
// row — keeps the goroutine closure small.
type calendarTarget struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	ID       uuid.UUID
	Name     string
	ICalURL  string
}

// tick runs one pass: fetch every enabled calendar across all
// tenants, fetch+parse each in parallel (bounded), upsert events.
// Per-calendar errors are logged but never abort the batch.
func (w *CalendarWorker) tick(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, w.perSyncTime)
	defer cancel()

	calendars, err := w.fetchEnabledCalendars(ctx)
	if err != nil {
		w.logger.Error("calendar worker: fetch calendars failed", "err", err)
		return
	}
	if len(calendars) == 0 {
		return
	}
	w.logger.Info("calendar worker: syncing", "calendars", len(calendars))

	sem := make(chan struct{}, w.maxPar)
	var wg sync.WaitGroup
	for _, c := range calendars {
		wg.Add(1)
		sem <- struct{}{}
		go func(c calendarTarget) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("calendar worker: sync panic",
						"calendar_id", c.ID, "err", r)
				}
			}()
			if serr := w.syncOne(ctx, c); serr != nil {
				w.logger.Error("calendar worker: sync failed",
					"calendar_id", c.ID, "name", c.Name, "err", serr)
			}
		}(c)
	}
	wg.Wait()
}

// fetchEnabledCalendars returns every enabled calendar in
// homelab_calendars. The (tenant_id, enabled) partial index keeps
// this cheap even with thousands of users.
func (w *CalendarWorker) fetchEnabledCalendars(ctx context.Context) ([]calendarTarget, error) {
	rows, err := w.pool.Pgx().Query(ctx,
		`SELECT tenant_id, user_id, id, name, ical_url
		   FROM homelab_calendars
		  WHERE enabled = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []calendarTarget{}
	for rows.Next() {
		var c calendarTarget
		if err := rows.Scan(&c.TenantID, &c.UserID, &c.ID, &c.Name, &c.ICalURL); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// syncOne does one fetch+parse+upsert for a single calendar. It
// is the shared code path used by both the worker's tick and the
// on-demand SyncCalendarByID helper (see calendar_sync.go).
//
// Flow:
//  1. GET the iCal URL with a 30s timeout
//  2. Parse with github.com/arran4/golang-ical
//  3. For each VEVENT, upsert into homelab_events
//  4. Update homelab_calendars.last_synced_at + last_sync_status
func (w *CalendarWorker) syncOne(parentCtx context.Context, c calendarTarget) error {
	fetchCtx, cancel := context.WithTimeout(parentCtx, w.fetchTimeout)
	defer cancel()

	events, fetchErr := w.fetchAndParse(fetchCtx, c.ICalURL)
	now := time.Now().UTC()

	if fetchErr != nil {
		if uerr := w.updateCalendarStatus(parentCtx, c.ID, "error", fetchErr.Error(), nil); uerr != nil {
			return fmt.Errorf("update status: %w (after fetch: %v)", uerr, fetchErr)
		}
		return fetchErr
	}
	syncedAt := now
	if uerr := w.updateCalendarStatus(parentCtx, c.ID, "success", "", &syncedAt); uerr != nil {
		return fmt.Errorf("update status: %w", uerr)
	}

	if len(events) == 0 {
		return nil
	}
	if err := w.upsertEvents(parentCtx, c, events); err != nil {
		return fmt.Errorf("upsert events: %w", err)
	}
	return nil
}

// fetchAndParse downloads the iCal feed and parses it. Returns
// (events, nil) on success; (nil, err) on any failure. The arran4
// parser is RFC 5545-compliant.
func (w *CalendarWorker) fetchAndParse(ctx context.Context, rawURL string) ([]parsedEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("User-Agent", "StackWatch-Homelab-CalendarWorker/1.0")
	req.Header.Set("Accept", "text/calendar, */*")
	client := &http.Client{Timeout: w.fetchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	cal, err := ics.ParseCalendar(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse ical: %w", err)
	}

	out := []parsedEvent{}
	for _, ve := range cal.Events() {
		uid := strings.TrimSpace(ve.Id())
		if uid == "" {
			// Some iCal publishers omit UID; skip rather than
			// upsert with an empty key (the UNIQUE constraint
			// would collapse every event into one row).
			continue
		}
		start, allDay, ok := readStart(ve)
		if !ok {
			continue
		}
		end, _ := ve.GetEndAt()
		out = append(out, parsedEvent{
			UID:         uid,
			Summary:     strings.TrimSpace(propertyValue(ve, ics.ComponentPropertySummary)),
			Description: strings.TrimSpace(propertyValue(ve, ics.ComponentPropertyDescription)),
			Location:    strings.TrimSpace(propertyValue(ve, ics.ComponentPropertyLocation)),
			StartsAt:    start.UTC(),
			EndsAt:      end.UTC(),
			AllDay:      allDay,
		})
		if len(out) >= w.maxEvents {
			// Cap the upsert batch — anything past this is
			// almost certainly a publisher gone wild.
			break
		}
	}
	return out, nil
}

// upsertEvents writes each parsedEvent to homelab_events using
// ON CONFLICT (calendar_id, uid) DO UPDATE. ON CONFLICT preserves
// the row id (UUID) so existing event references in the frontend
// keep working after a sync re-saves the event with new fields.
func (w *CalendarWorker) upsertEvents(ctx context.Context, c calendarTarget, events []parsedEvent) error {
	tx, err := w.pool.Pgx().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr.Error() != "tx is closed" {
			w.logger.Error("calendar worker: tx rollback failed", "err", rbErr)
		}
	}()

	for _, e := range events {
		var endsAt *time.Time
		if !e.EndsAt.IsZero() {
			es := e.EndsAt
			endsAt = &es
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO homelab_events
			        (tenant_id, user_id, calendar_id, uid, summary,
			         description, location, starts_at, ends_at, all_day)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			 ON CONFLICT (calendar_id, uid) DO UPDATE
			 SET summary     = EXCLUDED.summary,
			     description = EXCLUDED.description,
			     location    = EXCLUDED.location,
			     starts_at   = EXCLUDED.starts_at,
			     ends_at     = EXCLUDED.ends_at,
			     all_day     = EXCLUDED.all_day,
			     updated_at  = NOW()`,
			c.TenantID, c.UserID, c.ID, e.UID, e.Summary,
			e.Description, e.Location, e.StartsAt, endsAt, e.AllDay,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// updateCalendarStatus updates the homelab_calendars bookkeeping
// row after a sync attempt. Runs even when the sync itself
// failed so the dashboard can show the error message inline.
func (w *CalendarWorker) updateCalendarStatus(ctx context.Context, id uuid.UUID, status, errMsg string, syncedAt *time.Time) error {
	var syncedAtArg interface{}
	if syncedAt != nil {
		syncedAtArg = *syncedAt
	}
	_, err := w.pool.Pgx().Exec(ctx,
		`UPDATE homelab_calendars
		    SET last_synced_at   = COALESCE($2, last_synced_at),
		        last_sync_status = $3,
		        last_sync_error  = NULLIF($4, ''),
		        updated_at       = NOW()
		  WHERE id = $1`,
		id, syncedAtArg, status, errMsg,
	)
	return err
}
