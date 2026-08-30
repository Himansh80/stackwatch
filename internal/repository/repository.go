// Package repository provides the data access layer for StackWatch.
// All SQL lives here. Handlers and services call these interfaces, never pgx directly.
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stackwatch/platform/internal/db"
)

// DB is the abstraction over *db.Pool for query methods.
// Implementation may be real Postgres (pgx) or a mock for tests.
type DB interface {
	// Server queries
	ListServers(ctx context.Context, tenantID uuid.UUID) ([]Server, error)
	GetServer(ctx context.Context, tenantID, serverID uuid.UUID) (*Server, error)
	GetServerByHostname(ctx context.Context, hostname string) (*Server, error)
	CreateServer(ctx context.Context, tenantID uuid.UUID, in CreateServerInput) (*Server, error)
	UpdateServer(ctx context.Context, tenantID, serverID uuid.UUID, in UpdateServerInput) (bool, error)
	DeleteServer(ctx context.Context, tenantID, serverID uuid.UUID) (bool, error)
	CountServers(ctx context.Context, tenantID uuid.UUID) (int, error)

	// Metrics queries
	GetLatestMetric(ctx context.Context, serverID uuid.UUID, metricName string, window time.Duration) (float64, error)
	QueryMetrics(ctx context.Context, tenantID uuid.UUID, serverIDs []uuid.UUID, metricNames []string, start, end time.Time, limit int) ([]MetricPoint, error)
	InsertMetrics(ctx context.Context, tenantID uuid.UUID, points []MetricPointInput) error

	// Agent queries
	ListAgents(ctx context.Context, tenantID uuid.UUID) ([]Agent, error)
	CreateAgent(ctx context.Context, tenantID uuid.UUID, name string, ingestKey, keyHash string, labels []byte) (*Agent, error)

	// Tenant (unused — implement when needed)
	// GetTenantByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
}

// PgxRepository is the production implementation backed by *db.Pool.
type PgxRepository struct {
	pool *db.Pool
}

// NewPgxRepository constructs the repository against a real Postgres pool.
func NewPgxRepository(pool *db.Pool) *PgxRepository {
	return &PgxRepository{pool: pool}
}
