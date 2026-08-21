package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/kernel"
)

// LoginRequest is the JSON body for POST /auth/login.
//
// NOTE: Password has NO min-length binding tag here — we deliberately
// accept any non-empty password so users with legacy 8-char passwords
// (created before the policy upgrade) can still sign in. After
// verifying the password succeeds, if the password is < 10 chars we
// flip `must_change_password=true` in the DB, which forces them
// through /auth/change-password on the next request. The new policy
// itself is enforced inside ChangePassword.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is the JSON body for successful login.
type LoginResponse struct {
	Token  string `json:"token"`
	User   any    `json:"user"`
	Tenant any    `json:"tenant"`
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	// 1. Email format check (gin binding already validates this, but we
	//    want a more user-friendly error code than "bad_request").
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	ip := clientIP(c)

	// Rate limit BEFORE doing any DB lookup. 5 attempts per ip+email per 5min.
	if wait, err := checkLoginRateLimit(ip, email); err != nil {
		kernel.RespondRateLimited(c, err, int(wait.Round(time.Second).Seconds()))
		return
	}

	user, tenant, err := h.lookupUserAndTenant(c.Request.Context(), email)
	if err != nil {
		// Distinguish: email-not-registered vs other DB errors.
		// Product decision (see commit message): tell the user their
		// email is not registered so they get a clear next step.
		// The standard anti-enumeration mitigation is the rate
		// limiter above; we accept the trade-off here because
		// this is a self-hosted product and the operator wants
		// honest error feedback.
		if errors.Is(err, kernel.ErrNotFound) {
			kernel.RespondErrorWithCode(c, http.StatusNotFound, "email_not_found",
				"No account exists with that email.")
			return
		}
		kernel.RespondError(c, err)
		return
	}
	if !user.IsActive() {
		// Account exists but is disabled / suspended.
		kernel.RespondErrorWithCode(c, http.StatusForbidden, "account_disabled",
			"This account is disabled. Contact your administrator.")
		return
	}
	if !checkHash(user.PasswordHash, req.Password) {
		// Failed login → audit + brute-force counter
		h.auditLog(c.Request.Context(), &user.TenantID, &user.ID, "login.failed", user.Email, ip, map[string]any{"reason": "bad_password"})
		// Increment failed-login counter; if it crosses 10 in last hour, log a security event.
		count, _ := h.recordFailedLogin(c.Request.Context(), user.ID)
		if count >= 10 {
			h.auditLog(c.Request.Context(), &user.TenantID, &user.ID, "login.bruteforce_suspected", user.Email, ip, map[string]any{"failed_count_1h": count})
		}
		// Tell the user the password is wrong (not the email).
		kernel.RespondErrorWithCode(c, http.StatusUnauthorized, "bad_password",
			"Wrong password. Try again or reset your password.")
		return
	}

	// Reset failed-login counter on successful login.
	_, _ = h.pool.Pgx().Exec(c.Request.Context(),
		`DELETE FROM user_failed_logins WHERE user_id = $1`, user.ID)

	// Force-change-password upgrade: if the verified password is below
	// the new minimum length, set must_change_password=true so every
	// subsequent request is blocked by RequireAuth until the user
	// calls /auth/change-password with a policy-compliant new password.
	mustChange := user.MustChangePW
	if utf8.RuneCountInString(req.Password) < auth.MinLength {
		if _, err := h.pool.Pgx().Exec(c.Request.Context(),
			`UPDATE users SET must_change_password=true WHERE id=$1`, user.ID,
		); err == nil {
			mustChange = true
			h.auditLog(c.Request.Context(), &user.TenantID, &user.ID,
				"password.policy_upgrade_force_reset", user.Email, ip,
				map[string]any{"reason": "password_below_minimum_length"})
		}
	}

	tok, err := h.issuer.Issue(user.ID, user.TenantID, user.Email, user.Role, mustChange)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}

	// Update last_login_at + audit success.
	_, _ = h.pool.Pgx().Exec(c.Request.Context(),
		`UPDATE users SET last_login_at=NOW() WHERE id=$1`, user.ID)
	h.auditLog(c.Request.Context(), &user.TenantID, &user.ID, "login.success", user.Email, ip, nil)

	kernel.RespondOK(c, LoginResponse{Token: tok, User: user, Tenant: tenant})
}
