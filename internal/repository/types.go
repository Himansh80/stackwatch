// Package repository provides data types shared by the repository implementations.
package repository

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Server matches the DB row for servers table.
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
	DeletedAt     *time.Time      `json:"-"`
}

// CreateServerInput is the validated input for creating a server.
type CreateServerInput struct {
	Name      string          `json:"name"`
	Hostname  string          `json:"hostname"`
	IPAddress string          `json:"ip_address"`
	OS        string          `json:"os"`
	OSVersion string          `json:"os_version"`
	Arch      string          `json:"arch"`
	Tags      json.RawMessage `json:"tags"`
}

// UpdateServerInput is a partial update — nil fields are left alone.
type UpdateServerInput struct {
	Name          *string          `json:"name,omitempty"`
	Hostname      *string          `json:"hostname,omitempty"`
	IPAddress     *string          `json:"ip_address,omitempty"`
	OS            *string          `json:"os,omitempty"`
	OSVersion     *string          `json:"os_version,omitempty"`
	Arch          *string          `json:"arch,omitempty"`
	KernelVersion *string          `json:"kernel_version,omitempty"`
	CPUCores      *int             `json:"cpu_cores,omitempty"`
	CPUModel      *string          `json:"cpu_model,omitempty"`
	MemoryTotal   *int64           `json:"memory_total,omitempty"`
	DiskTotal     *int64           `json:"disk_total,omitempty"`
	Tags          *json.RawMessage `json:"tags,omitempty"`
	Status        *string          `json:"status,omitempty"`
}

// Agent matches the DB row for agents table.
type Agent struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	Name          string          `json:"name"`
	ServerID      *uuid.UUID      `json:"server_id,omitempty"`
	IngestKey     string          `json:"ingest_key,omitempty"`
	KeyHash       string          `json:"-"`
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

// MetricPoint matches a row in metric_points.
type MetricPoint struct {
	ServerID   uuid.UUID       `json:"server_id"`
	MetricName string          `json:"metric_name"`
	Value      float64         `json:"value"`
	Labels     json.RawMessage `json:"labels"`
	Timestamp  time.Time       `json:"timestamp"`
}

// MetricPointInput is for INSERT into metric_points.
type MetricPointInput struct {
	ServerID   uuid.UUID `json:"server_id"`
	MetricName string    `json:"metric_name"`
	Value      float64   `json:"value"`
}
