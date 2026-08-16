// Package db wraps the pgx connection pool. One file per type.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config is the database connection configuration.
type Config struct {
	URL               string
	MaxConns          int32
	MinConns          int32
	HealthCheckPeriod time.Duration
}

// DefaultConfig returns sane defaults.
func DefaultConfig(url string) Config {
	return Config{
		URL:               url,
		MaxConns:          50,
		MinConns:          5,
		HealthCheckPeriod: 30 * time.Second,
	}
}

// Pool wraps pgxpool.Pool with project conventions.
type Pool struct {
	pool *pgxpool.Pool
}

// Connect opens a pool and verifies connectivity.
func Connect(ctx context.Context, cfg Config) (*Pool, error) {
	pcfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	pcfg.MaxConns = cfg.MaxConns
	pcfg.MinConns = cfg.MinConns
	pcfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Pool{pool: pool}, nil
}

// Close releases all connections.
func (p *Pool) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

// Health checks the database is reachable.
func (p *Pool) Health(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Pgx returns the underlying pool for advanced usage.
func (p *Pool) Pgx() *pgxpool.Pool {
	return p.pool
}
