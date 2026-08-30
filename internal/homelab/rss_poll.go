// Sync helpers for the RssWorker — split out of rss.go to keep
// that file under the 400-LOC cap.
//
// These MUST stay in lock-step with the matching fetch+parse+upsert
// path in rss.go (PollFeedAndUpsertItems is the single entry point
// shared by the worker tick and the handler's POST /feeds fire-
// and-forget sync). The handler package can't import internal/homelab
// unexported types, so the handler package calls the exported
// helpers in this file directly.
//
// Pattern mirror: calendar_sync.go from Phase 4 (same DRY goal —
// one helper used by both the worker and the HTTP handler).
package homelab

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"

	"github.com/stackwatch/platform/internal/db"
)

// defaultRssSyncTimeout caps how long PollFeedAndUpsertItems will
// block waiting for a fetch+parse+upsert. 60s covers one slow
// feed + the bookkeeping UPDATE; longer feeds should be retried
// via the worker's 5min tick instead.
const defaultRssSyncTimeout = 60 * time.Second

// parsedRssItem is the intermediate shape produced by the gofeed
// parser and consumed by upsertRssItems. Kept unexported because
// it's an implementation detail of the worker package — only the
// feed-id key leaks across the API boundary (in error returns).
type parsedRssItem struct {
	GUID        string
	Title       string
	Link        string
	Summary     string
	Author      string
	PublishedAt time.Time
}

// PollFeedAndUpsertItems fires a single-feed poll on demand and
// returns the new bookkeeping state. Used by the worker's tick
// AND the handler's POST /rss/feeds (after-create poll) so the
// DRY contract is one helper, two callers.
//
// Synchronous (no goroutine launch) so the caller can include
// the new timestamp in the response. A 60s overall budget covers
// one slow feed + the bookkeeping UPDATE.
//
// Returns:
//
//	polledAt  — last_polled_at after the poll (zero if no row)
//	status    — last_poll_status ('success' | 'error' | '')
//	err       — non-nil when the poll itself failed (the caller
//	            can still read status to surface the error to the
//	            user; the row is already updated with the error
//	            message)
func PollFeedAndUpsertItems(parentCtx context.Context, pool *db.Pool, feedID uuid.UUID, logger *slog.Logger) (polledAt time.Time, status string, err error) {
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithTimeout(parentCtx, defaultRssSyncTimeout)
	defer cancel()

	var (
		tenantID, userID uuid.UUID
		name, feedURL    string
	)
	if qerr := pool.Pgx().QueryRow(ctx,
		`SELECT tenant_id, user_id, name, feed_url
		   FROM homelab_rss_feeds WHERE id = $1`,
		feedID,
	).Scan(&tenantID, &userID, &name, &feedURL); qerr != nil {
		return time.Time{}, "", qerr
	}

	if verr := validateFeedURL(feedURL); verr != nil {
		// Defensive — the handler validates at insert time
		// already, but if a malformed row somehow slipped
		// into the DB, surface the error rather than
		// silently failing the HTTP fetch.
		msg := verr.Error()
		_ = updateFeedStatus(ctx, pool, feedID, "error", msg, nil)
		return time.Time{}, "error", verr
	}

	items, fetchErr := fetchAndParseRss(ctx, feedURL)
	now := time.Now().UTC()

	if fetchErr != nil {
		if uerr := updateFeedStatus(ctx, pool, feedID, "error", fetchErr.Error(), nil); uerr != nil {
			return time.Time{}, "error", fmt.Errorf("update status: %w (after fetch: %v)", uerr, fetchErr)
		}
		return time.Time{}, "error", fetchErr
	}

	if uerr := updateFeedStatus(ctx, pool, feedID, "success", "", &now); uerr != nil {
		return now, "success", fmt.Errorf("update status: %w", uerr)
	}

	if len(items) == 0 {
		return now, "success", nil
	}
	if uerr := upsertRssItems(ctx, pool, tenantID, userID, feedID, items); uerr != nil {
		return now, "success", fmt.Errorf("upsert items: %w", uerr)
	}
	return now, "success", nil
}

