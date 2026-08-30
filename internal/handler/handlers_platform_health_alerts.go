// Tier 11 Phase 8 — Platform Health (PL8).
//
// handlers_platform_health_alerts.go — snapshot → alert list
// rule engine.
//
// Pulled out of handlers_platform_health.go to keep that
// route-handler file under the 400-LOC cap. The rule list is
// plain Go so a future config-driven implementation (Tier 12+)
// can swap buildPlatformAlerts without touching the route
// handlers.
//
// Severity tiers (matching Datadog / PagerDuty conventions):
//   critical : service-level outages + >50 open alerts
//   warning  : degraded services + elevated counts
//   info     : general state + minor anomalies
//
// All alerts share the same created_at (= latest snapshot_at)
// so the dashboard can group them under "as of HH:MM" instead
// of trying to reconstruct a per-row timeline.

package handler

import (
	"fmt"
	"time"
)

// platformAlertInput carries the latest snapshot values that
// drive the alert rules. Using a struct (not 12 positional
// args) keeps the call site readable + future-proof.
type platformAlertInput struct {
	Overall          string
	APIUp            bool
	DBUp             bool
	IngestUp         bool
	AlertEngineUp    bool
	AIEngineUp       bool
	WebTerminalUp    bool
	ServersDown      int
	ServersStale     int
	OpenAlerts       int
	FailedLogins24h  int
	BackupCount      int
	ActiveTenants24h int
	SnapshotAt       time.Time
}

// buildPlatformAlerts walks the rule list and returns the
// matching alerts. The output always allocates a non-nil
// slice so the JSON serializer emits [] (not null) for the
// no-alerts case.
func buildPlatformAlerts(in platformAlertInput) healthAlertsResp {
	out := healthAlertsResp{Alerts: []healthAlertRow{}}
	ts := in.SnapshotAt.UTC().Format(time.RFC3339)
	add := func(severity, message, link string) {
		out.Alerts = append(out.Alerts, healthAlertRow{
			Severity:  severity,
			Message:   message,
			CreatedAt: ts,
			Link:      link,
		})
	}

	// Critical.
	if !in.APIUp {
		add("critical", "API gateway reports down", "/dashboard")
	}
	if !in.DBUp {
		add("critical", "Postgres reports down", "/dashboard")
	}
	if !in.IngestUp {
		add("critical", "Ingest worker reports down", "/dashboard")
	}
	if !in.AlertEngineUp {
		add("critical", "Alert engine reports down", "/dashboard")
	}
	if !in.AIEngineUp {
		add("critical", "AI engine reports down", "/dashboard")
	}
	if in.ServersDown > 0 {
		add("critical",
			fmt.Sprintf("%d server(s) reporting down", in.ServersDown),
			"/homelab")
	}
	if in.OpenAlerts > 50 {
		add("critical",
			fmt.Sprintf("%d open alerts (high)", in.OpenAlerts),
			"/incidents")
	}

	// Warning.
	if !in.WebTerminalUp {
		add("warning", "Web terminal reports down", "/dashboard")
	}
	if in.FailedLogins24h > 100 {
		add("warning",
			fmt.Sprintf("%d failed logins in last 24h (high)", in.FailedLogins24h),
			"/security")
	}
	if in.ServersStale > 5 {
		add("warning",
			fmt.Sprintf("%d server(s) stale (>5min no heartbeat)", in.ServersStale),
			"/homelab")
	}
	if in.BackupCount == 0 {
		add("warning", "No backups exist yet", "/platform")
	}
	if in.ActiveTenants24h < 1 && in.Overall != "down" {
		add("warning", "No tenants have authenticated in the last 24h", "/platform")
	}
	if in.OpenAlerts > 10 {
		add("warning",
			fmt.Sprintf("%d open alerts (elevated)", in.OpenAlerts),
			"/incidents")
	}

	// Info.
	if in.FailedLogins24h > 10 {
		add("info",
			fmt.Sprintf("%d failed logins in last 24h", in.FailedLogins24h),
			"/security")
	}
	if in.OpenAlerts > 0 && in.OpenAlerts <= 10 {
		add("info",
			fmt.Sprintf("%d open alerts", in.OpenAlerts),
			"/incidents")
	}
	if in.Overall == "degraded" {
		add("info", "Overall platform status is DEGRADED", "/platform")
	}

	return out
}
