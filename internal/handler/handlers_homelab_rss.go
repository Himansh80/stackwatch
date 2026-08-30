// Tier 10 Phase 8 — RSS (H9). 5 protected endpoints back the
// per-user RSS subscription surface (homelab_rss_feeds) +
// the cached-item read/mark-read surface (homelab_rss_items).
//
//	GET    /api/v1/homelab/rss/feeds          — ListHomelabRssFeeds
//	POST   /api/v1/homelab/rss/feeds          — CreateHomelabRssFeed
//	DELETE /api/v1/homelab/rss/feeds/:id      — DeleteHomelabRssFeed
//	GET    /api/v1/homelab/rss/items          — ListHomelabRssItems
//	POST   /api/v1/homelab/rss/items/:id/read — MarkHomelabRssItemRead
//
// The RssWorker (internal/homelab/rss.go) ticks every 5min and
// upserts items from every enabled feed; POST /feeds fires an
// immediate poll after create (via PollFeedAndUpsertItems — same
// helper the worker uses, DRY) so a freshly-pinned feed shows
// items within seconds rather than waiting a full tick.
//
// Per-user (NOT per-tenant): every WHERE clause filters by both
// tenant_id and user_id so user A can never see or modify user
// B's feeds — matches US-8 in the speckit proposal.
//
// Validation:
//   - feed_url must be http/https parseable (matches the
//     worker's defensive check)
//   - feed must parse with gofeed AND yield ≥1 <item> — rejects
//     dead URLs (404) and paywalled placeholder feeds
//   - category must be in the allowedRssCategories palette or
//     defaultRssCategory
package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/homelab"
	"github.com/stackwatch/platform/internal/kernel"
)

// validateRssFeedURL rejects anything that isn't http/https.
// Same check the worker relies on, so we surface the error at
// insert time rather than at first poll.
func validateRssFeedURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("feed_url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("feed_url is not parseable: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("feed_url must be http:// or https://")
	}
	if u.Host == "" {
		return fmt.Errorf("feed_url is missing host")
	}
	return nil
}

// validateRssFeedName enforces a sane title. Empty / 200+ chars
// both rejected — the dashboard's RSS card uses the name as its
// primary label.
func validateRssFeedName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > rssFeedNameMaxLen {
		return fmt.Errorf("name must be %d characters or fewer", rssFeedNameMaxLen)
	}
	return nil
}

// normalizeRssCategory returns the category the handler will
// INSERT. Falls back to defaultRssCategory when the input is
// empty or doesn't match the allowed palette.
func normalizeRssCategory(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultRssCategory
	}
	if _, ok := allowedRssCategories[raw]; ok {
		return raw
	}
	return defaultRssCategory
}

// probeRssFeed does a one-shot HTTP GET + gofeed parse to verify
// the URL is reachable and the body has ≥1 <item>. Returns nil
// on success or a descriptive error string the handler can
// surface to the user verbatim. Bounded by 15s so a slow feed
// doesn't block the POST for too long.
func probeRssFeed(parentCtx context.Context, rawURL string) error {
	ctx, cancel := context.WithTimeout(parentCtx, 15*1000*1000*1000) // 15s
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("User-Agent", "StackWatch-Homelab-RssWorker/1.0")
	req.Header.Set("Accept", "application/atom+xml, application/rss+xml, application/json, */*")
	client := &http.Client{Timeout: 15 * 1000 * 1000 * 1000}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}
	parser := gofeed.NewParser()
	feed, err := parser.Parse(strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("parse failed: %w", err)
	}
	if len(feed.Items) == 0 {
		return errors.New("feed has zero items — publisher may be broken or paywalled")
	}
	return nil
}

// ------------------------------------------------------------------
// GET /api/v1/homelab/rss/feeds
// ------------------------------------------------------------------

