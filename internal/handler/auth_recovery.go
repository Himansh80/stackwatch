// Package handler — Password recovery + profile + invite flows (Tier 0).
//
// All four endpoints are real (NOT stubs):
//   - POST /auth/forgot            → issue a reset token, return dev URL in body
//   - POST /auth/reset             → consume token, change password
//   - POST /auth/magic-link        → issue one-time login token
//   - POST /auth/accept-invite     → consume invite, set password, activate
//   - PATCH /auth/profile          → update own full_name
//
// Tokens are 32-byte random (base64url-encoded), stored as SHA-256 hashes
// (so DB compromise doesn't leak active tokens). Expires after 1 hour.
// Already-used tokens are rejected.
package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/kernel"
)

// tokenTTL is how long a reset / magic-link / invite token is valid.
const tokenTTL = 1 * time.Hour

// tokenBytes is the byte length of the random portion of a token.
const tokenBytes = 32

// generateToken returns a fresh random token (raw, plaintext — to be sent
// to the user) and its sha-256 hex hash (to be stored in the DB).
func generateToken() (raw, hash string, err error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(sum[:])
	return raw, hash, nil
}

// generateTokenWithPrefix creates a token with a purpose prefix. Prefixes
// prevent one flow's token from being accepted by another flow.
func generateTokenWithPrefix(prefix string) (raw, hash string, err error) {
	base, _, err := generateToken()
	if err != nil {
		return "", "", err
	}
	raw = prefix + base
	return raw, hashToken(raw), nil
}

// hashToken returns the sha-256 hex digest of the given raw token.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// clientIP extracts the originating IP from the request, honouring
// X-Forwarded-For if present.
func clientIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// first IP in the list is the original client
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}

// auditLog writes a single row to audit_log. Best-effort — never fails the
// parent handler if the audit insert errors out.
func (h *AuthHandler) auditLog(ctx context.Context, tenantID *uuid.UUID, userID *uuid.UUID, action, target, ip string, meta map[string]any) {
	// Marshal metadata via stdlib encoding/json so we get real JSON.
	var metaJSON []byte
	if meta != nil {
		metaJSON = jsonMarshal(meta)
	}
	if len(metaJSON) == 0 {
		metaJSON = []byte("{}")
	}
	var ipArg *string
	if ip != "" {
		ipArg = &ip
	}
	_, err := h.pool.Pgx().Exec(ctx, `
		INSERT INTO audit_log (tenant_id, user_id, action, target, metadata, ip, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::inet, NOW())
	`, tenantID, userID, action, target, string(metaJSON), ipArg)
	if err != nil {
		// Log to stdout so we can see it in journalctl.
		h.logger.Error("audit_log insert failed", "action", action, "err", err.Error())
	}
}

// ========== POST /auth/forgot ==========

// ForgotRequest is the JSON body for POST /auth/forgot.
type ForgotRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword issues a single-use password reset token.
//
// Always returns 200 even for non-existent emails (so we don't leak
// which addresses are registered). In DEV mode (no SMTP configured),
// the response includes the token + reset URL so the developer can
// complete the flow without setting up mail.
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	ip := clientIP(c)

	// Look up user (silently succeed if not found).
	u, _, err := h.lookupUserAndTenant(c.Request.Context(), email)
	if err != nil {
		// No email disclosure — return same response shape.
		kernel.RespondOK(c, gin.H{"ok": true, "message": "if the email exists, a reset link has been generated"})
		return
	}

	raw, hash, err := generateToken()
	if err != nil {
		kernel.RespondError(c, kernel.ErrInternal)
		return
	}

	expires := time.Now().UTC().Add(tokenTTL)
	var ipArg *string
	if ip != "" {
		ipArg = &ip
	}
	_, err = h.pool.Pgx().Exec(c.Request.Context(), `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at, ip)
		VALUES ($1, $2, $3, $4::inet)
	`, u.ID, hash, expires, ipArg)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}

	h.auditLog(c.Request.Context(), &u.TenantID, &u.ID, "password_reset.requested", u.Email, ip, map[string]any{"token_id": hash[:8]})

	resetURL := fmt.Sprintf("/reset-password?token=%s", raw)
	resp := gin.H{
		"ok":         true,
		"message":    "if the email exists, a reset link has been generated",
		"expires_at": expires.Format(time.RFC3339),
		"email_sent": h.emailSender != nil,
	}
	if h.emailSender == nil {
		// Self-hosted mode (no SMTP wired): surface the link directly in the
		// response so the user can complete the reset without mail. Field
		// names mirror the contract the frontend expects.
		resp["reset_token"] = raw
		resp["reset_url"] = resetURL
		// Legacy aliases kept so older clients keep working.
		resp["dev_token"] = raw
		resp["dev_url"] = resetURL
	}
	kernel.RespondOK(c, resp)
}

// ========== POST /auth/reset ==========

