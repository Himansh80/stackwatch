package handler

import (
	"strings"

	"github.com/google/uuid"
)

// ------------------------------------------------------------------
// helpers
// ------------------------------------------------------------------

// parseSearchLimit parses a non-negative integer string, clamping
// silently to `capAt`. Returns searchError on non-numeric input.
// Empty string is treated as "no value" (returns 0, nil) so the
// caller can default.
func parseSearchLimit(raw string, capAt int) (int, error) {
	if raw == "" {
		return 0, nil
	}
	n := 0
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, &searchError{msg: "must be a non-negative integer"}
		}
		n = n*10 + int(ch-'0')
		if n > capAt {
			return capAt, nil
		}
	}
	return n, nil
}

// searchError is a tiny error type so we can pass through friendly
// strings without pulling in fmt just for one Sprintf.
type searchError struct{ msg string }

func (e *searchError) Error() string { return e.msg }

// escapeLikePattern escapes the wildcard chars so a search for
// "50% off" doesn't turn into "starts with 50". Backslash is the
// escape char (matches the rest of the homelab handlers).
func escapeLikePattern(q string) string {
	e := strings.ReplaceAll(q, `\`, `\\`)
	e = strings.ReplaceAll(e, "%", `\%`)
	e = strings.ReplaceAll(e, "_", `\_`)
	return "%" + e + "%"
}

// resolveSearchKinds parses the comma-separated ?kinds= value
// into a normalized slice. Empty input = all six kinds (the
// default). Unknown kinds return searchError so the frontend
// fails fast instead of silently dropping them.
func resolveSearchKinds(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		out := make([]string, 0, len(searchKindsCatalog))
		for _, k := range searchKindsCatalog {
			out = append(out, k.Kind)
		}
		return out, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := validSearchKindSet[p]; !ok {
			return nil, &searchError{msg: "unknown kind: " + p}
		}
		if _, dup := seen[p]; dup {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if len(out) == 0 {
		out = make([]string, 0, len(searchKindsCatalog))
		for _, k := range searchKindsCatalog {
			out = append(out, k.Kind)
		}
	}
	return out, nil
}

// kindEnabled is the O(6) "is this kind in the requested subset"
// check used by the SQL builder.
func kindEnabled(kinds []string, kind string) bool {
	for _, k := range kinds {
		if k == kind {
			return true
		}
	}
	return false
}

// buildSearchSQL constructs the parameterized UNION ALL with only
// the kinds the caller asked for. Always includes tenant_id +
// user_id in every WHERE clause.
//
// Returns SQL string and the args slice in the right order
// (placeholders: $1 = pattern, $2 = tenant_id, $3 = user_id,
// $4 = limit — reused across every branch).
func buildSearchSQL(pattern string, tenantID, userID uuid.UUID, kinds []string, limit int) (string, []any) {
	args := []any{pattern, tenantID, userID, limit}
	var branches []string
	for _, kind := range kinds {
		if branch, ok := searchUnionBranches[kind]; ok {
			branches = append(branches, branch)
		}
	}
	if len(branches) == 0 {
		// Defensive: should never hit because resolveSearchKinds
		// always returns ≥1 kind. If a future kind is added to
		// the catalog but not to searchUnionBranches, fall back
		// to a no-results query.
		return "SELECT NULL::text, NULL::text, NULL::text, NULL::text, NULL::text, 0 WHERE false", args
	}
	sql := strings.Join(branches, "\nUNION ALL\n")
	sql += "\nORDER BY 6 DESC, 3 ASC LIMIT $4"
	return sql, args
}

// buildSuggestionSQL is the lightweight version for typeahead.
// Drops snippet + score from the projection; limit is 8.
func buildSuggestionSQL(pattern string, tenantID, userID uuid.UUID, kinds []string, limit int) (string, []any) {
	args := []any{pattern, tenantID, userID, limit}
	var branches []string
	for _, kind := range kinds {
		if branch, ok := suggestionUnionBranches[kind]; ok {
			branches = append(branches, branch)
		}
	}
	if len(branches) == 0 {
		return "SELECT NULL::text, NULL::text, NULL::text, NULL::text WHERE false", args
	}
	sql := strings.Join(branches, "\nUNION ALL\n")
	sql += "\nORDER BY 4 DESC, 3 ASC LIMIT $4"
	return sql, args
}

// searchUnionBranches is the per-kind UNION arm. Each branch
// returns 6 columns (kind, id::text, title, snippet, url, score)
// in the SAME order so the outer ORDER BY (col 6 = score, col 3 =
// title) works across the union.
//
// Parameter binding: $1 = escaped LIKE pattern, $2 = tenant_id,
// $3 = user_id, $4 = limit. Adding a new kind here requires:
//  1. append a new searchKindMeta to searchKindsCatalog
//  2. add an arm to searchUnionBranches
//  3. add an arm to suggestionUnionBranches (smaller projection)
var searchUnionBranches = map[string]string{
	"notes": `
		SELECT 'notes'::text AS kind, n.id::text, n.title,
		       COALESCE(LEFT(n.body, 100), '') AS snippet,
		       '/homelab?tab=notes&id=' || n.id::text AS url,
		       (CASE WHEN n.title ILIKE $1 ESCAPE '\' THEN 3
		             WHEN n.body  ILIKE $1 ESCAPE '\' THEN 1
		             ELSE 0 END)::int AS score
		  FROM homelab_notes n
		 WHERE n.tenant_id = $2 AND n.user_id = $3
		   AND (n.title ILIKE $1 ESCAPE '\'
		        OR n.body  ILIKE $1 ESCAPE '\'
		        OR $1 = ANY(n.tags))`,
	"todos": `
		SELECT 'todos'::text AS kind, t.id::text, t.title,
		       COALESCE(LEFT(t.description, 100), '') AS snippet,
		       '/homelab?tab=todos&id=' || t.id::text AS url,
		       (CASE WHEN t.title       ILIKE $1 ESCAPE '\' THEN 3
		             WHEN t.description ILIKE $1 ESCAPE '\' THEN 1
		             ELSE 0 END)::int AS score
		  FROM homelab_todos t
		 WHERE t.tenant_id = $2 AND t.user_id = $3
		   AND (t.title ILIKE $1 ESCAPE '\'
		        OR t.description ILIKE $1 ESCAPE '\'
		        OR $1 = ANY(t.tags))`,
	"services": `
		SELECT 'services'::text AS kind, s.id::text, s.name,
		       '' AS snippet,
		       s.url AS url,
		       (CASE WHEN s.name ILIKE $1 ESCAPE '\' THEN 3 ELSE 0 END)::int AS score
		  FROM homelab_pinned_services s
		 WHERE s.tenant_id = $2 AND s.user_id = $3
		   AND s.name ILIKE $1 ESCAPE '\'`,
	"calendars": `
		SELECT 'calendars'::text AS kind, c.id::text, c.name,
		       '' AS snippet,
		       '/homelab?tab=calendar&id=' || c.id::text AS url,
		       (CASE WHEN c.name ILIKE $1 ESCAPE '\' THEN 3 ELSE 0 END)::int AS score
		  FROM homelab_calendars c
		 WHERE c.tenant_id = $2 AND c.user_id = $3
		   AND c.name ILIKE $1 ESCAPE '\'`,
	"downloads": `
		SELECT 'downloads'::text AS kind, d.id::text, d.name,
		       '' AS snippet,
		       '/homelab?tab=downloads&id=' || d.id::text AS url,
		       (CASE WHEN d.name ILIKE $1 ESCAPE '\' THEN 3 ELSE 0 END)::int AS score
		  FROM homelab_download_clients d
		 WHERE d.tenant_id = $2 AND d.user_id = $3
		   AND d.name ILIKE $1 ESCAPE '\'`,
	"media": `
		SELECT 'media'::text AS kind, m.id::text, m.name,
		       '' AS snippet,
		       '/homelab?tab=media&id=' || m.id::text AS url,
		       (CASE WHEN m.name ILIKE $1 ESCAPE '\' THEN 3 ELSE 0 END)::int AS score
		  FROM homelab_media_servers m
		 WHERE m.tenant_id = $2 AND m.user_id = $3
		   AND m.name ILIKE $1 ESCAPE '\'`,
}

// suggestionUnionBranches is the typeahead projection: 4 columns
// (kind, id, title, url) — no snippet, no score.
var suggestionUnionBranches = map[string]string{
	"notes": `
		SELECT 'notes'::text AS kind, n.id::text, n.title,
		       '/homelab?tab=notes&id=' || n.id::text AS url
		  FROM homelab_notes n
		 WHERE n.tenant_id = $2 AND n.user_id = $3
		   AND (n.title ILIKE $1 ESCAPE '\'
		        OR n.body  ILIKE $1 ESCAPE '\'
		        OR $1 = ANY(n.tags))`,
	"todos": `
		SELECT 'todos'::text AS kind, t.id::text, t.title,
		       '/homelab?tab=todos&id=' || t.id::text AS url
		  FROM homelab_todos t
		 WHERE t.tenant_id = $2 AND t.user_id = $3
		   AND (t.title ILIKE $1 ESCAPE '\'
		        OR t.description ILIKE $1 ESCAPE '\'
		        OR $1 = ANY(t.tags))`,
	"services": `
		SELECT 'services'::text AS kind, s.id::text, s.name,
		       s.url AS url
		  FROM homelab_pinned_services s
		 WHERE s.tenant_id = $2 AND s.user_id = $3
		   AND s.name ILIKE $1 ESCAPE '\'`,
	"calendars": `
		SELECT 'calendars'::text AS kind, c.id::text, c.name,
		       '/homelab?tab=calendar&id=' || c.id::text AS url
		  FROM homelab_calendars c
		 WHERE c.tenant_id = $2 AND c.user_id = $3
		   AND c.name ILIKE $1 ESCAPE '\'`,
	"downloads": `
		SELECT 'downloads'::text AS kind, d.id::text, d.name,
		       '/homelab?tab=downloads&id=' || d.id::text AS url
		  FROM homelab_download_clients d
		 WHERE d.tenant_id = $2 AND d.user_id = $3
		   AND d.name ILIKE $1 ESCAPE '\'`,
	"media": `
		SELECT 'media'::text AS kind, m.id::text, m.name,
		       '/homelab?tab=media&id=' || m.id::text AS url
		  FROM homelab_media_servers m
		 WHERE m.tenant_id = $2 AND m.user_id = $3
		   AND m.name ILIKE $1 ESCAPE '\'`,
}
