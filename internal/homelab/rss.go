// Package homelab — RSS worker (Tier 10 Phase 8 / H9 — RSS /
// Activity Feed).
//
// RssWorker ticks every 5min, fetches every enabled feed URL in
// homelab_rss_feeds across all tenants + users, parses each one
// with github.com/mmcdole/gofeed (Atom 1.0, RSS 2.0, JSON Feed
// 1.1), and upserts the contained <item>s into homelab_rss_items.
//
// Worker must not crash on bad feeds. Each fetch+parse runs inside
// its own goroutine + recovers from any panic so a hung HTTP
// connection, a 4xx/5xx, or a malformed body can never take the
// worker down.
//
// Per-user (NOT per-tenant): the tick iterates over ALL (tenant,
// user) pairs in the table. The (user_id) index makes the SELECT
// cheap on the (typically small) per-user feed set.
//
// Sync helpers (PollFeedAndUpsertItems / PollAllFeeds) live in
// rss_poll.go so this file stays under the 400-LOC cap. The split
// mirrors calendar.go + calendar_sync.go from Phase 4.
package homelab

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

const (
	// defaultRssTickInterval matches spec §H9 — "every 5min".
	defaultRssTickInterval = 5 * time.Minute

	// defaultRssMaxPar caps parallel feed fetches so a user
	// pinning 50 feeds can't blow the connection pool. 8 is
	// well below PostgreSQL's max_connections (typically 100).
	defaultRssMaxPar = 8

	// defaultRssFetchTimeout caps a single feed fetch. Most
	// RSS feeds respond in <2s; 30s is a generous ceiling that
	// still keeps the worker moving on stuck hosts.
	defaultRssFetchTimeout = 30 * time.Second

	// defaultRssMaxItemsPerPoll caps how many items we'll
	// INSERT in a single poll. RSS feeds have wildly variable
	// sizes (some re-publish their entire archive every poll);
	// 100 strikes a balance between fresh content and DB
	// pressure. Items past this are silently dropped — the
	// next poll will pick up newer entries first anyway.
	defaultRssMaxItemsPerPoll = 100

	// defaultRssPerTickBudget caps the whole tick. Any feeds
	// that didn't complete within the budget are left for the
	// next 5min tick.
	defaultRssPerTickBudget = 90 * time.Second
)

// RssWorker is the per-process singleton. Start() launches the
// goroutine; the goroutine respects ctx cancellation for graceful
// shutdown.
type RssWorker struct {
	pool         *db.Pool
	tickInterval time.Duration
	maxPar       int
	fetchTimeout time.Duration
	maxItems     int
	perTickTime  time.Duration
	logger       *slog.Logger
}

// RssOption mutates the worker at construction time.
type RssOption func(*RssWorker)

// WithRssTickInterval overrides the default 5min cadence.
func WithRssTickInterval(d time.Duration) RssOption {
	return func(w *RssWorker) { w.tickInterval = d }
}

// WithRssMaxPar overrides the parallel-fetch cap.
func WithRssMaxPar(n int) RssOption {
	return func(w *RssWorker) { w.maxPar = n }
}

// WithRssLogger overrides the default slog logger.
func WithRssLogger(l *slog.Logger) RssOption {
	return func(w *RssWorker) { w.logger = l }
}

// NewRssWorker builds the worker. Call once on api-gateway boot,
// then Start() in a goroutine.
func NewRssWorker(pool *db.Pool, opts ...RssOption) *RssWorker {
	w := &RssWorker{
		pool:         pool,
		tickInterval: defaultRssTickInterval,
		maxPar:       defaultRssMaxPar,
		fetchTimeout: defaultRssFetchTimeout,
		maxItems:     defaultRssMaxItemsPerPoll,
		perTickTime:  defaultRssPerTickBudget,
		logger:       slog.Default(),
	}
	for _, o := range opts {
		o(w)
	}
	return w
}

// Start launches the worker goroutine. Returns immediately; the
// goroutine exits when ctx is cancelled. The first tick fires
// immediately so a freshly-restarted gateway repopulates feeds
// within seconds rather than waiting a full 5min.
func (w *RssWorker) Start(ctx context.Context) {
	w.logger.Info("rss worker: starting",
		"tick", w.tickInterval,
		"max_par", w.maxPar,
		"max_items_per_feed", w.maxItems)

	go func() {
		// Best-effort startup tick — don't crash if it fails.
		func() {
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("rss worker: startup tick panic", "err", r)
				}
			}()
			w.tick(ctx)
		}()

		t := time.NewTicker(w.tickInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("rss worker: stopping")
				return
			case <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							w.logger.Error("rss worker: tick panic", "err", r)
						}
					}()
					w.tick(ctx)
				}()
			}
		}
	}()
}

// rssTarget is the small payload the worker pulls for each row —
// keeps the goroutine closure small.
type rssTarget struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	ID       uuid.UUID
	Name     string
	FeedURL  string
}

// tick runs one pass: fetch every enabled feed across all tenants
// in parallel (bounded), parse each with gofeed, upsert items.
// Per-feed errors are logged but never abort the batch.
func (w *RssWorker) tick(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, w.perTickTime)
	defer cancel()

	feeds, err := w.fetchEnabledFeeds(ctx)
	if err != nil {
		w.logger.Error("rss worker: fetch feeds failed", "err", err)
		return
	}
	if len(feeds) == 0 {
		return
	}
	w.logger.Info("rss worker: polling", "feeds", len(feeds))

	sem := make(chan struct{}, w.maxPar)
	var wg sync.WaitGroup
	for _, f := range feeds {
		wg.Add(1)
		sem <- struct{}{}
		go func(f rssTarget) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("rss worker: poll panic",
						"feed_id", f.ID, "err", r)
				}
			}()
			if _, _, serr := PollFeedAndUpsertItems(ctx, w.pool, f.ID, w.logger); serr != nil {
				w.logger.Error("rss worker: poll failed",
					"feed_id", f.ID, "name", f.Name, "err", serr)
			}
		}(f)
	}
	wg.Wait()
}

// fetchEnabledFeeds returns every enabled feed in
// homelab_rss_feeds. The (user_id) index keeps the SELECT cheap
// even with thousands of users.
func (w *RssWorker) fetchEnabledFeeds(ctx context.Context) ([]rssTarget, error) {
	rows, err := w.pool.Pgx().Query(ctx,
		`SELECT tenant_id, user_id, id, name, feed_url
		   FROM homelab_rss_feeds
		  WHERE enabled = TRUE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []rssTarget{}
	for rows.Next() {
		var f rssTarget
		if err := rows.Scan(&f.TenantID, &f.UserID, &f.ID, &f.Name, &f.FeedURL); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// validateFeedURL is the same allowlist the handler enforces at
// POST time. Used here as a defensive guard so a malformed row
// (somehow inserted with file:// or ftp://) can never escape into
// a Go HTTP client call.
func validateFeedURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("feed_url is empty")
	}
	if len(raw) < 8 {
		return fmt.Errorf("feed_url too short")
	}
	if raw[:7] != "http://" && raw[:8] != "https://" {
		return fmt.Errorf("feed_url must start with http:// or https://")
	}
	return nil
}
