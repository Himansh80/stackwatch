package handler

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/kernel"
)

// lookupUserAndTenant fetches the user and tenant by email.
func (h *AuthHandler) lookupUserAndTenant(ctx context.Context, email string) (*kernel.User, *kernel.Tenant, error) {
	row := h.pool.Pgx().QueryRow(ctx, `
		SELECT u.id, u.tenant_id, u.email, u.full_name, u.password_hash, u.role, u.status, u.must_change_password, u.created_at, u.updated_at,
		       t.id, t.name, t.slug, t.plan, t.status, t.created_at, t.updated_at
		FROM users u
		JOIN tenants t ON t.id = u.tenant_id
		WHERE u.email = $1 AND u.status != 'deleted' AND t.status != 'deleted'
		LIMIT 1
	`, email)
	u := &kernel.User{}
	t := &kernel.Tenant{}
	if err := row.Scan(
		&u.ID, &u.TenantID, &u.Email, &u.FullName, &u.PasswordHash, &u.Role, &u.Status, &u.MustChangePW, &u.CreatedAt, &u.UpdatedAt,
		&t.ID, &t.Name, &t.Slug, &t.Plan, &t.Status, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, kernel.ErrNotFound
		}
		return nil, nil, err
	}
	return u, t, nil
}

// checkHash is a thin wrapper around auth.CheckPassword for handler use.
func checkHash(hash, plain string) bool {
	return auth.CheckPassword(hash, plain)
}

// generateUserID returns a new UUID for a user.
func generateUserID() uuid.UUID {
	return uuid.New()
}

// generateTenantID returns a new UUID for a tenant.
func generateTenantID() uuid.UUID {
	return uuid.New()
}
