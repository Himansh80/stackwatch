// Tier 9 Phase 2 — SCIM Provisioning (Tier 9.2).
// HTTP route handlers for the 4 PROTECTED SCIM endpoints (RequireAuth):
//
//	GET    /api/v1/enterprise/scim/tokens       — ListSCIMTokens
//	POST   /api/v1/enterprise/scim/tokens       — CreateSCIMToken
//	DELETE /api/v1/enterprise/scim/tokens/:id   — RevokeSCIMToken
//	GET    /api/v1/enterprise/scim/sync-log     — ListSCIMSyncLog
//
// Plus the scimAuthMiddleware used by the 5 PUBLIC SCIM /v2/*
// endpoints (registered separately in handlers_scim_public.go).
//
// Security model:
//   - Protected endpoints honor tenant_id from JWT via the existing
//     tenantIDFromContext helper (tenant_context.go).
//   - SCIM tokens are hashed with bcrypt (cost 10) before insert; the
//     plaintext is returned to the caller EXACTLY ONCE on POST and
//     never persisted (we store the bcrypt hash only).
//   - The public /scim/v2/* endpoints in handlers_scim_public.go use
//     scimAuthMiddleware (defined here) which bcrypt-compares the
//     Bearer token against scim_tokens.token_hash.
//
// Every SCIM operation is logged to scim_sync_log (logSCIMSync helper,
// defined in handlers_scim_types.go) with status='success' or 'error' +
// error_message. Request bodies are never logged (PII risk).
package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// SCIMAuth validates the Bearer token in the Authorization header
// against the scim_tokens table. On success it stashes a
// scimClaims struct in the gin context (key scimClaimsCtxKey —
// defined in handlers_scim_types.go alongside the other shared
// types) so the downstream SCIM handlers can honor tenant_id +
// scopes. On failure it 401s and aborts.
//
// Exported (SCIMAuth, not scimAuthMiddleware) because the public
// router group is wired from cmd/api-gateway/routes_enterprise.go.
//
// bcrypt-compare is intentionally CPU-heavy (~50ms at cost 10) — this
// is by design (defends against brute force). The fire-and-forget
// last_used_at UPDATE runs after the response so a slow write doesn't
// add latency to the IdP's request.
//
// Volume note: real production usage sees 1-5 SCIM tokens per tenant
// total (Okta + Azure AD + Google Workspace each holding one). A real
// production-grade lookup would hash the supplied plaintext with a
// fast SHA-256 to get a candidate row id then bcrypt only that one
// row, but for that volume the table scan + per-row bcrypt is fine.
func SCIMAuth(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if len(h) < 8 || !strings.HasPrefix(h, "Bearer ") {
			kernel.RespondErrorWithCode(c, http.StatusUnauthorized, "unauthorized", "missing or malformed Authorization header")
			c.Abort()
			return
		}
		plain := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		if plain == "" {
			kernel.RespondErrorWithCode(c, http.StatusUnauthorized, "unauthorized", "empty bearer token")
			c.Abort()
			return
		}

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, tenant_id::text, token_hash, scopes, expires_at
			   FROM scim_tokens`)
		if err != nil {
			kernel.RespondError(c, err)
			c.Abort()
			return
		}
		defer rows.Close()

		var (
			matchedID      uuid.UUID
			matchedTenant  uuid.UUID
			matchedScopes  []string
			matchedExpires *time.Time
		)
		for rows.Next() {
			var idStr, tidStr, hash string
			var scopes []string
			var exp *time.Time
			if err := rows.Scan(&idStr, &tidStr, &hash, &scopes, &exp); err != nil {
				continue
			}
			if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err == nil {
				matchedID = uuid.MustParse(idStr)
				matchedTenant = uuid.MustParse(tidStr)
				matchedScopes = scopes
				matchedExpires = exp
				break
			}
		}
		if matchedID == uuid.Nil {
			kernel.RespondErrorWithCode(c, http.StatusUnauthorized, "unauthorized", "invalid bearer token")
			c.Abort()
			return
		}
		if matchedExpires != nil && matchedExpires.Before(time.Now()) {
			kernel.RespondErrorWithCode(c, http.StatusUnauthorized, "unauthorized", "bearer token expired")
			c.Abort()
			return
		}

		c.Set(scimClaimsCtxKey, &scimClaims{
			TokenID:  matchedID,
			TenantID: matchedTenant,
			Scopes:   matchedScopes,
		})

		// Fire-and-forget last_used_at bump. Use the request context
		// (so a cancelled request stops the write) but ignore the
		// error — a failed bump should never block the user.
		go func() {
			_, _ = pool.Pgx().Exec(c.Request.Context(),
				`UPDATE scim_tokens SET last_used_at = NOW() WHERE id = $1`, matchedID)
		}()

		c.Next()
	}
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/enterprise/scim/tokens
// ------------------------------------------------------------------

// ListSCIMTokens returns all SCIM tokens for the caller's tenant,
// newest first. NEVER returns plaintext — only metadata.
func ListSCIMTokens(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, tenant_id::text, name, scopes, expires_at::text, last_used_at::text, created_at::text
			   FROM scim_tokens WHERE tenant_id = $1 ORDER BY created_at DESC`,
			tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []scimTokenRow{}
		for rows.Next() {
			var id, tid, name, createdAt string
			var scopes []string
			var expiresAt, lastUsedAt *string
			if err := rows.Scan(&id, &tid, &name, &scopes, &expiresAt, &lastUsedAt, &createdAt); err != nil {
				continue
			}
			out = append(out, scimTokenRow{
				ID:         id,
				TenantID:   tid,
				Name:       name,
				Scopes:     scopes,
				ExpiresAt:  expiresAt,
				LastUsedAt: lastUsedAt,
				CreatedAt:  createdAt,
			})
		}
		kernel.RespondOK(c, gin.H{"tokens": out, "total": len(out)})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: POST /api/v1/enterprise/scim/tokens
// ------------------------------------------------------------------

// CreateSCIMToken generates a new SCIM token for the caller's tenant.
// The plaintext is returned ONCE (via the `plaintext_token` JSON
// field) and never persisted — only the bcrypt hash is stored.
func CreateSCIMToken(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req scimTokenReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Default scopes: users:read + users:write so the token is
		// useful for IdP-driven user lifecycle out of the box.
		scopes := req.Scopes
		if len(scopes) == 0 {
			scopes = []string{"users:read", "users:write"}
		}
		for _, s := range scopes {
			if !allowedSCIMScopes[s] {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
					fmt.Sprintf("unknown scope: %q", s))
				return
			}
		}

		plain, hash, err := generateSCIMToken()
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		// expires_at: NULL means "never expires". We bind only if the
		// caller supplied a value (string is RFC 3339). Parse errors
		// surface as 400.
		var expiresArg interface{}
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ExpiresAt)); err == nil && !t.IsZero() {
			expiresArg = t.UTC()
		} else if strings.TrimSpace(req.ExpiresAt) != "" {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"expires_at must be RFC 3339 (e.g. 2026-12-31T00:00:00Z)")
			return
		}

		var rowID, createdAt string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO scim_tokens (tenant_id, name, token_hash, scopes, expires_at)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id::text, created_at::text`,
			tenantID, strings.TrimSpace(req.Name), hash, scopes, expiresArg,
		).Scan(&rowID, &createdAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Only include expires_at in the response when non-NULL.
		var respExpires *string
		if t, ok := expiresArg.(time.Time); ok {
			s := t.Format(time.RFC3339)
			respExpires = &s
		}

		kernel.RespondCreated(c, scimTokenRow{
			ID:             rowID,
			TenantID:       tenantID.String(),
			Name:           strings.TrimSpace(req.Name),
			Scopes:         scopes,
			ExpiresAt:      respExpires,
			CreatedAt:      createdAt,
			PlaintextToken: plain, // one-shot
		})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: DELETE /api/v1/enterprise/scim/tokens/:id
// ------------------------------------------------------------------

// RevokeSCIMToken hard-deletes the row. Future requests with the
// revoked plaintext will 401 at scimAuthMiddleware because the
// bcrypt-compare won't find a row. We deliberately do NOT soft-delete
// here — a revoked token has no audit value beyond "this is revoked".
func RevokeSCIMToken(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM scim_tokens WHERE tenant_id = $1 AND id = $2`,
			tenantID, id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondOK(c, gin.H{"revoked": true, "id": id.String()})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/enterprise/scim/sync-log
