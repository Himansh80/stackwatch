// Tier 9 Phase 2 — SCIM Provisioning (Tier 9.2).
// Shared types + SCIM 2.0 access-control maps + cross-handler helpers
// for the 4 protected SCIM endpoints + 5 public SCIM /v2/* endpoints.
//
// 4 protected endpoints (RequireAuth):
//
//	GET    /api/v1/enterprise/scim/tokens       — list tokens (no plaintext)
//	POST   /api/v1/enterprise/scim/tokens       — create token (returns plaintext ONCE)
//	DELETE /api/v1/enterprise/scim/tokens/:id   — revoke token
//	GET    /api/v1/enterprise/scim/sync-log     — recent SCIM operations (filter ?since=X)
//
// 5 PUBLIC SCIM 2.0 endpoints (Bearer token auth, NOT JWT —
// implemented by scimAuthMiddleware in handlers_scim_protected.go):
//
//	GET    /scim/v2/Users                       — list users
//	POST   /scim/v2/Users                       — create user
//	PUT    /scim/v2/Users/:id                   — update user
//	DELETE /scim/v2/Users/:id                   — soft-disable user (status='disabled')
//	POST   /scim/v2/Groups                      — stub (Phase 2 minimum; full later)
//
// Helpers (defined here, used by both protected and public handlers):
//   - scimClaimsCtxKey / scimClaims / scimAuthMiddleware (auth context)
//   - scimClaimsFromContext / scimHasScope (downstream lookups)
//   - logSCIMSync (best-effort audit-row writer)
//   - generateSCIMToken (plaintext + bcrypt-hash pair generator)
//   - deriveSCIMFullName / deriveSCIMEmail (SCIM 2.0 → users row mappers)
//
// The public endpoints sit on a SEPARATE router group (NOT under
// RequireAuth) so IdPs can POST without a JWT. The scimAuthMiddleware
// validates the Bearer token against scim_tokens.token_hash
// (bcrypt-compare).
//
// Every SCIM operation writes a row to scim_sync_log with status
// 'success' on a clean op or 'error' + error_message on failure. We
// deliberately do NOT log request bodies — they can contain PII
// (emails, names).
package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/stackwatch/platform/internal/db"
)

// ------------------------------------------------------------------
// JSON row types — mirror the column shape and map directly to the
// `out` slice returned by each List* handler.
// ------------------------------------------------------------------

// scimTokenRow is the JSON shape for a single SCIM token returned by
// GET /scim/tokens + POST /scim/tokens. The plaintext token is
// INCLUDED ONLY on POST response (one-shot) — the list endpoint
// returns only metadata. `plaintext_token` is `omitempty` so the JSON
// shape is identical between list and create-once responses.
type scimTokenRow struct {
	ID         string   `json:"id"`
	TenantID   string   `json:"tenant_id"`
	Name       string   `json:"name"`
	Scopes     []string `json:"scopes"`
	ExpiresAt  *string  `json:"expires_at,omitempty"`
	LastUsedAt *string  `json:"last_used_at,omitempty"`
	CreatedAt  string   `json:"created_at"`
	// Plaintext token — only ever populated on POST response. NEVER
	// returned by GET. NEVER logged. NEVER persisted (we store the
	// bcrypt hash; this field is a transient one-shot read of the
	// generated value).
	PlaintextToken string `json:"plaintext_token,omitempty"`
}

