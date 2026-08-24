// Tier 9 Phase 1 — SSO Foundation (Tier 9.1).
// HTTP route handlers for the 6 protected SSO endpoints (RequireAuth).
//
//	GET    /api/v1/enterprise/sso/providers        — ListSSOProviders
//	POST   /api/v1/enterprise/sso/providers        — CreateSSOProvider
//	PATCH  /api/v1/enterprise/sso/providers/:id    — UpdateSSOProvider
//	DELETE /api/v1/enterprise/sso/providers/:id    — DeleteSSOProvider (soft)
//	POST   /api/v1/enterprise/sso/test             — TestSSOProvider
//	GET    /api/v1/enterprise/sso/connections      — ListSSOConnections
//
// The 3 PUBLIC callback endpoints (no JWT) live in
// handlers_sso_callbacks.go (InitiateSSO, OIDCCallback, SAMLCallback).
// Types live in handlers_sso_types.go. Helpers (encryption, OIDC
// discovery, SAML AuthnRequest builder, JIT provisioning, SAML XML
// parsing) live in handlers_sso_helpers.go / handlers_sso_jit.go.
//
// Encryption uses the existing internal/handler.encryptSecret helper
// (AES-GCM via CREDENTIALS_MASTER_KEY env) so we never store
// plaintext client_secret or SAML signing keys in the JSONB config.
package handler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListSSOProviders returns all SSO providers for the caller's tenant,
// newest first. Config jsonb is summarized to non-secret fields so
// admins can recognize a provider without us leaking secrets.
func ListSSOProviders(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, tenant_id::text, type, name, enabled, config, created_at::text
			   FROM sso_providers
			  WHERE tenant_id = $1
			  ORDER BY created_at DESC`, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []ssoProviderRow{}
		for rows.Next() {
			var id, tid, typ, name, createdAt string
			var enabled bool
			var rawConfig []byte
			if err := rows.Scan(&id, &tid, &typ, &name, &enabled, &rawConfig, &createdAt); err != nil {
				continue
			}
			out = append(out, ssoProviderRow{
				ID:            id,
				TenantID:      tid,
				Type:          typ,
				Name:          name,
				Enabled:       enabled,
				ConfigSummary: summarizeSSOConfig(typ, rawConfig),
				CreatedAt:     createdAt,
			})
		}
		kernel.RespondOK(c, gin.H{
			"providers": out,
			"total":     len(out),
		})
	}
}

// CreateSSOProvider validates the config, encrypts sensitive fields
// (client_secret for OIDC, x509_cert for SAML), inserts the row,
// and returns the new provider id.
func CreateSSOProvider(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req ssoProviderReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if !allowedSSOProviderTypes[req.Type] {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		encrypted, err := encryptSSOConfig(req.Type, req.Config)
		if err != nil {
			kernel.RespondError(c, fmt.Errorf("encrypt config: %w", err))
			return
		}
		var rowID, createdAt string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO sso_providers (tenant_id, type, name, config, enabled)
			 VALUES ($1, $2, $3, $4::jsonb, true)
			 RETURNING id::text, created_at::text`,
			tenantID, req.Type, strings.TrimSpace(req.Name), string(encrypted),
		).Scan(&rowID, &createdAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, ssoProviderRow{
			ID:            rowID,
			TenantID:      tenantID.String(),
			Type:          req.Type,
			Name:          strings.TrimSpace(req.Name),
			Enabled:       true,
			ConfigSummary: summarizeSSOConfig(req.Type, encrypted),
			CreatedAt:     createdAt,
		})
	}
}

