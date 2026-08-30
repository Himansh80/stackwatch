// Tier 10 Phase 8 — RSS items list + mark-read.
//
// Split out of handlers_homelab_rss.go so each file stays under
// the 400-LOC cap. Routes:
//
//	GET    /api/v1/homelab/rss/items          — ListHomelabRssItems
//	POST   /api/v1/homelab/rss/items/:id/read — MarkHomelabRssItemRead
//
// Per-user (NOT per-tenant): every WHERE clause filters by both
// tenant_id and user_id. The unread filter uses the
// idx_homelab_rss_items_unread partial index for O(1) badge reads.
package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// GET /api/v1/homelab/rss/items
// ------------------------------------------------------------------

// ListHomelabRssItems returns cached items for the caller. Query
// params:
//
//	feed_id — optional, restrict to one feed
//	unread  — optional ("true" = only items where read_at IS NULL)
//	limit   — optional (default 50, max 200)
//
// Results ordered by published_at DESC NULLS LAST, then created_at
// DESC.
func ListHomelabRssItems(pool *db.Pool) gin.HandlerFunc {
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

		// Parse + validate query params manually (no c.ShouldBindQuery
		// because of the parseable strings-as-bool quirk).
		var (
			feedIDStr = strings.TrimSpace(c.Query("feed_id"))
			unreadRaw = strings.TrimSpace(c.Query("unread"))
			limit     = rssItemsListLimit
		)
		if v := strings.TrimSpace(c.Query("limit")); v != "" {
			n := 0
			for _, ch := range v {
				if ch < '0' || ch > '9' {
					kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "limit must be a non-negative integer")
					return
				}
				n = n*10 + int(ch-'0')
				if n > rssItemsListLimitMax {
					n = rssItemsListLimitMax
				}
			}
			if n > 0 {
				limit = n
			}
		}

		var (
			args  []any
			conds []string
			idx   = 1
		)
		conds = append(conds, fmt.Sprintf("i.tenant_id = $%d", idx))
		args = append(args, tenantID)
		idx++
		conds = append(conds, fmt.Sprintf("i.user_id = $%d", idx))
		args = append(args, userID)
		idx++
		if feedIDStr != "" {
			fid, ferr := uuid.Parse(feedIDStr)
			if ferr != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "feed_id must be a uuid")
				return
			}
			conds = append(conds, fmt.Sprintf("i.feed_id = $%d", idx))
			args = append(args, fid)
			idx++
		}
		if strings.EqualFold(unreadRaw, "true") || unreadRaw == "1" {
			conds = append(conds, "i.read_at IS NULL")
		}

		args = append(args, limit)
		limitPlaceholder := fmt.Sprintf("$%d", idx)

		query := fmt.Sprintf(
			`SELECT i.id::text, i.tenant_id::text, i.user_id::text,
			        i.feed_id::text, COALESCE(f.name, ''),
			        i.guid, i.title, i.link,
			        COALESCE(i.summary, ''), COALESCE(i.author, ''),
			        i.published_at::text, i.read_at::text,
			        i.created_at::text
			   FROM homelab_rss_items i
			   LEFT JOIN homelab_rss_feeds f ON f.id = i.feed_id
			  WHERE %s
			  ORDER BY i.published_at DESC NULLS LAST, i.created_at DESC
			  LIMIT %s`,
			strings.Join(conds, " AND "),
			limitPlaceholder,
		)

		rows, err := pool.Pgx().Query(c.Request.Context(), query, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := []rssItemRow{}
		for rows.Next() {
			var (
				id, tID, uID, feedID, feedName, guid, title, link, createdAt string
				summary, author, publishedAt, readAt                         *string
			)
			if serr := rows.Scan(&id, &tID, &uID, &feedID, &feedName,
				&guid, &title, &link, &summary, &author,
				&publishedAt, &readAt, &createdAt); serr != nil {
				kernel.RespondError(c, serr)
				return
			}
			row := rssItemRow{
				ID:        id,
				TenantID:  tID,
				UserID:    uID,
				FeedID:    feedID,
				FeedName:  feedName,
				GUID:      guid,
				Title:     title,
				Link:      link,
				CreatedAt: createdAt,
			}
			if summary != nil {
				row.Summary = *summary
			}
			if author != nil {
				row.Author = *author
			}
			if publishedAt != nil {
				row.PublishedAt = *publishedAt
			}
			if readAt != nil {
				row.ReadAt = *readAt
			}
			out = append(out, row)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"items": out,
			"count": len(out),
			"limit": limit,
		})
	}
}

// ------------------------------------------------------------------
// POST /api/v1/homelab/rss/items/:id/read
// ------------------------------------------------------------------

// MarkHomelabRssItemRead sets read_at = NOW() on the given item
// if it belongs to the caller and isn't already read. Idempotent:
// POSTing on an already-read item returns 200 with {updated: 0}.
//
// MUST be registered BEFORE /feeds/:id DELETE so Gin's radix
// tree routes /items/:id/read to the correct handler.
func MarkHomelabRssItemRead(pool *db.Pool) gin.HandlerFunc {
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
			`UPDATE homelab_rss_items
			    SET read_at = NOW()
			  WHERE id = $1
			    AND tenant_id = $2
			    AND user_id = $3
			    AND read_at IS NULL`,
			id, tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondOK(c, gin.H{
			"item_id":       id.String(),
			"updated":       tag.RowsAffected(),
			"idempotent_ok": true,
		})
	}
}
