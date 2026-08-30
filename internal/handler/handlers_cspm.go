// Tier 7 Phase 2 — CSPM (D7) — Cloud Security Posture Management surface.
//
//	GET /api/v1/cspm/resources — list cloud resources (filter ?provider=&type=)
//	GET /api/v1/cspm/findings  — list misconfigurations (filter ?severity=&resolved=)
//
// scanResource() is a stub that returns a hard-coded set of sample
// findings so the page has something to display. Real scanning +
// remediation land in a future change. Every query honors tenant_id
// from the JWT.
package handler

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// allowedCSPMProviders — defense in depth at the API edge. Spec
// defines these five values; anything else falls back to "any" so
// the column never holds garbage.
var allowedCSPMProviders = map[string]bool{
	"proxmox": true, "truenas": true, "aws": true, "gcp": true, "azure": true,
}

// allowedCSPMResourceTypes — same defense in depth for resource_type.
var allowedCSPMResourceTypes = map[string]bool{
	"vm": true, "lxc": true, "dataset": true, "bucket": true, "volume": true,
}

// allowedCSPMSeverities — severity taxonomy (matches security_threats
// plus an "info" floor for CSPM-specific notices).
var allowedCSPMSeverities = map[string]bool{
	"info": true, "low": true, "medium": true, "high": true, "critical": true,
}

// cspmResourceRow is the JSON shape returned for a single resource.
type cspmResourceRow struct {
	ID            string `json:"id"`
	Provider      string `json:"provider"`
	ResourceType  string `json:"resource_type"`
	ResourceID    string `json:"resource_id"`
	Name          string `json:"name"`
	Region        string `json:"region,omitempty"`
	Config        string `json:"config"` // raw JSON string
	LastScannedAt string `json:"last_scanned_at"`
}

// cspmFindingRow is the JSON shape returned for a single finding.
type cspmFindingRow struct {
	ID          string  `json:"id"`
	ResourceID  string  `json:"resource_id"`
	Severity    string  `json:"severity"`
	FindingType string  `json:"finding_type"`
	Description string  `json:"description"`
	Remediation string  `json:"remediation,omitempty"`
	DetectedAt  string  `json:"detected_at"`
	ResolvedAt  *string `json:"resolved_at,omitempty"`
}

// ListCSPMResources returns cloud resources for the caller's tenant,
// newest first. Optional filters: ?provider=&type=.
func ListCSPMResources(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		provider := strings.ToLower(strings.TrimSpace(c.Query("provider")))
		resType := strings.ToLower(strings.TrimSpace(c.Query("type")))
		limit := clampLimit(c.Query("limit"), 100, 500)

		args := []any{tenantID}
		q := `SELECT id::text, provider, resource_type, resource_id,
		             name, COALESCE(region, ''),
		             COALESCE(config::text, '{}'),
		             last_scanned_at::text
		      FROM cspm_resources
		      WHERE tenant_id = $1`
		if allowedCSPMProviders[provider] {
			args = append(args, provider)
			q += " AND provider = $" + itoa(len(args))
		}
		if allowedCSPMResourceTypes[resType] {
			args = append(args, resType)
			q += " AND resource_type = $" + itoa(len(args))
		}
		args = append(args, limit)
		q += " ORDER BY last_scanned_at DESC LIMIT $" + itoa(len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []cspmResourceRow{}
		for rows.Next() {
			var r cspmResourceRow
			if err := rows.Scan(&r.ID, &r.Provider, &r.ResourceType,
				&r.ResourceID, &r.Name, &r.Region,
				&r.Config, &r.LastScannedAt); err != nil {
				continue
			}
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{"resources": out, "total": len(out)})
	}
}

// ListCSPMFindings returns misconfigurations for the caller's tenant,
// newest first. Optional filters:
//   - ?severity=critical|high|medium|low|info
//   - ?resolved=true|false (default: false)
func ListCSPMFindings(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		severity := strings.ToLower(strings.TrimSpace(c.Query("severity")))
		resolved := strings.ToLower(strings.TrimSpace(c.Query("resolved")))
		limit := clampLimit(c.Query("limit"), 100, 500)

		args := []any{tenantID}
		q := `SELECT id::text, resource_id, severity, finding_type,
		             description, COALESCE(remediation, ''),
		             detected_at::text, resolved_at::text
		      FROM cspm_findings
		      WHERE tenant_id = $1`
		if allowedCSPMSeverities[severity] {
			args = append(args, severity)
			q += " AND severity = $" + itoa(len(args))
		}
		if resolved == "true" {
			q += " AND resolved_at IS NOT NULL"
		} else if resolved == "false" {
			q += " AND resolved_at IS NULL"
		}
		args = append(args, limit)
		q += " ORDER BY detected_at DESC LIMIT $" + itoa(len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []cspmFindingRow{}
		for rows.Next() {
			var f cspmFindingRow
			var resolvedAt *string
			if err := rows.Scan(&f.ID, &f.ResourceID, &f.Severity,
				&f.FindingType, &f.Description, &f.Remediation,
				&f.DetectedAt, &resolvedAt); err != nil {
				continue
			}
			f.ResolvedAt = resolvedAt
			out = append(out, f)
		}
		kernel.RespondOK(c, gin.H{"findings": out, "total": len(out)})
	}
}

// scanResource is a stub that evaluates posture rules against a single
// resource and returns a sample finding set. In Phase 3 this returns
// hard-coded realistic findings so the page has content; future
// changes will plug in real rule engines.
//
// Returns at least one finding per call so callers have something to
// render. tenantID is accepted for symmetry with future API but is
// unused in the stub.
func scanResource(_ context.Context, _ *db.Pool, tenantID uuid.UUID, resource cspmResourceRow) []cspmFindingRow {
	_ = tenantID
	findings := []cspmFindingRow{}

	// Default-credential sweep applies to every resource type.
	findings = append(findings, cspmFindingRow{
		ResourceID:  resource.ResourceID,
		Severity:    "high",
		FindingType: "weak_credential",
		Description: "Default credentials detected",
		Remediation: "Rotate all default passwords",
		DetectedAt:  resource.LastScannedAt,
	})

	// VM-specific: open SSH port to the world.
	if resource.ResourceType == "vm" {
		findings = append(findings, cspmFindingRow{
			ResourceID:  resource.ResourceID,
			Severity:    "high",
			FindingType: "open_port",
			Description: "Port 22 open to 0.0.0.0/0",
			Remediation: "Restrict SSH access via firewall rules",
			DetectedAt:  resource.LastScannedAt,
		})
	}

	// Dataset/volume-specific: unencrypted at rest.
	if resource.ResourceType == "dataset" || resource.ResourceType == "volume" {
		findings = append(findings, cspmFindingRow{
			ResourceID:  resource.ResourceID,
			Severity:    "medium",
			FindingType: "unencrypted_volume",
			Description: "ZFS dataset not encrypted at rest",
			Remediation: "Enable ZFS native encryption",
			DetectedAt:  resource.LastScannedAt,
		})
	}

	return findings
}
