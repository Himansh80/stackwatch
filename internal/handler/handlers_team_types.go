// Tier 7 Phase 3 — Team/Collab (D12). Shared types + access-control maps.
//
// All queries honor tenant_id from the JWT. Permission and context_type
// enums are enforced at the API edge so the columns never hold garbage.
// The three tables backing this surface (dashboard_shares,
// timeline_comments, mentions) are created by the idempotent migration
// migrations/038_team.sql.
//
// Permission values:
//   - 'view' — read-only access to the shared dashboard
//   - 'edit' — can save dashboard layout / panel edits
//
// Context types for mentions (used by the UI to render the right
// deep-link and badge):
//   - 'incident'  → /incidents/{context_id}
//   - 'notebook'  → /notebooks  (the notebook the mention lives in)
//   - 'dashboard' → /dashboard?d={context_id}
//   - 'comment'   → /incidents/{context_id}#comments (timeline thread)
package handler

// allowedSharePermissions — defense in depth at the API edge. Spec
// defines these two values; anything else is silently dropped from
// the WRITE so the column never holds garbage.
var allowedSharePermissions = map[string]bool{
	"view": true, "edit": true,
}

// allowedMentionContextTypes — the context_type values the
// /api/v1/annotations/mentions endpoint is willing to render. New
// context types require a code change here + the deep-link map on
// the frontend; the partial index on read_at IS NULL is the same
// either way.
var allowedMentionContextTypes = map[string]bool{
	"incident":  true,
	"notebook":  true,
	"dashboard": true,
	"comment":   true,
}

// sharedDashboardRow is the JSON shape returned for a single
// dashboard_shares entry, enriched with the dashboard's display name
// + the grantor's full_name so the UI can render "Alice shared
// Latency — API Tier with you".
type sharedDashboardRow struct {
	DashboardID   string `json:"dashboard_id"`
	DashboardName string `json:"dashboard_name"`
	Permission    string `json:"permission"`
	OwnerName     string `json:"owner_name"`
	CreatedAt     string `json:"created_at"`
}

// shareDashboardReq is the JSON shape for POST /dashboards/share.
// All fields required — permission defaults to 'view' if the
// client sends the empty string.
type shareDashboardReq struct {
	DashboardID string `json:"dashboard_id" binding:"required,uuid"`
	UserID      string `json:"user_id"       binding:"required,uuid"`
	Permission  string `json:"permission"    binding:"omitempty,oneof=view edit"`
}

// mentionRow is the JSON shape returned for a single mentions entry,
// with the mentioning user's full_name hydrated so the UI can render
// "@alice mentioned you in …" without a second lookup.
type mentionRow struct {
	ID               string  `json:"id"`
	MentionedUserID  string  `json:"mentioned_user_id"`
	MentioningUserID string  `json:"mentioning_user_id"`
	MentioningName   string  `json:"mentioning_name"`
	ContextType      string  `json:"context_type"`
	ContextID        string  `json:"context_id"`
	ReadAt           *string `json:"read_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
}

// timelineCommentRow is the JSON shape returned for a single
// timeline_comments entry, with the author's full_name hydrated so
// the UI can render "alice • 2 minutes ago" without a second lookup.
type timelineCommentRow struct {
	ID         string  `json:"id"`
	IncidentID *string `json:"incident_id,omitempty"`
	UserID     string  `json:"user_id"`
	UserName   string  `json:"user_name"`
	Body       string  `json:"body"`
	CreatedAt  string  `json:"created_at"`
}

// timelineCommentReq is the JSON shape for POST /timeline/comments.
// incident_id is optional — a comment with no incident is allowed
// and surfaces in the audit-trail mode.
type timelineCommentReq struct {
	IncidentID string `json:"incident_id" binding:"omitempty,uuid"`
	Body       string `json:"body"        binding:"required,min=1,max=8192"`
}
