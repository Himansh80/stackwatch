// Tier 10 Phase 7 — Search (H7). 3 protected endpoints back the
// global search surface that crosses every per-user homelab
// sub-feature (notes, todos, services, calendars, downloads,
// media). No new tables — search reads the 13 existing homelab_*
// tables via a parameterized SQL UNION ALL. Honors tenant_id +
// user_id in EVERY branch so a user can never see another user's
// data through search.
//
//	GET /api/v1/homelab/search             — SearchHomelabGlobal (full results w/ snippet + score)
//	GET /api/v1/homelab/search/suggestions — SearchHomelabSuggestions (top-8 autocomplete)
//	GET /api/v1/homelab/search/kinds       — ListHomelabSearchKinds (kind catalog)
//
// Why a parameterized UNION ALL (not six separate queries, not an
// external index): the dataset is small per user (≤ a few hundred
// rows of each kind) and the planner parallelizes the CTE branches
// trivially. Adding Elasticsearch / pg_trgm is premature for the
// Phase 7 scope — if a power user outgrows this, swap in a
// tsvector column on homelab_notes later.
//
// Score formula: per-row, CASE WHEN title ILIKE q THEN 3 WHEN
// body/description ILIKE q THEN 1 ELSE 0 END. Ordering is score
// DESC then title ASC (stable). Limit cap (100) prevents a
// pathological q from returning thousands of rows.
package handler

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// searchLimitDefault / searchLimitMax bound the global search.
// Defaults match the spec §H7 (default 20, cap 100). Suggestions
// use a separate, smaller cap (8) for snappy typeahead.
const (
	searchLimitDefault = 20
	searchLimitMax     = 100
	suggestLimit       = 8
	maxSearchQueryLen  = 200
)

// searchKindRow is the per-row shape returned by global search.
// `url` is the relative path the frontend uses to navigate into
// the right tab on click — keeps the backend agnostic of the
// React Router setup.
type searchKindRow struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
	Score   int    `json:"score"`
}

// suggestionRow is the trimmed shape for typeahead. Frontend
// doesn't need snippet or score when pre-rendering the dropdown.
type suggestionRow struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// searchKindMeta is one row in the kinds catalog response. Kept
// hand-coded (NOT driven by the tables) so the frontend can show
// a stable "📝 Notes · 📌 Services · …" chip even if a particular
// table is empty. Phase 8/9 add RSS + Scheduler entries here.
type searchKindMeta struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
}

// searchKindsCatalog is the hardcoded list of searchable kinds.
// Order matches the tab strip left-to-right so the dropdown
// reads naturally. Phase 8/9 extend this list (RSS / Scheduler)
// in the same file.
var searchKindsCatalog = []searchKindMeta{
	{Kind: "notes", Label: "Notes", Icon: "📝"},
	{Kind: "todos", Label: "Todos", Icon: "✅"},
	{Kind: "services", Label: "Services", Icon: "📌"},
	{Kind: "calendars", Label: "Calendars", Icon: "📅"},
	{Kind: "downloads", Label: "Downloads", Icon: "⬇️"},
	{Kind: "media", Label: "Media", Icon: "🎬"},
}

// validSearchKindSet lets us cheaply reject unknown ?kinds=
// values without scanning the catalog slice every request.
var validSearchKindSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(searchKindsCatalog))
	for _, k := range searchKindsCatalog {
		m[k.Kind] = struct{}{}
	}
	return m
}()// ------------------------------------------------------------------
// GET /api/v1/homelab/search
// ------------------------------------------------------------------