// ------------------------------------------------------------------

// ListSCIMSyncLog returns recent SCIM operations for the caller's
// tenant. Supports ?since=<RFC3339> to filter by timestamp and
// ?limit=<1..200> to cap results (default 50).
func ListSCIMSyncLog(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		limit := 50
		if v := c.Query("limit"); v != "" {
			var n int
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n >= 1 && n <= 200 {
				limit = n
			}
		}
		since := strings.TrimSpace(c.Query("since"))
		var sinceTime time.Time
		if since != "" {
			t, err := time.Parse(time.RFC3339, since)
			if err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
					"since must be RFC 3339")
				return
			}
			sinceTime = t
		}

		args := []interface{}{tenantID}
		q := `SELECT id::text, tenant_id::text, op, COALESCE(external_id, ''), resource_type, status, COALESCE(error_message, ''), created_at::text
		        FROM scim_sync_log WHERE tenant_id = $1`
		if !sinceTime.IsZero() {
			args = append(args, sinceTime)
			q += fmt.Sprintf(" AND created_at >= $%d", len(args))
		}
		args = append(args, limit)
		q += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []scimSyncLogRow{}
		for rows.Next() {
			var r scimSyncLogRow
			if err := rows.Scan(&r.ID, &r.TenantID, &r.Op, &r.ExternalID, &r.ResourceType, &r.Status, &r.ErrorMessage, &r.CreatedAt); err != nil {
				continue
			}
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{"entries": out, "total": len(out), "limit": limit})
	}
}