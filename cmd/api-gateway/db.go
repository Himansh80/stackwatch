package main

import (
	"context"
	"time"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
)

// connectDatabase opens the connection pool.
func connectDatabase(ctx context.Context, url string) (*db.Pool, error) {
	return db.Connect(ctx, db.DefaultConfig(url))
}

// newIssuer creates the JWT issuer with a stable secret and 24h TTL.
func newIssuer(secret string) *auth.Issuer {
	return auth.NewIssuer(secret, 24*time.Hour)
}