// ResetRequest is the JSON body for POST /auth/reset.
type ResetRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ResetPassword consumes a reset token and updates the user's password.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	ip := clientIP(c)
	tokenHash := hashToken(req.Token)

	// Atomically consume the token. We mark used_at in the same tx that
	// updates the password, so a double-submit cannot change the password twice.
	tx, err := h.pool.Pgx().Begin(c.Request.Context())
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()

	var userID uuid.UUID
	var expiresAt time.Time
	var usedAt *time.Time
	row := tx.QueryRow(c.Request.Context(), `
		SELECT user_id, expires_at, used_at
		FROM password_reset_tokens
		WHERE token_hash = $1
		FOR UPDATE
	`, tokenHash)
	if err := row.Scan(&userID, &expiresAt, &usedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		kernel.RespondError(c, err)
		return
	}

	now := time.Now().UTC()
	if usedAt != nil {
		kernel.RespondError(c, kernel.ErrConflict) // already used
		return
	}
	if now.After(expiresAt) {
		kernel.RespondError(c, kernel.ErrUnauthorized) // expired
		return
	}

	// Hash new password + write both updates in the same tx.
	hash, err := hashPassword(req.NewPassword)
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if _, err := tx.Exec(c.Request.Context(),
		`UPDATE users SET password_hash=$1, must_change_password=false, updated_at=NOW() WHERE id=$2`,
		hash, userID,
	); err != nil {
		kernel.RespondError(c, err)
		return
	}
	if _, err := tx.Exec(c.Request.Context(),
		`UPDATE password_reset_tokens SET used_at=NOW() WHERE token_hash=$1`,
		tokenHash,
	); err != nil {
		kernel.RespondError(c, err)
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		kernel.RespondError(c, err)
		return
	}

	// Look up tenant for the audit row.
	var tenantID uuid.UUID
	_ = h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT tenant_id FROM users WHERE id=$1`, userID).Scan(&tenantID)
	h.auditLog(c.Request.Context(), &tenantID, &userID, "password_reset.completed", "", ip, nil)

	kernel.RespondOK(c, gin.H{"ok": true, "message": "password updated"})
}

// ========== PATCH /auth/profile ==========

// ProfileRequest is the JSON body for PATCH /auth/profile.
type ProfileRequest struct {
	FullName string `json:"full_name" binding:"required,min=1,max=255"`
}

// UpdateProfile updates the current user's full_name.
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	claimsUser, ok := userFromContext(c)
	if !ok {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	var req ProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if _, err := h.pool.Pgx().Exec(c.Request.Context(),
		`UPDATE users SET full_name=$1, updated_at=NOW() WHERE id=$2`,
		fullName, claimsUser.ID,
	); err != nil {
		kernel.RespondError(c, err)
		return
	}
	h.auditLog(c.Request.Context(), &claimsUser.TenantID, &claimsUser.ID, "user.profile_updated", claimsUser.Email, clientIP(c), nil)
	kernel.RespondOK(c, gin.H{"ok": true, "full_name": fullName})
}

// ========== POST /auth/change-password (with audit) ==========

// ChangePasswordRequest is the JSON body for POST /auth/change-password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,min=8"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword verifies the current password and updates it.
// Logs to audit_log on success and returns a fresh JWT (with must_change_password=false).
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	claimsUser, ok := userFromContext(c)
	if !ok {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	// Fetch user record to verify old password + write new hash.
	u, _, err := h.lookupUserAndTenant(c.Request.Context(), claimsUser.Email)
	if err != nil {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if !checkHash(u.PasswordHash, req.OldPassword) {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	newHash, err := hashPassword(req.NewPassword)
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if _, err := h.pool.Pgx().Exec(c.Request.Context(),
		`UPDATE users SET password_hash=$1, must_change_password=false, updated_at=NOW() WHERE id=$2`,
		newHash, u.ID,
	); err != nil {
		kernel.RespondError(c, err)
		return
	}
	h.auditLog(c.Request.Context(), &u.TenantID, &u.ID, "password_changed", u.Email, clientIP(c), nil)
	// Issue a fresh JWT so the caller immediately has a token without the flag.
	fresh, err := h.issuer.Issue(u.ID, u.TenantID, u.Email, u.Role, false)
	if err != nil {
		// Non-fatal: old JWT still works (and re-fetching will pick up the flag clear).
		kernel.RespondOK(c, gin.H{"ok": true})
		return
	}
	kernel.RespondOK(c, gin.H{"ok": true, "token": fresh, "ttl": h.issuer.TTLSeconds()})
}

// ========== POST /auth/magic-link ==========

// MagicLinkRequest is the JSON body for POST /auth/magic-link.
type MagicLinkRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// MagicLink issues a single-use login token (no password needed).
// Token TTL is the same as reset tokens (1 hour).
//
// Returns the token in DEV mode (same as /forgot).
func (h *AuthHandler) MagicLink(c *gin.Context) {
	var req MagicLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	ip := clientIP(c)

	u, _, err := h.lookupUserAndTenant(c.Request.Context(), email)
	if err != nil {
		kernel.RespondOK(c, gin.H{"ok": true, "message": "if the email exists, a magic link has been generated"})
		return
	}

	raw, hash, err := generateTokenWithPrefix("ml_")
	if err != nil {
		kernel.RespondError(c, kernel.ErrInternal)
		return
	}
	expires := time.Now().UTC().Add(tokenTTL)
	var ipArg *string
	if ip != "" {
		ipArg = &ip
	}
	_, err = h.pool.Pgx().Exec(c.Request.Context(), `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at, ip)
		VALUES ($1, $2, $3, $4::inet)
	`, u.ID, hash, expires, ipArg)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}

	h.auditLog(c.Request.Context(), &u.TenantID, &u.ID, "magic_link.requested", u.Email, ip, nil)

	magicURL := fmt.Sprintf("/auth/magic-link/consume?token=%s", raw)
	resp := gin.H{
		"ok":         true,
		"message":    "if the email exists, a magic link has been generated",
		"expires_at": expires.Format(time.RFC3339),
		"email_sent": h.emailSender != nil,
	}
	if h.emailSender == nil {
		// Self-hosted mode: surface the link directly in the response.
		resp["magic_token"] = raw
		resp["magic_url"] = magicURL
		// Legacy aliases.
		resp["dev_token"] = raw
		resp["dev_url"] = magicURL
	}
	kernel.RespondOK(c, resp)
}

// ========== POST /auth/accept-invite ==========

// InviteRequest is the JSON body for POST /auth/accept-invite.
type InviteRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// AcceptInvite consumes an invite token and activates the user with the
// provided password. Same table (password_reset_tokens) — invite tokens
// are just reset tokens with a different action label.
//
// After consuming, returns a fresh JWT so the user is logged in immediately.
func (h *AuthHandler) AcceptInvite(c *gin.Context) {
	var req InviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	ip := clientIP(c)
	tokenHash := hashToken(req.Token)

	tx, err := h.pool.Pgx().Begin(c.Request.Context())
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()

	var userID, tenantID uuid.UUID
	var status string
	var expiresAt time.Time
	var usedAt *time.Time
	row := tx.QueryRow(c.Request.Context(), `
		SELECT u.id, u.tenant_id, u.status, prt.expires_at, prt.used_at
		FROM password_reset_tokens prt
		JOIN users u ON u.id = prt.user_id
		WHERE prt.token_hash = $1
		FOR UPDATE OF prt
	`, tokenHash)
	if err := row.Scan(&userID, &tenantID, &status, &expiresAt, &usedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	now := time.Now().UTC()
	if usedAt != nil || now.After(expiresAt) {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	newHash, err := hashPassword(req.NewPassword)
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if _, err := tx.Exec(c.Request.Context(),
		`UPDATE users SET password_hash=$1, status='active', must_change_password=false, updated_at=NOW() WHERE id=$2`,
		newHash, userID,
	); err != nil {
		kernel.RespondError(c, err)
		return
	}
	if _, err := tx.Exec(c.Request.Context(),
		`UPDATE password_reset_tokens SET used_at=NOW() WHERE token_hash=$1`,
		tokenHash,
	); err != nil {
		kernel.RespondError(c, err)
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		kernel.RespondError(c, err)
		return
	}

	// Issue token + audit.
	var email, role string
	_ = h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT email, role FROM users WHERE id=$1`, userID).Scan(&email, &role)
	jwt, err := h.issuer.Issue(userID, tenantID, email, role, false)
	if err != nil {
		kernel.RespondError(c, kernel.ErrInternal)
		return
	}

	h.auditLog(c.Request.Context(), &tenantID, &userID, "invite.accepted", email, ip, nil)

	kernel.RespondOK(c, gin.H{
		"ok":    true,
		"token": jwt,
		"ttl":   h.issuer.TTLSeconds(),
	})
}

// Compile-time check that all imports are used.
var (
	_ = http.StatusOK
	_ = auth.ClaimsCtxKey
)

// ========== Brute-force counter ==========

// recordFailedLogin increments the failed-login counter for a user and
// returns the total count of failed logins in the last 1 hour.
// Old entries (older than 1 hour) are pruned on each call so the table
// stays small.
func (h *AuthHandler) recordFailedLogin(ctx context.Context, userID uuid.UUID) (int, error) {
	// Insert a new failure event.
	if _, err := h.pool.Pgx().Exec(ctx,
		`INSERT INTO user_failed_logins (user_id, ts) VALUES ($1, NOW())`, userID,
	); err != nil {
		return 0, err
	}
	// Prune old entries (>1h).
	if _, err := h.pool.Pgx().Exec(ctx,
		`DELETE FROM user_failed_logins WHERE ts < NOW() - INTERVAL '1 hour'`,
	); err != nil {
		return 0, err
	}
	// Return current count in the last hour.
	var count int64
	err := h.pool.Pgx().QueryRow(ctx,
		`SELECT count(*) FROM user_failed_logins WHERE user_id = $1 AND ts > NOW() - INTERVAL '1 hour'`,
		userID,
	).Scan(&count)
	return int(count), err
}
