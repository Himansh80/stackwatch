// Tier 7 Phase 2 — Synthetics Full (D5) — Run-now + SLA + results.
//
//	POST /api/v1/synthetics/tests-full/:test_id/run      — synchronous run, returns result_id
//	GET  /api/v1/synthetics/tests-full/:test_id/sla      — uptime + p50/p95/p99 + SLA verdict
//	GET  /api/v1/synthetics/tests-full/:test_id/results  — recent runs (?limit=50)
//
// SLA is computed on demand from the last 100 test_runs rows.
package handler

import (
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// RunSynthTestNow executes a test synchronously via the runner and
// returns the inserted test_run row id.
func RunSynthTestNow(pool *db.Pool, runner *SyntheticsRunner) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("test_id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Verify test exists + belongs to tenant before executing.
		var typ, url string
		var timeoutMs int
		var headers, assertions []byte
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT type, url, timeout_ms,
			        COALESCE(headers::text, '{}'), COALESCE(assertions::text, '[]')
			 FROM synthetics_tests
			 WHERE tenant_id = $1 AND id = $2`,
			tenantID, id,
		).Scan(&typ, &url, &timeoutMs, &headers, &assertions)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		runID, result, runErr := runner.ExecuteSync(c.Request.Context(),
			tenantID, id, typ, url, timeoutMs, headers, assertions)
		if runErr != nil {
			kernel.RespondError(c, runErr)
			return
		}
		kernel.RespondOK(c, gin.H{
			"result_id":     runID.String(),
			"test_id":       id.String(),
			"status":        result.Status,
			"response_ms":   result.ResponseMs,
			"response_code": result.StatusCode,
			"error":         result.Error,
		})
	}
}

// GetSynthTestSLA aggregates last 100 runs into uptime %, p50/p95/p99
// response time, breach count, and whether the test is meeting SLA.
func GetSynthTestSLA(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("test_id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var targetUptime float64
		var targetMs int
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT COALESCE(sla_uptime_pct, 99.9)::float8, COALESCE(sla_response_ms, 1000)
			 FROM synthetics_tests WHERE tenant_id = $1 AND id = $2`,
			tenantID, id,
		).Scan(&targetUptime, &targetMs)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		// Aggregate last 100 runs.
		var totalRuns, passed, failed, timeouts, errors int
		var responseTimes []int
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT
			   COUNT(*),
			   COUNT(*) FILTER (WHERE status = 'pass'),
			   COUNT(*) FILTER (WHERE status = 'fail'),
			   COUNT(*) FILTER (WHERE status = 'timeout'),
			   COUNT(*) FILTER (WHERE status = 'error'),
			   COALESCE(array_agg(duration_ms ORDER BY started_at DESC)
			            FILTER (WHERE duration_ms IS NOT NULL), '{}')
			 FROM (
			   SELECT status, duration_ms, started_at FROM synthetics_test_runs
			   WHERE tenant_id = $1 AND test_id = $2
			   ORDER BY started_at DESC LIMIT 100
			 ) last100`,
			tenantID, id,
		).Scan(&totalRuns, &passed, &failed, &timeouts, &errors, &responseTimes)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		uptimePct := 100.0
		if totalRuns > 0 {
			uptimePct = float64(passed) / float64(totalRuns) * 100.0
		}
		var p50, p95, p99 int
		if len(responseTimes) > 0 {
			sort.Ints(responseTimes)
			p50 = percentile(responseTimes, 0.50)
			p95 = percentile(responseTimes, 0.95)
			p99 = percentile(responseTimes, 0.99)
		}
		breachCount := failed + timeouts + errors
		slaMet := uptimePct >= targetUptime && (p95 == 0 || p95 <= targetMs)
		kernel.RespondOK(c, gin.H{
			"test_id":                id.String(),
			"uptime_pct":             roundTo2(uptimePct),
			"p50_response_ms":        p50,
			"p95_response_ms":        p95,
			"p99_response_ms":        p99,
			"breach_count":           breachCount,
			"sla_target_uptime_pct":  targetUptime,
			"sla_target_response_ms": targetMs,
			"sla_met":                slaMet,
			"total_runs":             totalRuns,
			"passed":                 passed,
			"failed":                 failed,
			"timeouts":               timeouts,
			"errors":                 errors,
		})
	}
}

// GetSynthTestResults returns recent runs for one test (default 50).
func GetSynthTestResults(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("test_id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		limitStr := c.DefaultQuery("limit", "50")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 500 {
			limit = 50
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, started_at::text, duration_ms, status, response_code,
			        COALESCE(error_message, ''), assertions_passed, assertions_failed,
			        COALESCE(response_body_excerpt, '')
			 FROM synthetics_test_runs
			 WHERE tenant_id = $1 AND test_id = $2
			 ORDER BY started_at DESC LIMIT $3`,
			tenantID, id, limit,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []gin.H{}
		for rows.Next() {
			var rid, startedAt, status, errMsg, excerpt string
			var durMs, respCode, assP, assF int
			if err := rows.Scan(&rid, &startedAt, &durMs, &status, &respCode,
				&errMsg, &assP, &assF, &excerpt); err != nil {
				continue
			}
			out = append(out, gin.H{
				"id":                    rid,
				"started_at":            startedAt,
				"duration_ms":           durMs,
				"status":                status,
				"response_code":         respCode,
				"error_message":         errMsg,
				"assertions_passed":     assP,
				"assertions_failed":     assF,
				"response_body_excerpt": excerpt,
			})
		}
		kernel.RespondOK(c, gin.H{"results": out, "total": len(out)})
	}
}

// percentile returns the value at the given percentile (0.0 - 1.0)
// of a pre-sorted ascending int slice.
func percentile(sorted []int, p float64) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)) * p)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// roundTo2 rounds a float to 2 decimal places for stable JSON output.
func roundTo2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}