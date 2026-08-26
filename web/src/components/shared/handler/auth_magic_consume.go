package handler

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/kernel"
)

// ConsumeMagicLink consumes a purpose-scoped magic-link token and returns a
// fresh JWT. Magic tokens are prefixed with "ml_" so reset and invite tokens
// cannot be used as login tokens.
func (h *AuthHandler) ConsumeMagicLink(c *gin.Context) {
	rawToken := strings.TrimSpace(c.Query("token"))
	if rawToken == "" || !strings.HasPrefix(rawToken, "ml_") {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}

	tx, err := h.pool.Pgx().Begin(c.Request.Context())
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()

	var userID, tenantID uuid.UUID
	var email, role string
	var expiresAt time.Time
	var usedAt *time.Time
	row := tx.QueryRow(c.Request.Context(), `
		SELECT u.id, u.tenant_id, u.email, u.role, prt.expires_at, prt.used_at
		FROM password_reset_tokens prt
		JOIN users u ON u.id = prt.user_id
		WHERE prt.token_hash = $1
		FOR UPDATE OF prt
	`, hashToken(rawToken))
	if err := row.Scan(&userID, &tenantID, &email, &role, &expiresAt, &usedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	if usedAt != nil || time.Now().UTC().After(expiresAt) {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}

	if _, err := tx.Exec(c.Request.Context(),
		`UPDATE password_reset_tokens SET used_at=NOW() WHERE token_hash=$1`,
		hashToken(rawToken),
	); err != nil {
		kernel.RespondError(c, err)
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		kernel.RespondError(c, err)
		return
	}

	jwt, err := h.issuer.Issue(userID, tenantID, email, role, false)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	h.auditLog(c.Request.Context(), &tenantID, &userID, "magic_link.accepted", email, clientIP(c), nil)
	kernel.RespondOK(c, gin.H{"ok": true, "token": jwt, "ttl": h.issuer.TTLSeconds()})
}
