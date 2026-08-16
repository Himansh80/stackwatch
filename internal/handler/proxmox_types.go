package handler

import (
	"github.com/google/uuid"
	"github.com/stackwatch/platform/internal/kernel"
)

// ProxmoxHost is the JSON shape for /proxmox/hosts.
type ProxmoxHost struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Name           string    `json:"name"`
	BaseURL        string    `json:"base_url"`
	APIToken       string    `json:"-"`  // never expose in JSON
	VerifyTLS      bool      `json:"verify_tls"`
	NodeName       string    `json:"node_name,omitempty"`
	Status         string    `json:"status"`
	LastCheckAt    string    `json:"last_check_at,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
	CreatedAt      string    `json:"created_at"`
}

// ProxmoxNode is a discovered Proxmox cluster node.
type ProxmoxNode struct {
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	UptimeSeconds  int64   `json:"uptime_seconds"`
	CPUCount       int     `json:"cpu_count"`
	CPUUsage       float64 `json:"cpu_usage"`
	MemTotal       int64   `json:"mem_total"`
	MemUsed        int64   `json:"mem_used"`
	DiskTotal      int64   `json:"disk_total"`
	DiskUsed       int64   `json:"disk_used"`
}

// ProxmoxVM is a discovered VM or LXC container.
type ProxmoxVM struct {
	Node          string  `json:"node"`
	VMID          int     `json:"vmid"`
	Name          string  `json:"name"`
	Kind          string  `json:"kind"`  // qemu or lxc
	Status        string  `json:"status"`
	CPUCount      int     `json:"cpu_count"`
	CPUUsage      float64 `json:"cpu_usage"`
	MemTotal      int64   `json:"mem_total"`
	MemUsed       int64   `json:"mem_used"`
	DiskTotal     int64   `json:"disk_total"`
	DiskUsed      int64   `json:"disk_used"`
	NetIn         int64   `json:"net_in"`
	NetOut        int64   `json:"net_out"`
	UptimeSeconds int64   `json:"uptime_seconds"`
	Tags          string  `json:"tags,omitempty"`
}

// ProxmoxStorage is a Proxmox storage pool.
type ProxmoxStorage struct {
	Name      string  `json:"name"`
	Kind      string  `json:"kind"`
	Status    string  `json:"status"`
	TotalBytes int64  `json:"total_bytes"`
	UsedBytes  int64  `json:"used_bytes"`
	AvailBytes int64  `json:"avail_bytes"`
	UsagePct   float64 `json:"usage_pct"`
}

// ensure kernel import isn't dropped (used elsewhere)
var _ = kernel.PlanFree

// unused import guard for uuid
var _ uuid.UUID
