// Tier 11 Phase 3 — Self-Service Signup (PL3).
//
// HTTP route handlers for the 3 PUBLIC signup endpoints:
//
//	POST /api/v1/platform/signup         — CreateSignup
//	POST /api/v1/platform/signup/verify  — VerifySignup
//	POST /api/v1/platform/signup/resend  — ResendSignup
//
// All three are PUBLIC (no JWT, no RequireAuth). The freshly-
// typed-in email address has no StackWatch credentials yet —
// the signup flow exists to mint them. Same pattern Tier 11
// PL1 uses for /deploy/install-script and Tier 9 uses for
// SCIM 2.0 (handlers_scim_public.go registered via
// mountEnterprisePublicRoutes).
//
// Shared types live in handlers_platform_signup_types.go (kept
// under the 400-LOC cap by the split). The in-memory rate-limit
// counter (signupRateLimit) is defined here as a package-level
// singleton — every CreateSignup call mutates it.
//
// Why these endpoints:
//
//   - CreateSignup is the entry point. It validates email +
//     password, hashes the password, mints a random
//     verification_token (32 bytes hex), inserts a
//     pending_verification row into platform_signups, and
//     returns the token in dev mode (no SMTP today).
//
//   - VerifySignup matches the token, atomically inserts the
//     tenant + admin user, marks the signup verified, and
//     returns a fresh JWT so the user is logged-in immediately.
//
//   - ResendSignup is the "I lost the email" recovery. It
//     never reveals whether the email exists (200 either way)
//     and rotates the token if the row is still pending.
//
// Security posture (matches spec §"PL3 — Self-Service Signup"):
//
//   - Server-side email regex (HTML5 is a hint, not a gate).
//   - auth.ValidatePassword (length, common-list, letter+digit).
//   - bcrypt cost-10 hash via auth.HashPassword (same as
//     auth_signup.go::Signup so a verified signup is
//     indistinguishable from an admin-issued signup at the
//     auth layer).
//   - Per-IP rate limit: 10 signups/day per source IP (the
//     spec's stated target). Implemented as an in-memory map
//     keyed by IP — adequate for Phase 3 single-instance
//     deploy; Phase 7 (PL7 — Rate Limiting) will replace it
//     with a Redis-backed token-bucket when multi-replica
//     lands.
//   - 401/403 are NOT applicable on a PUBLIC endpoint —
//     errors are 400 (validation), 409 (already pending /
//     already verified), or 429 (rate limit).
package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// =====================================================================
// Public endpoint: POST /api/v1/platform/signup
// =====================================================================

// CreateSignup accepts a new signup attempt and creates a
// pending-verification row.
//
// Body: signupReq{email, password, full_name, organization_name}.
// On success: 201 + signupResp{id, email, status, verification_token (dev), next_step}.
// On failure: 400 (validation), 409 (email already pending), 429 (rate limit).
func CreateSignup(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		// Rate limit FIRST so an attacker can't burn bcrypt CPU
		// by hammering the endpoint. The cap is per-IP per-day
		// per the spec's PL3 metric row.
		allowed, retryAfter := signupAllowed(ip)
		if !allowed {
			kernel.RespondRateLimited(c, kernel.ErrTooManyRequests, retryAfter)
			return
		}

		var req signupReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Lowercase email so the unique check below is
		// case-insensitive (the column is text, no citext).
		email := strings.ToLower(strings.TrimSpace(req.Email))

		// Server-side password validation (HTML5 minlength is
		// only a hint — a curl caller bypasses it).
		if err := auth.ValidatePassword(req.Password); err != nil {
			if ppe, ok := auth.IsPasswordPolicyError(err); ok {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, ppe.Code, ppe.Msg)
				return
			}
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}

		// Reject if there's a pending signup for this email.
		// Verified signups (the row exists with status=verified)
		// also reject — once a tenant is created for an email,
		// the user must log in, not re-signup.
		exists, err := hasPendingSignupForEmail(c.Request.Context(), email, pool)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if exists {
			kernel.RespondErrorWithCode(c, http.StatusConflict, "conflict",
				"a signup for this email is already pending or completed; check your inbox or sign in")
			return
		}

		// Hash + token + insert.
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		token := newVerificationToken()

		id := uuid.New()
		if _, err := pool.Pgx().Exec(c.Request.Context(), `
			INSERT INTO platform_signups
				(id, email, password_hash, full_name, organization_name,
				 verification_token, signup_ip, signup_user_agent, status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending_verification', NOW())
		`, id, email, hash, strings.TrimSpace(req.FullName), strings.TrimSpace(req.OrganizationName),
			token, ip, c.Request.UserAgent()); err != nil {
			// isUniqueViolation path: a parallel /signup call
			// won the race after our SELECT. Surface as 409.
			if isUniqueViolation(err) {
				kernel.RespondErrorWithCode(c, http.StatusConflict, "conflict",
					"a signup for this email is already pending or completed")
				return
			}
			kernel.RespondError(c, err)
			return
		}

		// Dev mode: surface the token in the response body so
		// an operator can paste it into /verify without SMTP.
		// A future PL5 build will flip signupDevMode=false once
		// Resend/SMTP is wired (see signupResp docs).
		var tokenOut *string
		if signupDevMode {
			t := token
			tokenOut = &t
		}
		kernel.RespondCreated(c, signupResp{
			ID:                id.String(),
			Email:             email,
			Status:            "pending_verification",
			VerificationToken: tokenOut,
			NextStep:          "verify_email",
		})
	}
}

