// Tier 10 Phase 8 — RSS (H9).
//
// Shared types for the homelab RSS surface. Lives in its own
// file so handlers_homelab_rss.go (5 routes) can import these
// types without growing past the 400-LOC cap.
//
// Storage: homelab_rss_feeds + homelab_rss_items
// (migrations/041_homelab.sql — Phase 8 extension).
//
// Per-user design: every query is filtered by both tenant_id and
// user_id. A user in tenant A cannot see (or modify) another
// user's RSS subscriptions even if they share a tenant — matches
// US-8 in the speckit proposal ("my homelab doesn't change under
// me when my co-founder rearranges theirs").
package handler

import "time"

// ------------------------------------------------------------------
// Feed-row shape — what /homelab/rss/feeds returns + accepts.
// ------------------------------------------------------------------

// rssFeedRow is the JSON shape returned by GET /rss/feeds and
// POST /rss/feeds. Mirrors the columns of homelab_rss_feeds plus
// an unread_count that the handler computes with a sub-query so
// the frontend can show "12 unread" badges without a follow-up
// GET. last_polled_at + last_poll_status are RFC3339-ish strings
// so the frontend can render relative timestamps directly.
type rssFeedRow struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	UserID         string `json:"user_id"`
	Name           string `json:"name"`
	FeedURL        string `json:"feed_url"`
	Category       string `json:"category"`
	Enabled        bool   `json:"enabled"`
	LastPolledAt   string `json:"last_polled_at,omitempty"`
	LastPollStatus string `json:"last_poll_status,omitempty"`
	LastPollError  string `json:"last_poll_error,omitempty"`
	UnreadCount    int    `json:"unread_count"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// rssFeedReq is the body for POST /rss/feeds. name + feed_url
// are required; category is optional (defaults to "general").
// The handler validates feed_url is http/https parseable and
// that the feed parses with at least one item (rejects dead
// URLs early).
type rssFeedReq struct {
	Name     string `json:"name"      binding:"required"`
	FeedURL  string `json:"feed_url"  binding:"required"`
	Category string `json:"category"`
}

// scanRssFeedRow scans one row of the standard feeds SELECT into
// an rssFeedRow. Helper shared by List + Create so the Scan
// signature stays in one place.
type rssFeedRowScan struct {
	ID             string
	TenantID       string
	UserID         string
	Name           string
	FeedURL        string
	Category       string
	Enabled        bool
	LastPolledAt   *string
	LastPollStatus *string
	LastPollError  *string
	UnreadCount    int
	CreatedAt      string
	UpdatedAt      string
}

func newRssFeedRowFromScan(s rssFeedRowScan) rssFeedRow {
	row := rssFeedRow{
		ID:          s.ID,
		TenantID:    s.TenantID,
		UserID:      s.UserID,
		Name:        s.Name,
		FeedURL:     s.FeedURL,
		Category:    s.Category,
		Enabled:     s.Enabled,
		UnreadCount: s.UnreadCount,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
	if s.LastPolledAt != nil {
		row.LastPolledAt = *s.LastPolledAt
	}
	if s.LastPollStatus != nil {
		row.LastPollStatus = *s.LastPollStatus
	}
	if s.LastPollError != nil {
		row.LastPollError = *s.LastPollError
	}
	return row
}

// ------------------------------------------------------------------
// Item-row shape — what /homelab/rss/items returns.
// ------------------------------------------------------------------

// rssItemRow is the JSON shape returned by GET /rss/items.
// feed_id + feed_name are denormalized into the response so the
// frontend can render the source feed label inline without a
// follow-up GET /feeds call. published_at is RFC3339 string so
// the frontend can do relative-time formatting directly.
type rssItemRow struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	UserID      string `json:"user_id"`
	FeedID      string `json:"feed_id"`
	FeedName    string `json:"feed_name"`
	GUID        string `json:"guid"`
	Title       string `json:"title"`
	Link        string `json:"link"`
	Summary     string `json:"summary,omitempty"`
	Author      string `json:"author,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
	ReadAt      string `json:"read_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// rssItemListReq is the parsed query string for GET /rss/items.
// feed_id is optional (omit = all the caller's feeds). unread is
// optional (true = only unread). limit caps the response size
// (default 50, max 200) so a power user with 20 feeds × thousands
// of items can't blow the dashboard.
type rssItemListReq struct {
	FeedID string `form:"feed_id"`
	Unread string `form:"unread"`
	Limit  int    `form:"limit"`
}

// ------------------------------------------------------------------
// Validation enums — enforced on POST to reject bad input before
// any DB call. Mirrors the no-CHECK-constraint design note in
// migrations/041_homelab.sql.
// ------------------------------------------------------------------

// allowedRssCategories is the small set the frontend's category
// dropdown offers. Locked to a short list so the chip color and
// label stay consistent across the dashboard. Free-form text
// would invite near-duplicates ("News" / "news" / "news feeds").
//
// Adding a new category here is safe — existing rows that use a
// removed category simply don't match the new dropdown; the
// widget falls back to "general".
var allowedRssCategories = map[string]struct{}{
	"general":  {},
	"news":     {},
	"releases": {},
	"blogs":    {},
	"podcasts": {},
	"other":    {},
}

// defaultRssCategory is the row's default when the user doesn't
// pick one in the POST body. Matches the table DEFAULT.
const defaultRssCategory = "general"

// rssFeedNameMaxLen caps the user-supplied feed name so a
// pathological name can't blow the dashboard layout. 200 matches
// the limit used for calendars + notes (consistent UX rule).
const rssFeedNameMaxLen = 200

// rssItemsListLimit is the default cap on GET /rss/items. The
// widget shows the most recent ~50 items per tab; 50 is the
// snappy default that keeps the response <100 KB.
const rssItemsListLimit = 50

// rssItemsListLimitMax caps the user-supplied ?limit= so a
// malicious caller can't request millions of rows. 200 covers
// the legitimate "show me everything for the last week" case.
const rssItemsListLimitMax = 200

// timeParseLayout is the layout used to parse the frontend's
// datetime-local inputs (YYYY-MM-DDTHH:MM). Reserved for a future
// ?published_after= filter; declared now so the imports list
// stays stable when the filter lands.
const rssTimeParseLayout = "2006-01-02T15:04"

// _ keeps time referenced so the imports list doesn't get
// pruned by goimports in case future revisions want to use the
// time package directly (e.g. for RFC3339 helpers). Cheap
// insurance vs. a future import churn.
var _ = time.RFC3339