// SearchHomelabGlobal returns ranked matches across every
// searchable homelab table for the caller. Single SQL UNION ALL
// (searchUnionSQL below) parameterized on (tenant_id, user_id,
// escaped pattern). Score ranks title hits (3) above body hits
// (1); ties broken by title ASC for stable ordering.
//
// Query params:
//
//	q     — required, search string (after trim, ≤ 200 chars)
//	limit — optional, default 20, capped at 100
//	kinds — optional, comma-separated subset of
//	         {notes, todos, services, calendars, downloads, media}.
//	         Default = all six. Unknown values 400.
//
// Response: {results, count, q, kinds, limit}. Each result
// carries kind, id, title, snippet (notes/todos only), url,
// score. The frontend uses url to navigate into the right tab on
// click.
func SearchHomelabGlobal(pool *db.Pool) gin.HandlerFunc {
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

		q := strings.TrimSpace(c.Query("q"))
		if q == "" {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "q is required")
			return
		}
		if len(q) > maxSearchQueryLen {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "q must be 200 characters or fewer")
			return
		}

		limit := searchLimitDefault
		if v := strings.TrimSpace(c.Query("limit")); v != "" {
			n, perr := parseSearchLimit(v, searchLimitMax)
			if perr != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "limit: "+perr.Error())
				return
			}
			if n > 0 {
				limit = n
			}
		}

		kinds, kerr := resolveSearchKinds(c.Query("kinds"))
		if kerr != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", kerr.Error())
			return
		}

		pattern := escapeLikePattern(q)

		// Build the dynamic SQL — only includes UNION branches
		// for the kinds the caller asked for. Always keeps
		// tenant_id + user_id in the WHERE of every branch.
		sqlStr, args := buildSearchSQL(pattern, tenantID, userID, kinds, limit)

		rows, err := pool.Pgx().Query(c.Request.Context(), sqlStr, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := []searchKindRow{}
		for rows.Next() {
			var r searchKindRow
			if serr := rows.Scan(&r.Kind, &r.ID, &r.Title,
				&r.Snippet, &r.URL, &r.Score); serr != nil {
				kernel.RespondError(c, serr)
				return
			}
			out = append(out, r)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Stable ranking on the merged result: score DESC, then
		// title ASC for ties (already enforced inside the SQL
		// ORDER BY, but we re-sort here in case the planner
		// returns rows out of order across CTE branches).
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Score != out[j].Score {
				return out[i].Score > out[j].Score
			}
			return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title)
		})
		if len(out) > limit {
			out = out[:limit]
		}

		kernel.RespondOK(c, gin.H{
			"results": out,
			"count":   len(out),
			"q":       q,
			"kinds":   kinds,
			"limit":   limit,
		})
	}
}

// ------------------------------------------------------------------
// GET /api/v1/homelab/search/suggestions
// ------------------------------------------------------------------

// SearchHomelabSuggestions is the typeahead endpoint that powers
// the GlobalSearch dropdown. Always returns at most `suggestLimit`
// (8) results, ordered by score DESC then title ASC, no snippet
// (the dropdown has no room). Used on every keystroke after a
// 300ms debounce in the frontend.
//
// Query params:
//
//	q — required, search string (≤ 200 chars)
//	kinds — optional subset, same vocabulary as global search.
//
// Empty q returns an empty list (200) — keeps the typeahead
// responsive when the user clears the input.
func SearchHomelabSuggestions(pool *db.Pool) gin.HandlerFunc {
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

		q := strings.TrimSpace(c.Query("q"))
		if q == "" {
			kernel.RespondOK(c, gin.H{
				"suggestions": []suggestionRow{},
				"count":       0,
				"q":           q,
			})
			return
		}
		if len(q) > maxSearchQueryLen {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "q must be 200 characters or fewer")
			return
		}

		kinds, kerr := resolveSearchKinds(c.Query("kinds"))
		if kerr != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", kerr.Error())
			return
		}

		pattern := escapeLikePattern(q)

		// Smaller projection (kind, id, title, url only — no
		// snippet/score) and a hard limit of 8.
		sqlStr, args := buildSuggestionSQL(pattern, tenantID, userID, kinds, suggestLimit)

		rows, err := pool.Pgx().Query(c.Request.Context(), sqlStr, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := []suggestionRow{}
		for rows.Next() {
			var s suggestionRow
			if serr := rows.Scan(&s.Kind, &s.ID, &s.Title, &s.URL); serr != nil {
				kernel.RespondError(c, serr)
				return
			}
			out = append(out, s)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondOK(c, gin.H{
			"suggestions": out,
			"count":       len(out),
			"q":           q,
			"kinds":       kinds,
		})
	}
}

// ------------------------------------------------------------------
// GET /api/v1/homelab/search/kinds
// ------------------------------------------------------------------

// ListHomelabSearchKinds returns the static catalog of searchable
// kinds — same shape as the dropdown chips in the frontend. Used
// by the GlobalSearch component to render a "kind filter" strip.
// Order matches searchKindsCatalog so the chips read naturally.
func ListHomelabSearchKinds() gin.HandlerFunc {
	return func(c *gin.Context) {
		kernel.RespondOK(c, gin.H{
			"kinds": searchKindsCatalog,
			"count": len(searchKindsCatalog),
		})
	}
}