// fetchAndParseRss downloads the feed and parses it. Returns
// (items, nil) on success; (nil, err) on any failure. The gofeed
// parser auto-detects Atom 1.0 / RSS 2.0 / JSON Feed 1.1.
func fetchAndParseRss(ctx context.Context, rawURL string) ([]parsedRssItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("User-Agent", "StackWatch-Homelab-RssWorker/1.0")
	req.Header.Set("Accept", "application/atom+xml, application/rss+xml, application/json, */*")
	client := &http.Client{Timeout: defaultRssFetchTimeout}
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

	parser := gofeed.NewParser()
	feed, err := parser.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse rss: %w", err)
	}

	out := []parsedRssItem{}
	for _, it := range feed.Items {
		guid := strings.TrimSpace(it.GUID)
		link := strings.TrimSpace(it.Link)
		title := strings.TrimSpace(it.Title)
		// Some publishers emit <item>s without a guid; skip
		// rather than upsert with an empty key (the UNIQUE
		// constraint would collapse every such item into one
		// row).
		if guid == "" {
			continue
		}
		// Title or link must be present — without one we can't
		// render the row in the widget.
		if title == "" || link == "" {
			continue
		}
		published := time.Time{}
		if it.PublishedParsed != nil {
			published = it.PublishedParsed.UTC()
		} else if it.UpdatedParsed != nil {
			published = it.UpdatedParsed.UTC()
		}
		// Author may be a *gofeed.Person (with .Name) or a plain
		// string depending on feed format. Normalize to string.
		var authorStr string
		if it.Author != nil {
			authorStr = strings.TrimSpace(it.Author.Name)
		}
		out = append(out, parsedRssItem{
			GUID:        guid,
			Title:       title,
			Link:        link,
			Summary:     strings.TrimSpace(it.Description),
			Author:      authorStr,
			PublishedAt: published,
		})
		if len(out) >= defaultRssMaxItemsPerPoll {
			// Cap the upsert batch — anything past this is
			// almost certainly a publisher gone wild.
			break
		}
	}
	return out, nil
}

// upsertRssItems writes each parsedRssItem to homelab_rss_items
// using ON CONFLICT (feed_id, guid) DO UPDATE. ON CONFLICT
// preserves the row id (UUID) so existing references in the
// frontend keep working after a poll re-saves the item with new
// fields. read_at is NEVER touched — only the user can mark an
// item read (POST /items/:id/read).
func upsertRssItems(ctx context.Context, pool *db.Pool, tenantID, userID, feedID uuid.UUID, items []parsedRssItem) error {
	tx, err := pool.Pgx().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr.Error() != "tx is closed" {
			// Best-effort rollback — log + continue.
			_ = rbErr
		}
	}()

	for _, it := range items {
		var pubAt interface{}
		if !it.PublishedAt.IsZero() {
			pubAt = it.PublishedAt
		}
		var summary, author interface{}
		if it.Summary != "" {
			summary = it.Summary
		}
		if it.Author != "" {
			author = it.Author
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO homelab_rss_items
			        (tenant_id, user_id, feed_id, guid,
			         title, link, summary, author, published_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (feed_id, guid) DO UPDATE
			 SET title         = EXCLUDED.title,
			     link          = EXCLUDED.link,
			     summary       = EXCLUDED.summary,
			     author        = EXCLUDED.author,
			     published_at  = EXCLUDED.published_at`,
			tenantID, userID, feedID, it.GUID,
			it.Title, it.Link, summary, author, pubAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// updateFeedStatus updates the homelab_rss_feeds bookkeeping row
// after a poll attempt. Runs even when the poll itself failed so
// the dashboard can show the error message inline.
func updateFeedStatus(ctx context.Context, pool *db.Pool, id uuid.UUID, status, errMsg string, polledAt *time.Time) error {
	var polledAtArg interface{}
	if polledAt != nil {
		polledAtArg = *polledAt
	}
	_, err := pool.Pgx().Exec(ctx,
		`UPDATE homelab_rss_feeds
		    SET last_polled_at   = COALESCE($2, last_polled_at),
		        last_poll_status = $3,
		        last_poll_error  = NULLIF($4, ''),
		        updated_at       = NOW()
		  WHERE id = $1`,
		id, polledAtArg, status, errMsg,
	)
	return err
}
