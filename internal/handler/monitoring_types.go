package handler

// monitoring_types.go — shared types for Tier 0.5 monitoring.
// Server, Agent, request bodies, and the getTenantID helper.

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Server represents a monitored server.
type Server struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	AgentID       *uuid.UUID      `json:"agent_id,omitempty"`
	Name          string          `json:"name"`
	Hostname      string          `json:"hostname"`
	IPAddress     string          `json:"ip_address"`
	OS            string          `json:"os"`
	OSVersion     string          `json:"os_version"`
	Arch          string          `json:"arch"`
	KernelVersion string          `json:"kernel_version"`
	CPUCores      int             `json:"cpu_cores"`
	CPUModel      string          `json:"cpu_model"`
	MemoryTotal   int64           `json:"memory_total"`
	DiskTotal     int64           `json:"disk_total"`
	Tags          json.RawMessage `json:"tags"`
	Status        string          `json:"status"`
	LastSeenAt    *time.Time      `json:"last_seen_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// ServerCreateRequest is the JSON body for POST /servers.
type ServerCreateRequest struct {
	Name          string          `json:"name" binding:"required,max=255"`
	Hostname      string          `json:"hostname" binding:"max=255"`
	IPAddress     string          `json:"ip_address" binding:"omitempty,ip"`
	OS            string          `json:"os" binding:"max=100"`
	OSVersion     string          `json:"os_version" binding:"max=100"`
	Arch          string          `json:"arch" binding:"max=50"`
	KernelVersion string          `json:"kernel_version" binding:"max=100"`
	CPUCores      int             `json:"cpu_cores" binding:"omitempty,min=1,max=1024"`
	CPUModel      string          `json:"cpu_model" binding:"max=255"`
	MemoryTotal   int64           `json:"memory_total" binding:"omitempty,min=0"`
	DiskTotal     int64           `json:"disk_total" binding:"omitempty,min=0"`
	Tags          json.RawMessage `json:"tags"`
}

// ServerUpdateRequest is the JSON body for PATCH /servers/:id (all fields optional).
type ServerUpdateRequest struct {
	Name          *string          `json:"name,omitempty" binding:"omitempty,max=255"`
	Hostname      *string          `json:"hostname,omitempty" binding:"omitempty,max=255"`
	IPAddress     *string          `json:"ip_address,omitempty" binding:"omitempty,ip"`
	OS            *string          `json:"os,omitempty" binding:"omitempty,max=100"`
	OSVersion     *string          `json:"os_version,omitempty" binding:"omitempty,max=100"`
	Arch          *string          `json:"arch,omitempty" binding:"omitempty,max=50"`
	KernelVersion *string          `json:"kernel_version,omitempty" binding:"omitempty,max=100"`
	CPUCores      *int             `json:"cpu_cores,omitempty" binding:"omitempty,min=1,max=1024"`
	CPUModel      *string          `json:"cpu_model,omitempty" binding:"omitempty,max=255"`
	MemoryTotal   *int64           `json:"memory_total,omitempty" binding:"omitempty,min=0"`
	DiskTotal     *int64           `json:"disk_total,omitempty" binding:"omitempty,min=0"`
	Tags          *json.RawMessage `json:"tags,omitempty"`
	Status        *string          `json:"status,omitempty" binding:"omitempty,oneof=up down stale unknown"`
}

// Agent represents a monitoring agent (enrolled binary on a server).
type Agent struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	Name          string          `json:"name"`
	ServerID      *uuid.UUID      `json:"server_id,omitempty"`
	IngestKey     string          `json:"ingest_key,omitempty"`
	KeyHash       string          `json:"-"` // never serialized
	Status        string          `json:"status"`
	Version       string          `json:"version"`
	OS            string          `json:"os"`
	Arch          string          `json:"arch"`
	Labels        json.RawMessage `json:"labels"`
	LastSeenAt    *time.Time      `json:"last_seen_at,omitempty"`
	LastHeartbeat *time.Time      `json:"last_heartbeat,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// AgentCreateRequest is the body for POST /agents/enroll.
