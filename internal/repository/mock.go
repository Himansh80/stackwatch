package repository

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MockRepository is an in-memory implementation for tests.
type MockRepository struct {
	mu       sync.Mutex
	servers  map[uuid.UUID]Server
	agents   map[uuid.UUID]Agent
	metrics  []MetricPoint
	tenants  map[uuid.UUID]struct{}
	hostname map[string]uuid.UUID // hostname → server_id
}

// NewMockRepository returns an empty in-memory repo.
func NewMockRepository() *MockRepository {
	return &MockRepository{
		servers:  make(map[uuid.UUID]Server),
		agents:   make(map[uuid.UUID]Agent),
		metrics:  make([]MetricPoint, 0),
		tenants:  make(map[uuid.UUID]struct{}),
		hostname: make(map[string]uuid.UUID),
	}
}

// AddTenant seeds a tenant so handlers that call tenant-lookup succeed.
func (m *MockRepository) AddTenant(ctx context.Context, tenantID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenants[tenantID] = struct{}{}
}

// CreateServer inserts a server in-memory.
func (m *MockRepository) CreateServer(ctx context.Context, tenantID uuid.UUID, in CreateServerInput) (*Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	s := Server{
		ID:        id,
		TenantID:  tenantID,
		Name:      in.Name,
		Hostname:  in.Hostname,
		IPAddress: in.IPAddress,
		OS:        in.OS,
		Arch:      in.Arch,
		Tags:      []byte("{}"),
		Status:    "unknown",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.servers[id] = s
	m.hostname[in.Hostname] = id
	return &s, nil
}

// GetServer returns the server or pgx.ErrNoRows.
func (m *MockRepository) GetServer(ctx context.Context, tenantID, serverID uuid.UUID) (*Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.servers[serverID]
	if !ok || s.TenantID != tenantID {
		return nil, ErrNotFound
	}
	return &s, nil
}

// GetServerByHostname looks up a server by hostname (used by ingest).
func (m *MockRepository) GetServerByHostname(ctx context.Context, hostname string) (*Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.hostname[hostname]
	if !ok {
		return nil, ErrNotFound
	}
	s := m.servers[id]
	return &s, nil
}

// ListServers returns all servers for a tenant.
func (m *MockRepository) ListServers(ctx context.Context, tenantID uuid.UUID) ([]Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Server, 0, len(m.servers))
	for _, s := range m.servers {
		if s.TenantID == tenantID {
			out = append(out, s)
		}
	}
	return out, nil
}

// UpdateServer applies a partial update. Returns rowsAffected.
func (m *MockRepository) UpdateServer(ctx context.Context, tenantID, serverID uuid.UUID, in UpdateServerInput) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.servers[serverID]
	if !ok || s.TenantID != tenantID {
		return 0, ErrNotFound
	}
	if in.Name != nil {
		s.Name = *in.Name
	}
	if in.Hostname != nil {
		s.Hostname = *in.Hostname
		delete(m.hostname, s.Hostname)
		m.hostname[*in.Hostname] = serverID
	}
	if in.Status != nil {
		s.Status = *in.Status
	}
	s.UpdatedAt = time.Now()
	m.servers[serverID] = s
	return 1, nil
}

// DeleteServer removes the server.
func (m *MockRepository) DeleteServer(ctx context.Context, tenantID, serverID uuid.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.servers[serverID]
	if !ok || s.TenantID != tenantID {
		return 0, ErrNotFound
	}
	delete(m.servers, serverID)
	delete(m.hostname, s.Hostname)
	return 1, nil
}

// TouchServer bumps last_seen_at + status.
func (m *MockRepository) TouchServer(ctx context.Context, tenantID, serverID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.servers[serverID]
	if !ok || s.TenantID != tenantID {
		return ErrNotFound
	}
	now := time.Now()
	s.LastSeenAt = &now
	s.Status = "up"
	s.UpdatedAt = now
	m.servers[serverID] = s
	return nil
}

// AutoRegisterServer creates a server under the first tenant (self-hosted UX).
func (m *MockRepository) AutoRegisterServer(ctx context.Context, hostname string) (*Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Use first known tenant, or fallback zero
	var tenantID uuid.UUID
	for t := range m.tenants {
		tenantID = t
		break
	}
	id := uuid.New()
	s := Server{
		ID:        id,
		TenantID:  tenantID,
		Name:      hostname,
		Hostname:  hostname,
		Status:    "unknown",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Tags:      []byte("{}"),
	}
	m.servers[id] = s
	m.hostname[hostname] = id
	return &s, nil
}

// ListAgents returns all agents for a tenant.
func (m *MockRepository) ListAgents(ctx context.Context, tenantID uuid.UUID) ([]Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Agent, 0, len(m.agents))
	for _, a := range m.agents {
		if a.TenantID == tenantID {
			out = append(out, a)
		}
	}
	return out, nil
}

// CreateAgent inserts an agent.
func (m *MockRepository) CreateAgent(ctx context.Context, tenantID uuid.UUID, name string, ingestKey, keyHash string, labels []byte) (*Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	a := Agent{
		ID:        id,
		TenantID:  tenantID,
		Name:      name,
		IngestKey: ingestKey,
		KeyHash:   keyHash,
		Status:    "pending",
		Labels:    labels,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	m.agents[id] = a
	return &a, nil
}

// InsertMetrics appends points.
func (m *MockRepository) InsertMetrics(ctx context.Context, tenantID uuid.UUID, points []MetricPointInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range points {
		m.metrics = append(m.metrics, MetricPoint{
			ServerID:   p.ServerID,
			MetricName: p.MetricName,
			Value:      p.Value,
			Timestamp:  time.Now(),
		})
	}
	return nil
}

// QueryMetrics returns matching points (crude time filter).
func (m *MockRepository) QueryMetrics(ctx context.Context, tenantID uuid.UUID, serverIDs []uuid.UUID, metricNames []string, start, end time.Time, limit int) ([]MetricPoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []MetricPoint
	for _, p := range m.metrics {
		if p.Timestamp.Before(start) || p.Timestamp.After(end) {
			continue
		}
		if len(serverIDs) > 0 {
			found := false
			for _, sid := range serverIDs {
				if sid == p.ServerID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(metricNames) > 0 {
			found := false
			for _, mn := range metricNames {
				if mn == p.MetricName {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		out = append(out, p)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// ErrNotFound matches pgx.ErrNoRows for callers that switch on it.
var ErrNotFound = errNotFound{}

type errNotFound struct{}

func (errNotFound) Error() string { return "not found" }

// FromPgxErr translates pgx errors into repository errors.
// Returns nil if err is nil.
func FromPgxErr(err error) error {
	if err == nil {
		return nil
	}
	if err.Error() == "no rows in result set" {
		return ErrNotFound
	}
	return err
}

// nullIfEmpty returns nil for an empty string so Postgres stores NULL
// instead of rejecting the empty value (required for inet columns).
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
