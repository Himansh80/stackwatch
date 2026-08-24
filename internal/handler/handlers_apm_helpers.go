// Tier 7 — APM (D2) — Helpers shared by the APM handlers.
//
// Kept tiny on purpose — just upsertAPMService + nullIfEmpty. Every
// APM handler in the package uses these so the service-resolution
// rule (auto-create on first use) is identical across files.
package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// upsertAPMService creates a service row on demand if missing.
// Returns the service UUID. Empty lang/fw/env passed through.
//
// Used by IngestAPMTrace so the root service referenced by a trace
// is auto-created the first time we see it for this tenant — keeps
// the SDK side free of a separate "register then send" handshake.
func upsertAPMService(ctx context.Context, pool *db.Pool, tenantID uuid.UUID, name, lang, fw, env string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.Pgx().QueryRow(ctx,
		`INSERT INTO apm_services (tenant_id, name, language, framework, environment)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (tenant_id, name) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id`,
		tenantID, name, lang, fw, env,
	).Scan(&id)
	return id, err
}

// nullIfEmpty returns nil for "" strings (so the DB column gets NULL)
// and the value itself otherwise. Used for optional text columns.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