// hasPendingSignupForEmail returns true if the email already has
// a row in platform_signups that is pending or verified. We treat
// "verified" the same as "pending" for the dedup check — once an
// account exists for an email, the user must log in (POST
// /auth/login) rather than creating a second tenant.
func hasPendingSignupForEmail(ctx context.Context, email string, pool *db.Pool) (bool, error) {
	var exists bool
	err := pool.Pgx().QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM platform_signups
			WHERE email = $1 AND status IN ('pending_verification', 'verified')
		)
	`, email).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// =====================================================================
// Public endpoint: POST /api/v1/platform/signup/verify
// =====================================================================
//
// VerifySignup lives in handlers_platform_signup_verify.go (its
// 3-statement tx makes it the chunkiest of the three handlers
// and a good split-out candidate).

// =====================================================================
// Public endpoint: POST /api/v1/platform/signup/resend
// =====================================================================

// ResendSignup rotates the verification_token on the latest
// pending signup for an email. The response is identical
// whether or not the email exists — we never leak account
// presence to a third party.
//
// On success (200): resendResp{status, verification_token (dev), next_step}.
// On failure: 400 (validation only — see above for the no-leak rule).
func ResendSignup(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req resendReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		email := strings.ToLower(strings.TrimSpace(req.Email))

		// Look up the most-recent pending signup for this email.
		// If found and still pending, rotate the token. Otherwise
		// return 200 with no token — indistinguishable from "no
		// such email".
		var signupID uuid.UUID
		err := pool.Pgx().QueryRow(c.Request.Context(), `
			SELECT id FROM platform_signups
			WHERE email = $1 AND status = 'pending_verification'
			ORDER BY created_at DESC
			LIMIT 1
		`, email).Scan(&signupID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// No pending row — also covers the
				// "email doesn't exist" case. Return 200
				// with status=ok, no token. The caller
				// can't tell which case it was.
				kernel.RespondOK(c, resendResp{
					Status:   "ok",
					NextStep: "verify_email",
				})
				return
			}
			kernel.RespondError(c, err)
			return
		}

		// Generate a new token + UPDATE.
		token := newVerificationToken()
		if _, err := pool.Pgx().Exec(c.Request.Context(), `
			UPDATE platform_signups
			SET verification_token = $2
			WHERE id = $1 AND status = 'pending_verification'
		`, signupID, token); err != nil {
			kernel.RespondError(c, err)
			return
		}

		var tokenOut *string
		if signupDevMode {
			t := token
			tokenOut = &t
		}
		kernel.RespondOK(c, resendResp{
			Status:            "ok",
			VerificationToken: tokenOut,
			NextStep:          "verify_email",
		})
	}
}

// =====================================================================
// Helpers live in handlers_platform_signup_helpers.go so this file
// stays under the 400-LOC cap.
// =====================================================================