// scimSyncLogRow is the JSON shape for a single audit row returned
// by GET /scim/sync-log. `error_message` is omitempty so successful
// rows don't carry a noisy null field.
type scimSyncLogRow struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	Op           string `json:"op"`
	ExternalID   string `json:"external_id,omitempty"`
	ResourceType string `json:"resource_type"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
	CreatedAt    string `json:"created_at"`
}

// ------------------------------------------------------------------
// Request types — body shape for POST endpoints.
// ------------------------------------------------------------------

// scimTokenReq is the JSON body for POST /scim/tokens. `name` is
// required (display name for the token); `scopes` is optional
// (default = ['users:read', 'users:write']); `expires_at` is
// optional (RFC 3339 timestamp; null = no expiry).
type scimTokenReq struct {
	Name      string   `json:"name"      binding:"required,min=1,max=64"`
	Scopes    []string `json:"scopes"    binding:"omitempty,dive,oneof=users:read users:write groups:read groups:write"`
	ExpiresAt string   `json:"expires_at" binding:"omitempty"`
}

// ------------------------------------------------------------------
// SCIM 2.0 payload shapes — used by the PUBLIC /scim/v2/* endpoints.
//
// We implement the MINIMUM subset that real IdPs (Okta, Azure AD,
// Google Workspace) actually exercise for user lifecycle. Group
// support is a stub for Phase 2 and lands fully in a later phase.
//
// Spec reference: RFC 7644 (SCIM Protocol) + RFC 7643 (SCIM Core
// Schema). We don't pull in a SCIM library — the JSON shapes are
// small enough to hand-roll.
// ------------------------------------------------------------------

// scimName mirrors the SCIM 2.0 `name` complex type (RFC 7643 §4.1.1).
// Only givenName + familyName are surfaced; honorificPrefix/Suffix etc.
// are accepted but ignored.
type scimName struct {
	GivenName  string `json:"givenName,omitempty"`
	FamilyName string `json:"familyName,omitempty"`
	Formatted  string `json:"formatted,omitempty"`
}

// scimEmail mirrors a single entry in SCIM 2.0 `emails` array
// (RFC 7643 §4.1.2).
type scimEmail struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

// scimUserResource is the SCIM 2.0 User resource (RFC 7643 §4.1).
// We accept a permissive superset of the spec — IdPs (Okta / Azure
// AD / Google Workspace) each add their own custom fields which we
// ignore gracefully. The fields we care about for provisioning are:
//
//	userName    — email-like login (Okta) OR the unique identifier
//	name        — givenName + familyName → users.full_name
//	emails      — primary email → users.email
//	active      — boolean → users.status ('active' if true, 'disabled' if false)
//	externalId  — IdP-side stable identifier (we store it on a
//	              synthetic metadata column; for Phase 2 we just log it
//	              into scim_sync_log.external_id)
type scimUserResource struct {
	Schemas  []string    `json:"schemas"`
	ID       string      `json:"id,omitempty"`
	UserName string      `json:"userName"`
	Name     *scimName   `json:"name,omitempty"`
	Emails   []scimEmail `json:"emails,omitempty"`
	Active   *bool       `json:"active,omitempty"`
	// ExternalID is the IdP-side stable identifier. SCIM spec calls
	// it `externalId` (camelCase). Stored on the user row as
	// `external_id` for Phase 2 — we keep it in scim_sync_log as well
	// so the sync-log endpoint can show "who was provisioned when".
	ExternalID string `json:"externalId,omitempty"`
}

// scimListResponse is the SCIM 2.0 ListResponse wrapper (RFC 7644 §3.4.2).
// `totalResults` is the count; `Resources` is the array of items;
// `itemsPerPage` + `startIndex` are pagination metadata.
type scimListResponse struct {
	Schemas      []string           `json:"schemas"`
	TotalResults int                `json:"totalResults"`
	ItemsPerPage int                `json:"itemsPerPage"`
	StartIndex   int                `json:"startIndex"`
	Resources    []scimUserResource `json:"Resources"`
}

// ------------------------------------------------------------------
// Access-control maps — whitelist valid enum values.
// ------------------------------------------------------------------

// allowedSCIMOps is the whitelist for the `op` column on
// scim_sync_log. The same set is enforced at the SQL CHECK level
// (well — we accept any text but our handlers only ever write these
// four values, so the JSON switch is the source of truth).
var allowedSCIMOps = map[string]bool{
	"create": true,
	"update": true,
	"delete": true,
	"list":   true,
}

// allowedSCIMResourceTypes is the whitelist for `resource_type`.
// Phase 2 only emits 'User'; 'Group' is reserved for later phases.
var allowedSCIMResourceTypes = map[string]bool{
	"User":  true,
	"Group": true,
}

// allowedSCIMScopes is the whitelist for SCIM token scopes. Same
// set as the binding tag on scimTokenReq.Scopes — kept here so
// handlers can check at runtime (e.g., "does this token have
// users:write?") without importing the binding package.
var allowedSCIMScopes = map[string]bool{
	"users:read":   true,
	"users:write":  true,
	"groups:read":  true,
	"groups:write": true,
}

// ------------------------------------------------------------------
// Shared context bag for SCIM-authenticated requests.
// Used by both handlers_scim_protected.go (the 4 protected handlers
// don't use it but it's exported) and handlers_scim_public.go (the
// 5 public endpoints that DO use it).
// ------------------------------------------------------------------

// scimClaimsCtxKey is the gin.Context key for the SCIM-bearer-token
// claims set by scimAuthMiddleware. Mirrors tenantCtxKey in
// tenant_context.go.
const scimClaimsCtxKey = "scim.claims"

// scimClaims is the per-request context bag for a SCIM-authenticated
// request. Set by scimAuthMiddleware after a successful bcrypt-compare.
type scimClaims struct {
	TokenID  uuid.UUID
	TenantID uuid.UUID
	Scopes   []string
}

// scimClaimsFromContext fetches the scimClaims bag set by
// scimAuthMiddleware. Returns (nil, false) when middleware didn't run.
func scimClaimsFromContext(c *gin.Context) (*scimClaims, bool) {
	v, ok := c.Get(scimClaimsCtxKey)
	if !ok {
		return nil, false
	}
	cl, ok := v.(*scimClaims)
	return cl, ok
}

// scimHasScope reports whether the caller's token carries the
// requested scope.
func scimHasScope(cl *scimClaims, scope string) bool {
	if cl == nil {
		return false
	}
	for _, s := range cl.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// ------------------------------------------------------------------
// SCIM 2.0 → users row mappers (used by the public handlers).
// ------------------------------------------------------------------

// deriveSCIMFullName composes a `full_name` from the SCIM name
// complex type. Falls back to `formatted` or to `userName` so we
// always have something to write into users.full_name.
func deriveSCIMFullName(u *scimUserResource) string {
	if u.Name != nil {
		g, f := strings.TrimSpace(u.Name.GivenName), strings.TrimSpace(u.Name.FamilyName)
		if g != "" || f != "" {
			return strings.TrimSpace(g + " " + f)
		}
		if u.Name.Formatted != "" {
			return u.Name.Formatted
		}
	}
	return strings.TrimSpace(u.UserName)
}

// deriveSCIMEmail returns the primary email from the SCIM emails
// array, falling back to the first email, falling back to userName
// (SCIM says userName MUST be email-like for User resources).
func deriveSCIMEmail(u *scimUserResource) string {
	for _, e := range u.Emails {
		if e.Primary && e.Value != "" {
			return e.Value
		}
	}
	for _, e := range u.Emails {
		if e.Value != "" {
			return e.Value
		}
	}
	return strings.TrimSpace(u.UserName)
}

// ------------------------------------------------------------------
// generateSCIMToken returns a (plaintext, hash) pair. The plaintext
// is "scim_" + base64url(32 random bytes) — URL-safe and short enough
// to paste into an IdP. The hash is bcrypt(plaintext) at cost 10
// (matches the Tier 0 api_keys policy).
// ------------------------------------------------------------------
func generateSCIMToken() (plaintext string, hash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("rand: %w", err)
	}
	plaintext = "scim_" + base64.RawURLEncoding.EncodeToString(raw)
	hb, err := bcrypt.GenerateFromPassword([]byte(plaintext), 10)
	if err != nil {
		return "", "", fmt.Errorf("bcrypt: %w", err)
	}
	return plaintext, string(hb), nil
}

// ------------------------------------------------------------------
// logSCIMSync writes one row to scim_sync_log. Best-effort — a
// failure here MUST NOT block the user-facing response, so we attach
// the warning to an X-SCIM-Log-Warning header instead of 5xx'ing.
//
// `externalID` is the IdP-side identifier (Okta `00uxxx`, Azure GUID,
// etc.) — safe to log (PII risk is low: this is a stable opaque id,
// not the user's email/name).
// ------------------------------------------------------------------
func logSCIMSync(c *gin.Context, pool *db.Pool, tenantID uuid.UUID, op, externalID, resourceType, status, errMsg string) {
	_, err := pool.Pgx().Exec(c.Request.Context(),
		`INSERT INTO scim_sync_log (tenant_id, op, external_id, resource_type, status, error_message)
		 VALUES ($1, $2, NULLIF($3, ''), $4, $5, NULLIF($6, ''))`,
		tenantID, op, externalID, resourceType, status, errMsg)
	if err != nil {
		c.Header("X-SCIM-Log-Warning", err.Error())
	}
}