type AgentCreateRequest struct {
	Name   string          `json:"name" binding:"required,max=255"`
	Labels json.RawMessage `json:"labels"`
}

// IngestHeartbeatRequest is the agent's periodic POST body.
// Agent must provide either server_id (looked up) OR hostname (auto-registers if missing).
type IngestHeartbeatRequest struct {
	ServerID   uuid.UUID              `json:"server_id,omitempty"`
	AgentID    uuid.UUID              `json:"agent_id,omitempty"`
	Hostname   string                 `json:"hostname,omitempty"`
	CPU        float64                `json:"cpu,omitempty"`
	Memory     float64                `json:"memory,omitempty"`
	Disk       float64                `json:"disk,omitempty"`
	Network    float64                `json:"network,omitempty"`
	Load       float64                `json:"load,omitempty"`
	Uptime     int64                  `json:"uptime,omitempty"`
	Processes  int                    `json:"processes,omitempty"`
	Containers int                    `json:"containers,omitempty"`
	Metrics    map[string]float64     `json:"metrics,omitempty"`
	Info       map[string]interface{} `json:"info,omitempty"`
}

// MetricPointRequest is a single metric sample (used in batch ingest).
type MetricPointRequest struct {
	ServerID   uuid.UUID       `json:"server_id" binding:"required"`
	MetricName string          `json:"metric_name" binding:"required,max=255"`
	Value      float64         `json:"value" binding:"required"`
	Labels     json.RawMessage `json:"labels"`
	Timestamp  *time.Time      `json:"timestamp,omitempty"`
}

// MetricPointsBatchRequest is the body for POST /metrics/batch (multi-point ingest).
type MetricPointsBatchRequest struct {
	Points []MetricPointRequest `json:"points" binding:"required,min=1,max=10000"`
}

// MetricQueryRequest is the body for POST /metrics/query.
type MetricQueryRequest struct {
	ServerIDs   []uuid.UUID `json:"server_ids,omitempty" binding:"max=100"`
	MetricNames []string    `json:"metric_names" binding:"required,min=1,max=50"`
	Start       *time.Time  `json:"start,omitempty"`
	End         *time.Time  `json:"end,omitempty"`
	Interval    string      `json:"interval,omitempty"`
	Limit       int         `json:"limit,omitempty" binding:"omitempty,min=1,max=10000"`
}

// LogQueryRequest is the body for POST /logs/query.
type LogQueryRequest struct {
	ServerIDs []uuid.UUID `json:"server_ids,omitempty" binding:"max=100"`
	Levels    []string    `json:"levels,omitempty"`
	Query     string      `json:"query,omitempty"`
	Start     *time.Time  `json:"start,omitempty"`
	End       *time.Time  `json:"end,omitempty"`
	Limit     int         `json:"limit,omitempty" binding:"omitempty,min=1,max=10000"`
	Offset    int         `json:"offset,omitempty" binding:"omitempty,min=0"`
}

// getTenantID returns the caller's tenant id from the gin context.
// Set by RequireAuth middleware; falls back to userFromContext for robustness.
// Returns (tenantID, true) on success; (uuid.Nil, false) otherwise.
func getTenantID(c *gin.Context) (uuid.UUID, bool) {
	if v, ok := c.Get("auth.tenant_id"); ok {
		if tid, ok := v.(uuid.UUID); ok && tid != uuid.Nil {
			return tid, true
		}
	}
	// Fallback: try from user context
	if user, ok := userFromContext(c); ok {
		return user.TenantID, true
	}
	return uuid.Nil, false
}

// join concatenates elems with sep (stdlib strings.Join replacement, pre-generic).
func join(elems []string, sep string) string {
	if len(elems) == 0 {
		return ""
	}
	result := elems[0]
	for i := 1; i < len(elems); i++ {
		result += sep + elems[i]
	}
	return result
}