// UpdateSSOProvider patches name / enabled / config. Re-encrypts
// config if provided.
func UpdateSSOProvider(pool *db.Pool) gin.HandlerFunc {
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
		var req ssoProviderPatchReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if req.Name == nil && req.Enabled == nil && req.Config == nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Fetch existing type (for re-encryption).
		var existingType string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT type FROM sso_providers WHERE tenant_id = $1 AND id = $2`,
			tenantID, id,
		).Scan(&existingType)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				kernel.RespondError(c, kernel.ErrNotFound)
			} else {
				kernel.RespondError(c, err)
			}
			return
		}

		sets := []string{}
		args := []interface{}{}
		idx := 1
		if req.Name != nil {
			sets = append(sets, fmt.Sprintf("name = $%d", idx))
			args = append(args, strings.TrimSpace(*req.Name))
			idx++
		}
		if req.Enabled != nil {
			sets = append(sets, fmt.Sprintf("enabled = $%d", idx))
			args = append(args, *req.Enabled)
			idx++
		}
		if req.Config != nil {
			encrypted, err := encryptSSOConfig(existingType, req.Config)
			if err != nil {
				kernel.RespondError(c, fmt.Errorf("encrypt config: %w", err))
				return
			}
			sets = append(sets, fmt.Sprintf("config = $%d::jsonb", idx))
			args = append(args, string(encrypted))
			idx++
		}
		args = append(args, tenantID, id)
		q := fmt.Sprintf(`UPDATE sso_providers SET %s
		                   WHERE tenant_id = $%d AND id = $%d`,
			strings.Join(sets, ", "), idx, idx+1)
		tag, err := pool.Pgx().Exec(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondOK(c, gin.H{"id": id.String(), "updated": true})
	}
}

// DeleteSSOProvider sets enabled=false (soft delete) so we keep the
// audit trail of which users signed in via this provider.
func DeleteSSOProvider(pool *db.Pool) gin.HandlerFunc {
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
			`UPDATE sso_providers SET enabled = false
			  WHERE tenant_id = $1 AND id = $2`, tenantID, id,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondOK(c, gin.H{"id": id.String(), "deleted": true, "soft": true})
	}
}

// TestSSOProvider validates an IdP's connectivity without doing a full
// auth flow. For OIDC it fetches the discovery document; for SAML it
// parses metadata + verifies the X509 cert is decryptable.
func TestSSOProvider(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req ssoTestReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		pid, _ := uuid.Parse(req.ProviderID)

		var typ string
		var rawConfig []byte
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT type, config FROM sso_providers
			  WHERE tenant_id = $1 AND id = $2`,
			tenantID, pid,
		).Scan(&typ, &rawConfig)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				kernel.RespondError(c, kernel.ErrNotFound)
			} else {
				kernel.RespondError(c, err)
			}
			return
		}
		details, err := testSSOConfigConnectivity(typ, rawConfig)
		if err != nil {
			kernel.RespondOK(c, gin.H{
				"ok": false, "type": typ, "details": details, "error": err.Error(),
			})
			return
		}
		kernel.RespondOK(c, gin.H{"ok": true, "type": typ, "details": details})
	}
}

// ListSSOConnections returns the caller's SSO connections (one row
// per provider they have signed in via).
func ListSSOConnections(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT sc.id::text, sc.tenant_id::text, sc.user_id::text,
			        sc.provider_id::text, sp.name, sc.subject,
			        sc.created_at::text, sc.last_used_at::text
			   FROM sso_connections sc
			   JOIN sso_providers sp ON sp.id = sc.provider_id
			  WHERE sc.user_id = $1 AND sc.tenant_id = $2
			  ORDER BY sc.last_used_at DESC NULLS LAST`, userID, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []ssoConnectionRow{}
		for rows.Next() {
			var r ssoConnectionRow
			var lastUsed *string
			if err := rows.Scan(&r.ID, &r.TenantID, &r.UserID, &r.ProviderID,
				&r.ProviderName, &r.Subject, &r.CreatedAt, &lastUsed); err == nil {
				r.LastUsedAt = lastUsed
				out = append(out, r)
			}
		}
		kernel.RespondOK(c, gin.H{"connections": out, "total": len(out)})
	}
}
