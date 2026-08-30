package repository

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stackwatch/platform/internal/db"
)

// PgxServerRepo is the Postgres implementation of ServerRepo.
type PgxServerRepo struct {
	pool *db.Pool
}

// NewPgxServerRepo constructs a Postgres-backed server repository.
func NewPgxServerRepo(pool *db.Pool) *PgxServerRepo {
	return &PgxServerRepo{pool: pool}
}

// ListServers fetches all servers for a tenant.
func (r *PgxServerRepo) ListServers(ctx context.Context, tenantID uuid.UUID) ([]Server, error) {
	rows, err := r.pool.Pgx().Query(ctx, `
		SELECT id, tenant_id, agent_id, name, hostname, COALESCE(ip_address::text, ''),
		       COALESCE(os,''), COALESCE(os_version,''), COALESCE(arch,''),
		       COALESCE(kernel_version,''), COALESCE(cpu_cores,-1), COALESCE(cpu_model,''),
		       COALESCE(memory_total,-1), COALESCE(disk_total,-1),
		       tags, status, last_seen_at, created_at, updated_at
		FROM servers
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []Server
	for rows.Next() {
		var s Server
		var agentID *uuid.UUID
		var tags []byte
		var lastSeenAt *time.Time
		if err := rows.Scan(&s.ID, &s.TenantID, &agentID, &s.Name, &s.Hostname, &s.IPAddress,
			&s.OS, &s.OSVersion, &s.Arch, &s.KernelVersion, &s.CPUCores, &s.CPUModel,
			&s.MemoryTotal, &s.DiskTotal, &tags, &s.Status, &lastSeenAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.AgentID = agentID
		s.Tags = tags
		s.LastSeenAt = lastSeenAt
		servers = append(servers, s)
	}
	return servers, nil
}

// GetServer returns a single server by ID (scoped by tenant).
func (r *PgxServerRepo) GetServer(ctx context.Context, tenantID, serverID uuid.UUID) (*Server, error) {
	var s Server
	var agentID *uuid.UUID
	var tags []byte
	var lastSeenAt *time.Time
	err := r.pool.Pgx().QueryRow(ctx, `
		SELECT id, tenant_id, agent_id, name, hostname, COALESCE(ip_address::text, ''),
		       COALESCE(os,''), COALESCE(os_version,''), COALESCE(arch,''),
		       COALESCE(kernel_version,''), COALESCE(cpu_cores,-1), COALESCE(cpu_model,''),
		       COALESCE(memory_total,-1), COALESCE(disk_total,-1),
		       tags, status, last_seen_at, created_at, updated_at
		FROM servers
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
	`, tenantID, serverID).Scan(
		&s.ID, &s.TenantID, &agentID, &s.Name, &s.Hostname, &s.IPAddress,
		&s.OS, &s.OSVersion, &s.Arch, &s.KernelVersion, &s.CPUCores, &s.CPUModel,
		&s.MemoryTotal, &s.DiskTotal, &tags, &s.Status, &lastSeenAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, FromPgxErr(err)
	}
	s.AgentID = agentID
	s.Tags = tags
	s.LastSeenAt = lastSeenAt
	return &s, nil
}

// CreateServer inserts a new server row.
func (r *PgxServerRepo) CreateServer(ctx context.Context, tenantID uuid.UUID, in CreateServerInput) (*Server, error) {
	id := uuid.New()
	tags := in.Tags
	if tags == nil {
		tags = []byte("{}")
	}
	_, err := r.pool.Pgx().Exec(ctx, `
		INSERT INTO servers (id, tenant_id, name, hostname, ip_address, os, os_version, arch,
		                    kernel_version, cpu_cores, cpu_model, memory_total, disk_total, tags, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 'unknown')
	`, id, tenantID, in.Name, in.Hostname, nullIfEmpty(in.IPAddress), in.OS, in.OSVersion,
		in.Arch, "", 0, "", 0, 0, tags)
	if err != nil {
		return nil, err
	}
	return r.GetServer(ctx, tenantID, id)
}

// UpdateServer applies a partial update. Returns rowsAffected=1 on success.
func (r *PgxServerRepo) UpdateServer(ctx context.Context, tenantID, serverID uuid.UUID, in UpdateServerInput) (int64, error) {
	setClauses := []string{}
	args := []interface{}{tenantID, serverID}
	argIdx := 3

	if in.Name != nil {
		setClauses = append(setClauses, "name = $"+strconv.Itoa(argIdx))
		args = append(args, *in.Name)
		argIdx++
	}
	if in.Hostname != nil {
		setClauses = append(setClauses, "hostname = $"+strconv.Itoa(argIdx))
		args = append(args, *in.Hostname)
		argIdx++
	}
	if in.IPAddress != nil {
		setClauses = append(setClauses, "ip_address = $"+strconv.Itoa(argIdx))
		args = append(args, *in.IPAddress)
		argIdx++
	}
	if in.OS != nil {
		setClauses = append(setClauses, "os = $"+strconv.Itoa(argIdx))
		args = append(args, *in.OS)
		argIdx++
	}
	if in.OSVersion != nil {
		setClauses = append(setClauses, "os_version = $"+strconv.Itoa(argIdx))
		args = append(args, *in.OSVersion)
		argIdx++
	}
	if in.Arch != nil {
		setClauses = append(setClauses, "arch = $"+strconv.Itoa(argIdx))
		args = append(args, *in.Arch)
		argIdx++
	}
	if in.KernelVersion != nil {
		setClauses = append(setClauses, "kernel_version = $"+strconv.Itoa(argIdx))
		args = append(args, *in.KernelVersion)
		argIdx++
	}
	if in.CPUCores != nil {
		setClauses = append(setClauses, "cpu_cores = $"+strconv.Itoa(argIdx))
		args = append(args, *in.CPUCores)
		argIdx++
	}
	if in.CPUModel != nil {
		setClauses = append(setClauses, "cpu_model = $"+strconv.Itoa(argIdx))
		args = append(args, *in.CPUModel)
		argIdx++
	}
	if in.MemoryTotal != nil {
		setClauses = append(setClauses, "memory_total = $"+strconv.Itoa(argIdx))
		args = append(args, *in.MemoryTotal)
		argIdx++
	}
	if in.DiskTotal != nil {
		setClauses = append(setClauses, "disk_total = $"+strconv.Itoa(argIdx))
		args = append(args, *in.DiskTotal)
		argIdx++
	}
	if in.Tags != nil {
		setClauses = append(setClauses, "tags = $"+strconv.Itoa(argIdx))
		args = append(args, *in.Tags)
		argIdx++
	}
	if in.Status != nil {
		setClauses = append(setClauses, "status = $"+strconv.Itoa(argIdx))
		args = append(args, *in.Status)
		argIdx++
	}

	if len(setClauses) == 0 {
		return 0, nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := "UPDATE servers SET " + strings.Join(setClauses, ", ") +
		" WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL"

	result, err := r.pool.Pgx().Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// DeleteServer soft-deletes a server (sets deleted_at).
func (r *PgxServerRepo) DeleteServer(ctx context.Context, tenantID, serverID uuid.UUID) (int64, error) {
	result, err := r.pool.Pgx().Exec(ctx, `
		UPDATE servers SET deleted_at = NOW(), updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
	`, tenantID, serverID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// TouchServer updates last_seen_at and status='up' (used by ingest heartbeat).
func (r *PgxServerRepo) TouchServer(ctx context.Context, tenantID, serverID uuid.UUID) error {
	_, err := r.pool.Pgx().Exec(ctx, `
		UPDATE servers
		SET last_seen_at = NOW(), status = 'up', updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
	`, tenantID, serverID)
	return err
}

// FindServerByHostname returns the first server matching hostname (any tenant).
// Used by the public ingest heartbeat to auto-register by hostname.
func (r *PgxServerRepo) FindServerByHostname(ctx context.Context, hostname string) (*Server, error) {
	var s Server
	var agentID *uuid.UUID
	var tags []byte
	var lastSeenAt *time.Time
	err := r.pool.Pgx().QueryRow(ctx, `
		SELECT id, tenant_id, agent_id, name, hostname, COALESCE(ip_address::text, ''),
		       COALESCE(os,''), COALESCE(os_version,''), COALESCE(arch,''),
		       COALESCE(kernel_version,''), COALESCE(cpu_cores,-1), COALESCE(cpu_model,''),
		       COALESCE(memory_total,-1), COALESCE(disk_total,-1),
		       tags, status, last_seen_at, created_at, updated_at
		FROM servers
		WHERE hostname = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT 1
	`, hostname).Scan(
		&s.ID, &s.TenantID, &agentID, &s.Name, &s.Hostname, &s.IPAddress,
		&s.OS, &s.OSVersion, &s.Arch, &s.KernelVersion, &s.CPUCores, &s.CPUModel,
		&s.MemoryTotal, &s.DiskTotal, &tags, &s.Status, &lastSeenAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, FromPgxErr(err)
	}
	s.AgentID = agentID
	s.Tags = tags
	s.LastSeenAt = lastSeenAt
	return &s, nil
}

// AutoRegisterServer inserts a new server with just a hostname (first tenant, self-hosted default).
func (r *PgxServerRepo) AutoRegisterServer(ctx context.Context, hostname string) (*Server, error) {
	var tenantID uuid.UUID
	if err := r.pool.Pgx().QueryRow(ctx, "SELECT id FROM tenants ORDER BY created_at ASC LIMIT 1").Scan(&tenantID); err != nil {
		return nil, err
	}
	id := uuid.New()
	_, err := r.pool.Pgx().Exec(ctx, `
		INSERT INTO servers (id, tenant_id, name, hostname, status)
		VALUES ($1, $2, $3, $3, 'unknown')
	`, id, tenantID, hostname)
	if err != nil {
		return nil, err
	}
	return r.GetServer(ctx, tenantID, id)
}
