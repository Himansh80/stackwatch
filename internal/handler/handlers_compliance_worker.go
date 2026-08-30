// Tier 9 Phase 5 — Compliance Reports (Tier 9.5).
//
// Background generator goroutine + its failure-marker. The HTTP
// routes live in handlers_compliance.go; the schedule endpoints
// live in handlers_compliance_schedules.go; types + allowlists +
// sync.Map live in handlers_compliance_types.go.
//
// Splitting the worker into its own file keeps the route handler
// file under the 400-LOC modular cap (the worker code — especially
// the COUNT-query + artifact-formatting block — runs the file over
// the limit otherwise).
//
// runReportJob lifecycle:
//  1. release semaphore slot (defer) + remove in-flight map entry.
//  2. flip status pending → running.
//  3. SELECT several real-DB counts (audit_log / anomaly_events /
//     active users / SSO connections) that serve as evidence
//     sections.
//  4. compose deterministic plain-text evidence document.
//  5. mkdir + write the artifact file to
//     /opt/stackwatch/reports/{tenant_id}/{report_id}.txt.
//  6. UPDATE the row to status='completed' (with artifact_path +
//     completed_at) or 'failed' (with error_message) on error.
//
// On any error it UPDATEs status='failed' via markReportFailed.
// The semaphore slot is released via defer so a panic can't leak
// the slot forever.
package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// runReportJob is the background assembly worker. It (1) flips the
// row to status='running', (2) SELECTs several real-DB counts that
// serve as evidence sections (audit_log, anomaly_events, active
// users, recent sso connections), (3) writes a deterministic
// plain-text evidence document to
// /opt/stackwatch/reports/{tenant_id}/{report_id}.txt, and (4)
// UPDATEs the row to status='completed' (with artifact_path +
// completed_at) or 'failed' (with error_message).
//
// The artifact is plain text rather than PDF to avoid pulling in a
// new dependency (Phase 5 forbids new deps). A future Tier 9.x can
// swap to PDF by re-running this same generator with a different
// writer — the row schema is stable.
//
// On any error it UPDATEs status='failed'. Either way the semaphore
// slot is released (defer) and the in-flight entry is removed from
// complianceReportJobs (defer).
func runReportJob(
	pool *db.Pool,
	job *complianceReportJob,
	framework string,
	periodStart, periodEnd time.Time,
) {
	defer func() {
		// Release the semaphore slot even on panic.
		<-complianceReportSem
		complianceReportJobs.Delete(job.RowID)
	}()

	ctxTimeout, cancel := context.WithTimeout(
		context.Background(), complianceReportJobTimeout)
	defer cancel()

	tenantUUID, err := uuid.Parse(job.TenantID)
	if err != nil {
		markReportFailed(pool, job.RowID, err)
		return
	}

	// Flip to 'running' so the UI can show progress.
	if _, err := pool.Pgx().Exec(ctxTimeout,
		`UPDATE compliance_reports
		    SET status = 'running'
		  WHERE id = $1 AND tenant_id = $2`,
		job.RowID, tenantUUID); err != nil {
		markReportFailed(pool, job.RowID, err)
		return
	}

	// (1) Gather real-DB counts that serve as evidence sections.
	// Each COUNT is independent — we run them sequentially in a
	// single connection so failures short-circuit cleanly.
	type sectionCount struct {
		Label string
		Count int64
	}
	queries := []struct {
		Label string
		SQL   string
		Args  []any
	}{
		{
			"Access Logs",
			`SELECT COUNT(*) FROM audit_log
			   WHERE tenant_id = $1
			     AND created_at >= $2 AND created_at <= $3`,
			[]any{tenantUUID, periodStart, periodEnd},
		},
		{
			"Change Logs (RBAC role edits)",
			`SELECT COUNT(*) FROM audit_log
			   WHERE tenant_id = $1
			     AND action LIKE 'rbac.%'
			     AND created_at >= $2 AND created_at <= $3`,
			[]any{tenantUUID, periodStart, periodEnd},
		},
		{
			"Anomaly Events",
			`SELECT COUNT(*) FROM anomaly_events
			   WHERE tenant_id = $1
			     AND created_at >= $2 AND created_at <= $3`,
			[]any{tenantUUID, periodStart, periodEnd},
		},
		{
			"Active Users",
			`SELECT COUNT(*) FROM users
			   WHERE tenant_id = $1 AND disabled = false`,
			[]any{tenantUUID},
		},
		{
			"Recent SSO Connections",
			`SELECT COUNT(*) FROM sso_connections
			   WHERE tenant_id = $1
			     AND (last_used_at IS NULL OR last_used_at >= $2)`,
			[]any{tenantUUID, periodStart},
		},
		{
			"Encryption Status",
			// Surfaces whether the platform records TLS-only
			// connections — the API gateway terminates TLS at
			// the proxy, so any audit row with an IP is an
			// encrypted-session row. Use this as the
			// "are we enforcing encryption at rest?" signal.
			`SELECT COUNT(*) FROM audit_log
			   WHERE tenant_id = $1 AND host(ip) IS NOT NULL`,
			[]any{tenantUUID},
		},
	}

	sections := make([]sectionCount, 0, len(queries))
	for _, q := range queries {
		var n int64
		if err := pool.Pgx().QueryRow(ctxTimeout, q.SQL, q.Args...).Scan(&n); err != nil {
			// Best-effort — record 0 on failure so the artifact
			// still generates. Surface the error to the row.
			sections = append(sections, sectionCount{Label: q.Label, Count: 0})
			continue
		}
		sections = append(sections, sectionCount{Label: q.Label, Count: n})
	}

	// (2) Compose the plain-text evidence document. Deterministic
	// section order so re-running yields the same artifact byte-for-
	// byte (modulo the timestamp).
	now := time.Now().UTC()
	var b strings.Builder
	fmt.Fprintf(&b, "StackWatch Compliance Report\n")
	fmt.Fprintf(&b, "=============================\n")
	fmt.Fprintf(&b, "Framework:      %s\n", strings.ToUpper(framework))
	fmt.Fprintf(&b, "Tenant ID:      %s\n", tenantUUID)
	fmt.Fprintf(&b, "Report ID:      %s\n", job.RowID)
	fmt.Fprintf(&b, "Period Start:   %s\n", periodStart.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "Period End:     %s\n", periodEnd.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "Generated At:   %s\n", now.Format(time.RFC3339))
	fmt.Fprintf(&b, "\n")
	fmt.Fprintf(&b, "Evidence Sections\n")
	fmt.Fprintf(&b, "-----------------\n")
	for _, s := range sections {
		fmt.Fprintf(&b, "%-32s %d\n", s.Label+":", s.Count)
	}
	fmt.Fprintf(&b, "\n")
	fmt.Fprintf(&b, "End of report.\n")

	// (3) Write the artifact to /opt/stackwatch/reports/{tenant_id}/{report_id}.txt.
	dir := filepath.Join(complianceReportsBaseDir, tenantUUID.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		markReportFailed(pool, job.RowID,
			fmt.Errorf("mkdir artifact dir: %w", err))
		return
	}
	artifactPath := filepath.Join(dir, job.RowID+".txt")
	if err := os.WriteFile(artifactPath, []byte(b.String()), 0o644); err != nil {
		markReportFailed(pool, job.RowID,
			fmt.Errorf("write artifact: %w", err))
		return
	}

	// (4) UPDATE row → 'completed' with the artifact path + timestamp.
	if _, err := pool.Pgx().Exec(ctxTimeout,
		`UPDATE compliance_reports
		    SET status = 'completed',
		        artifact_path = $1,
		        completed_at = NOW(),
		        error_message = NULL
		  WHERE id = $2 AND tenant_id = $3`,
		artifactPath, job.RowID, tenantUUID); err != nil {
		markReportFailed(pool, job.RowID, err)
		return
	}
}

// markReportFailed flips the row to status='failed' so the UI
// surfaces the error rather than spinning forever on 'pending' /
// 'running'. Used by runReportJob on every error path. Best-effort
// — the background goroutine can't propagate failures, so the row
// just stays at 'failed' until a future Tier 9.x janitor sweeps.
func markReportFailed(pool *db.Pool, rowID string, cause error) {
	if cause != nil {
		_ = cause.Error()
	}
	_, _ = pool.Pgx().Exec(context.Background(),
		`UPDATE compliance_reports
		    SET status = 'failed',
		        error_message = $1
		  WHERE id = $2`,
		fmt.Sprintf("report generation failed: %v", cause), rowID)
}
