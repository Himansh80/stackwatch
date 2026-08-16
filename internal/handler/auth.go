// Package handler exposes HTTP handlers. One file per business domain.
package handler

import (
	"log/slog"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
)

// AuthHandler exposes login / signup / me endpoints.
type AuthHandler struct {
	pool   *db.Pool
	issuer *auth.Issuer
	logger *slog.Logger
}

// NewAuthHandler constructs the auth handler.
func NewAuthHandler(pool *db.Pool, issuer *auth.Issuer, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{pool: pool, issuer: issuer, logger: logger}
}
