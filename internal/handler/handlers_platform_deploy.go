// Tier 11 Phase 1 — Push-Button Deploy (PL1).
// HTTP route handlers for the 3 deploy endpoints:
//
//	POST /api/v1/platform/deploy/install-token   — CreateInstallToken (protected)
//	GET  /api/v1/platform/deploy/install-script  — GetInstallScript  (PUBLIC — see mountPlatformPublicRoutes)
//	GET  /api/v1/platform/deploy/stats           — GetInstallStats   (protected)
//
// Shared types live in handlers_platform_deploy_types.go (kept
// under the 400-LOC cap by the split). The platform helper
// (token generation, lookup, mark-used) lives in
// internal/platform/deploy.go.
//
// Why two of three are protected and one is public:
//   The create-token + stats endpoints are tenant-scoped — only
//   the tenant's admins can mint new tokens or see install
//   activity. The install-script endpoint MUST be public because
//   a freshly-installed Linux box has no JWT yet; the token
//   itself IS the credential. Same pattern Tier 9 uses for SCIM
//   2.0 (handlers_scim_public.go registered separately via
//   mountEnterprisePublicRoutes).
//
// Token lifecycle (mirrors the spec PL1 + §"Migration 042: PL1
// tables"):
//   1. Admin calls POST /install-token with optional label.
//   2. Handler returns 201 + {token, expires_at, id} where
//      `token` is the plaintext shown ONCE.
//   3. Admin pastes the rendered curl one-liner into a fresh
//      Linux box, which calls GET /install-script with the
//      token in the query string.
//   4. PUBLIC endpoint bcrypt-validates the token, marks
//      used_at = now() + used_by_ip = caller, and returns the
//      bash script with the token + backend baked in.
//   5. Replays of step 4 get 410 Gone — the token is one-time use.
//   6. After 1h un-used tokens are rejected by LookupInstallToken's
//      "expires_at > now()" filter.
package handler

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
	"github.com/stackwatch/platform/internal/platform"
)

// labelPattern is compiled once at package init. We use a
// pre-compiled regex for the inbound check (handler-side) but a
// hand-rolled ASCII scan in platform.TrimLabel for the storage
// trim — see that file for why.
var labelPattern = regexp.MustCompile(allowedLabelChars)

// ------------------------------------------------------------------
// Protected endpoint: POST /api/v1/platform/deploy/install-token
// ------------------------------------------------------------------

// CreateInstallToken mints a fresh one-time install token for
// the caller's tenant. The plaintext is returned ONCE in the
// response body and is never persisted (only the bcrypt hash
// lives in deploy_install_tokens).
//
// Per spec PL1: token expires in 1 hour (installTokenLifetime
// in handlers_platform_deploy_types.go). The handler does NOT
// expose the lifetime as a request field — Phase 5 (Tenant
// Limits) will add a plan-based override.
func CreateInstallToken(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		// Body is optional (label can be empty).
		var req installTokenReq
		if c.Request.ContentLength > 0 {
			if err := c.ShouldBindJSON(&req); err != nil {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
		}

		// Validate label against the regex. TrimLabel below is
		// more permissive (drops invalid chars) but we 400 here
		// when the caller supplied a value that doesn't match
		// the policy at all — operators want explicit feedback
		// rather than silent truncation.
		cleanLabel := platform.TrimLabel(req.Label)
		if strings.TrimSpace(req.Label) != "" && cleanLabel == "" {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				`label may only contain letters, digits, space, dash, underscore (max 64 chars)`)
			return
		}
		// Belt-and-braces: also enforce the canonical regex so a
		// future TrimLabel behavior change doesn't accidentally
		// widen the allowlist.
		if req.Label != "" && !labelPattern.MatchString(req.Label) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				`label may only contain letters, digits, space, dash, underscore (max 64 chars)`)
			return
		}

		plain, expiresAt, err := platform.GenerateInstallToken(c.Request.Context(), tenantID, cleanLabel, installTokenLifetime, pool)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondCreated(c, installTokenRow{
			TenantID:  tenantID.String(),
			CreatedAt: time.Now().UTC(),
			ExpiresAt: expiresAt,
			Label:     labelOrNil(cleanLabel),
			Token:     plain, // one-shot plaintext
		})
	}
}

// labelOrNil returns a pointer to the trimmed label when non-
// empty, nil otherwise. Mirrors how the SQL column treats NULL.
func labelOrNil(s string) *string {
	if s == "" {
		return nil
	}
	v := s
	return &v
}

