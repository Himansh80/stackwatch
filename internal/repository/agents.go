package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListAgents returns all agents for a tenant.
func (r *PgxServerRepo) ListAgents(ctx context.Context, tenantID uuid.UUID) ([]Agent, error) {
	rows, err := r.pool.Pgx().Query(ctx, `
		SELECT id, tenant_id, name, server_id, status,
		       COALESCE(version,''), COALESCE(os,''), COALESCE(arch,''),
		       labels, last_seen_at, last_heartbeat, created_at, updated_at
		FROM agents
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		var serverID *uuid.UUID
		var labels []byte
		var lastSeenAt, lastHeartbeat *time.Time
		if err := rows.Scan(&a.ID, &a.TenantID, &a.Name, &serverID, &a.Status,
			&a.Version, &a.OS, &a.Arch, &labels, &lastSeenAt, &lastHeartbeat, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.ServerID = serverID
		a.Labels = labels
		a.LastSeenAt = lastSeenAt
		a.LastHeartbeat = lastHeartbeat
		agents = append(agents, a)
	}
	return agents, nil
}

// CreateAgent inserts a new agent with the given ingest key.
func (r *PgxServerRepo) CreateAgent(ctx context.Context, tenantID uuid.UUID, name, ingestKey, keyHash string, labels []byte) (*Agent, error) {
	id := uuid.New()
	_, err := r.pool.Pgx().Exec(ctx, `
		INSERT INTO agents (id, tenant_id, name, ingest_key, key_hash, labels, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
	`, id, tenantID, name, ingestKey, keyHash, labels)
	if err != nil {
		return nil, err
	}
	return &Agent{
		ID:        id,
		TenantID:  tenantID,
		Name:      name,
		IngestKey: ingestKey,
		Status:    "pending",
		Labels:    labels,
	}, nil
}