// ListHomelabRssFeeds returns every feed the caller has pinned.
// Each row carries unread_count (computed via a subquery) and
// the worker bookkeeping fields so the frontend can render the
// status pill without a follow-up call.
func ListHomelabRssFeeds(pool *db.Pool) gin.HandlerFunc {
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
			`SELECT f.id::text, f.tenant_id::text, f.user_id::text,
			        f.name, f.feed_url, f.category, f.enabled,
			        f.last_polled_at::text, f.last_poll_status, f.last_poll_error,
			        COALESCE(u.cnt, 0),
			        f.created_at::text, f.updated_at::text
			   FROM homelab_rss_feeds f
			   LEFT JOIN (
			       SELECT feed_id, COUNT(*)::int AS cnt
			         FROM homelab_rss_items
			        WHERE read_at IS NULL
			        GROUP BY feed_id
			   ) u ON u.feed_id = f.id
			  WHERE f.tenant_id = $1 AND f.user_id = $2
			  ORDER BY f.created_at DESC`,
			tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := []rssFeedRow{}
		for rows.Next() {
			var (
				id, tID, uID, name, feedURL, category, createdAt, updatedAt string
				enabled                                                     bool
				lastPolled, lastStatus, lastError                           *string
				unread                                                      int
			)
			if serr := rows.Scan(&id, &tID, &uID, &name, &feedURL, &category,
				&enabled, &lastPolled, &lastStatus, &lastError,
				&unread, &createdAt, &updatedAt); serr != nil {
				kernel.RespondError(c, serr)
				return
			}
			out = append(out, newRssFeedRowFromScan(rssFeedRowScan{
				ID: id, TenantID: tID, UserID: uID,
				Name: name, FeedURL: feedURL, Category: category,
				Enabled:        enabled,
				LastPolledAt:   lastPolled,
				LastPollStatus: lastStatus,
				LastPollError:  lastError,
				UnreadCount:    unread,
				CreatedAt:      createdAt,
				UpdatedAt:      updatedAt,
			}))
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"feeds": out,
			"count": len(out),
		})
	}
}

// ------------------------------------------------------------------
// POST /api/v1/homelab/rss/feeds
// ------------------------------------------------------------------

// CreateHomelabRssFeed inserts a new feed for the caller. Fires
// an immediate poll after the INSERT (fire-and-forget) so a
// freshly-pinned feed shows items within seconds.
//
// Validates name (non-empty, ≤ 200 chars), feed_url (http/https
// parseable), and category (allowed palette or default).
// Also probes the feed — must parse with gofeed AND yield ≥1
// <item>. Returns 201 with the inserted row.
func CreateHomelabRssFeed(pool *db.Pool) gin.HandlerFunc {
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

		var req rssFeedReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if err := validateRssFeedName(req.Name); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		req.FeedURL = strings.TrimSpace(req.FeedURL)
		if err := validateRssFeedURL(req.FeedURL); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		// Probe the feed BEFORE inserting so a dead URL
		// never makes it into the table. The UNIQUE
		// (tenant_id, user_id, feed_url) would otherwise
		// lock in a row we can never satisfy.
		if perr := probeRssFeed(c.Request.Context(), req.FeedURL); perr != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_feed", "feed rejected: "+perr.Error())
			return
		}
		category := normalizeRssCategory(req.Category)

		var (
			id, createdAt, updatedAt string
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO homelab_rss_feeds
			        (tenant_id, user_id, name, feed_url, category)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id::text, created_at::text, updated_at::text`,
			tenantID, userID, req.Name, req.FeedURL, category,
		).Scan(&id, &createdAt, &updatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Fire-and-forget immediate poll so items show within
		// seconds rather than waiting a full worker tick. The
		// poll helper recovers from any panic internally.
		newID, perr := uuid.Parse(id)
		if perr == nil {
			go homelab.PollFeedAndUpsertItems(context.Background(), pool, newID, slog.Default())
		}

		kernel.RespondCreated(c, rssFeedRow{
			ID:        id,
			TenantID:  tenantID.String(),
			UserID:    userID.String(),
			Name:      req.Name,
			FeedURL:   req.FeedURL,
			Category:  category,
			Enabled:   true,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
}

// ------------------------------------------------------------------
// DELETE /api/v1/homelab/rss/feeds/:id
// ------------------------------------------------------------------

// DeleteHomelabRssFeed removes a feed (and its items via ON
// DELETE CASCADE FK). Idempotent: 200 with {deleted: 0} when
// the row doesn't exist or isn't the caller's.
func DeleteHomelabRssFeed(pool *db.Pool) gin.HandlerFunc {
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
			`DELETE FROM homelab_rss_feeds
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondOK(c, gin.H{
			"deleted":       tag.RowsAffected(),
			"feed_id":       id.String(),
			"idempotent_ok": true,
		})
	}
}