// ------------------------------------------------------------------
// PUBLIC endpoint: GET /api/v1/platform/deploy/install-script
// ------------------------------------------------------------------

// GetInstallScript is the PUBLIC endpoint that returns a bash
// one-liner with the token baked in. The freshly-installed Linux
// box hits this URL with NO JWT — the install token itself is
// the credential (Datadog-style: the operator pastes the curl
// snippet the admin generated).
//
// Query params:
//   token   — required, the plaintext returned by POST /install-token
//   backend — required, the URL the installer will POST heartbeats to
//             (e.g. https://smarthomelab.fun or http://192.168.0.115:8080)
//
// On success:
//   200 with a Content-Type: text/x-shellscript body containing
//   the bash script, then the row is marked used_at = now() +
//   used_by_ip = c.ClientIP().
//
// On failure:
//   400 — missing token / backend
//   410 — token not found (already used, expired, or never existed)
func GetInstallScript(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.Query("token"))
		backend := strings.TrimSpace(c.Query("backend"))
		if token == "" || backend == "" {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"both token and backend query params are required")
			return
		}

		row, err := platform.LookupInstallToken(c.Request.Context(), token, pool)
		if err != nil {
			if errors.Is(err, platform.ErrTokenNotFound) {
				kernel.RespondErrorWithCode(c, http.StatusGone, "gone",
					"install token has already been used, expired, or never existed")
				return
			}
			kernel.RespondError(c, err)
			return
		}

		// Mark the token used NOW (best-effort). A failure here
		// doesn't block the response — the next replay attempt
		// will see used_at IS NOT NULL and 410 from
		// LookupInstallToken's filter.
		ip := c.ClientIP()
		_ = platform.MarkInstallTokenUsed(c.Request.Context(), row.ID, ip, pool)

		// Render the bash snippet. We DO NOT escape the
		// backend URL because it's a URL the operator just
		// typed — if it's malformed the curl line will fail
		// on the box, not here. We DO single-quote-wrap the
		// token so an unlikely "swi_<weird>" value can't
		// break out of the quoting.
		script := renderInstallScript(token, backend, row.ExpiresAt.Format(time.RFC3339))

		c.Header("Content-Type", "text/x-shellscript; charset=utf-8")
		c.Header("Content-Disposition", `attachment; filename="install.sh"`)
		c.String(http.StatusOK, script)
	}
}

// renderInstallScript produces the bash one-liner returned by
// GET /install-script. Kept as its own function so the handler
// stays readable + the format is unit-testable.
func renderInstallScript(token, backend, expiresAt string) string {
	// Single-quote-wrap the token so an embedded single-quote
	// would break the bash — replace ' with '\''
	safeToken := strings.ReplaceAll(token, "'", `'\''`)
	return fmt.Sprintf(`#!/bin/bash
# StackWatch Agent Installer
# Backend: %s
# Token: %s (one-time use, expires %s)
set -euo pipefail
curl -fsSL %s/installers/linux/amd64/installer -o /tmp/stackwatch-installer
chmod +x /tmp/stackwatch-installer
sudo /tmp/stackwatch-installer install --backend '%s' --token '%s'
`, backend, token, expiresAt, backend, backend, safeToken)
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/platform/deploy/stats
// ------------------------------------------------------------------

// GetInstallStats returns aggregated install-token statistics
// for the caller's tenant so the DeploySection KPI strip can
// render the four numbers in one round-trip.
//
// Counts:
//   installs_24h  — used_at >= now() - 24h
//   installs_7d   — used_at >= now() -  7d
//   installs_30d  — used_at >= now() - 30d
//   tokens_active — used_at IS NULL AND expires_at > now()
//   tokens_total  — all rows for the tenant (cumulative)
func GetInstallStats(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		var s installStats
		row := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT
			   count(*) FILTER (WHERE used_at >= now() - interval '24 hours') AS installs_24h,
			   count(*) FILTER (WHERE used_at >= now() - interval  '7 days')   AS installs_7d,
			   count(*) FILTER (WHERE used_at >= now() - interval '30 days')   AS installs_30d,
			   count(*) FILTER (WHERE used_at IS NULL AND expires_at > now())  AS tokens_active,
			   count(*)                                                          AS tokens_total
			 FROM deploy_install_tokens
			 WHERE tenant_id = $1`,
			tenantID)
		if err := row.Scan(&s.Installs24h, &s.Installs7d, &s.Installs30d, &s.TokensActive, &s.TokensTotal); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, s)
	}
}