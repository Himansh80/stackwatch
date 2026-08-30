// Tier 11 Phase 3 — Self-Service Signup (PL3) — verify flow.
//
// VerifySignup is split into its own file because it's the
// chunkiest handler in the PL3 surface (3 SQL statements inside
// one transaction: tenant insert, user insert, signup update).
// Splitting keeps handlers_platform_signup.go under the
// 400-LOC cap while letting this file focus on the atomic
// tenant-creation choreography.
//
// Why a transaction:
//
//	A crash between the tenant INSERT and the user INSERT would
//	leave a tenant with no admin user — a half-created account
//	that no one can sign into. Wrapping the three statements in
//	a tx guarantees either all three succeed or none of them
//	do. The tx.Rollback on early-return is a defensive belt-
//	and-braces against an err path that doesn't commit —
//	pgx auto-rolls-back on tx.Close so it's redundant but
//	harmless.
//
// Why clear the verification_token on UPDATE:
//
//	Once consumed, the token should be unguessable / unplayable.
//	Setting it to NULL shrinks the partial index (so a replay
//	returns 404 instead of 409) AND removes the only thing that
//	could re-activate the row. Status='verified' is a separate
//	signal, but the token itself being NULL is what makes the
//	row truly single-use.
package handler

import (
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

// VerifySignup matches the verification_token, atomically
// creates the tenant + admin user (mirroring auth_signup.go::Signup),
// marks the signup verified, and returns a fresh JWT so the
// frontend can drop the user straight into the dashboard.
//
// On failure:
//
//	400 — missing/short token
//	404 — token does not exist (or already used)
//	409 — signup exists but is already verified (token was
//	      already consumed — the user can simply log in)
//
// On success: 201 + verifyResp{token (JWT), tenant_id, user_id, email, role}.
func VerifySignup(pool *db.Pool, issuer *auth.Issuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req verifyReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		token := strings.TrimSpace(req.Token)
		if token == "" {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "token is required")
			return
		}

		// Lookup the row by token. The partial index on
		// (verification_token) WHERE NOT NULL makes this O(1).
		var (
			signupID                               uuid.UUID
			email, passwordHash, fullName, orgName string
			status                                 string
		)
		err := pool.Pgx().QueryRow(c.Request.Context(), `
			SELECT id, email, password_hash, full_name, organization_name, status
			FROM platform_signups
			WHERE verification_token = $1
			LIMIT 1
		`, token).Scan(&signupID, &email, &passwordHash, &fullName, &orgName, &status)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				kernel.RespondErrorWithCode(c, http.StatusNotFound, "not_found",
					"verification token is invalid or has already been used")
				return
			}
			kernel.RespondError(c, err)
			return
		}
		if status != "pending_verification" {
			// Most likely 'verified' — already consumed.
			// Returning 409 (rather than 200 with a stale
			// tenant) lets the frontend distinguish "click
			// again" from "go log in".
			kernel.RespondErrorWithCode(c, http.StatusConflict, "conflict",
				"this signup has already been verified; sign in instead")
			return
		}

		// Atomically: create tenant + admin user + flip signup
		// to verified. All inside one tx so a crash mid-way
		// can't leave a tenant with no admin (or vice-versa).
		tenantID := uuid.New()
		userID := uuid.New()
		tx, err := pool.Pgx().Begin(c.Request.Context())
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer func() { _ = tx.Rollback(c.Request.Context()) }()

		if _, err := tx.Exec(c.Request.Context(), `
			INSERT INTO tenants (id, name, slug, plan, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		`, tenantID, orgName, slugify(orgName), kernel.PlanFree, kernel.TenantActive); err != nil {
			if isUniqueViolation(err) {
				// Slug collision — very unlikely (the slug
				// is derived from the org name which the
				// user just invented) but surface a clean 409.
				kernel.RespondErrorWithCode(c, http.StatusConflict, "conflict",
					"organization name conflicts with an existing tenant")
				return
			}
			kernel.RespondError(c, err)
			return
		}
		if _, err := tx.Exec(c.Request.Context(), `
			INSERT INTO users
				(id, tenant_id, email, full_name, password_hash, role, status,
				 email_verified, must_change_password, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, true, false, NOW(), NOW())
		`, userID, tenantID, email, fullName, passwordHash, kernel.RoleAdmin, kernel.UserActive); err != nil {
			if isUniqueViolation(err) {
				// Email collision against users.email — a
				// parallel /verify on the same token won the
				// race (or the email was created via another
				// signup channel in the gap). Tell the user
				// to log in instead.
				kernel.RespondErrorWithCode(c, http.StatusConflict, "conflict",
					"an account with this email already exists; sign in instead")
				return
			}
			kernel.RespondError(c, err)
			return
		}
		// Flip the signup row to verified — clear the token so
		// the partial index shrinks and a replay returns 404
		// (handled above) rather than 409. Stamp the FKs so
		// the admin dashboard can audit "who signed up and
		// when".
		if _, err := tx.Exec(c.Request.Context(), `
			UPDATE platform_signups
			SET status = 'verified',
			    verified_at = NOW(),
			    completed_at = NOW(),
			    tenant_id = $2,
			    admin_user_id = $3,
			    verification_token = NULL
			WHERE id = $1 AND status = 'pending_verification'
		`, signupID, tenantID, userID); err != nil {
			kernel.RespondError(c, err)
			return
		}
		if err := tx.Commit(c.Request.Context()); err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Issue a JWT so the frontend can drop the user straight
		// into the dashboard. Same issuer settings as
		// auth_signup.go::Signup.
		jwtStr, err := issuer.Issue(userID, tenantID, email, kernel.RoleAdmin, false)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondCreated(c, verifyResp{
			Token:    jwtStr,
			TenantID: tenantID.String(),
			UserID:   userID.String(),
			Email:    email,
			Role:     kernel.RoleAdmin,
		})
	}
}
