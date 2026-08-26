// Package handler exposes HTTP handlers. One file per business domain.
package handler

import (
	"log/slog"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
)

// EmailSender is anything that can deliver a transactional email.
// When non-nil, /auth/forgot and /auth/magic-link will send real
// emails instead of returning the reset link in the response body.
//
// Today we don't implement SMTP — SetEmailSender(nil) keeps the
// self-hosted default where the link is surfaced inline for the
// user. When SMTP gets wired, just pass a struct that satisfies
// this interface (e.g. a thin wrapper around Resend's HTTP API).
type EmailSender interface {
	SendPasswordReset(toEmail, resetURL string) error
	SendMagicLink(toEmail, magicURL string) error
}

// AuthHandler exposes login / signup / me endpoints.
type AuthHandler struct {
	pool        *db.Pool
	issuer      *auth.Issuer
	logger      *slog.Logger
	emailSender EmailSender // optional — nil means self-hosted mode
}

// NewAuthHandler constructs the auth handler.
func NewAuthHandler(pool *db.Pool, issuer *auth.Issuer, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{pool: pool, issuer: issuer, logger: logger}
}

// SetEmailSender installs the email delivery backend. Pass nil to
// restore self-hosted mode (reset link in response body).
func (h *AuthHandler) SetEmailSender(s EmailSender) {
	h.emailSender = s
}
