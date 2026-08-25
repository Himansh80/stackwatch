// Tier 11 Phase 8 — Platform Health (PL8).
//
// handlers_platform_health_forecast.go — capacity-exhaustion
// linear regression helper.
//
// Pulled out of handlers_platform_health.go to keep that
// route-handler file under the 400-LOC cap. The math is
// self-contained so a future ML-based forecaster (Tier 12+)
// can swap forecastExhaustion without touching the route
// handlers.

package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/stackwatch/platform/internal/db"
)

// forecastExhaustion runs a simple linear model on the last 30
// daily averages of `column` and returns an ISO8601 timestamp
// when the trend would cross `cap`. Empty string means "no
// signal" (worker has <2 days of data, slope <= 0, or already
// exhausted).
//
// Used by HealthCapacityForecast for the 3 PL8 capacity
// dimensions (disk, db_connections, api throughput). The math
// is intentionally cheap (no matrix library, just sums) — a
// future Tier can replace it with proper time-series
// forecasting without changing the JSON contract.
//
// Why "no signal" returns "": the dashboard renders an empty
// row instead of a misleading "1970-01-01" or "now()", so the
// caller MUST distinguish the two states. An empty string is
// the easiest serialization-stable signal.
func forecastExhaustion(ctx context.Context, pool *db.Pool, column string, cap int64) string {
	// Daily averages over the last 30 days. NULL → skipped by AVG.
	sql := fmt.Sprintf(`
	   SELECT date_trunc('day', snapshot_at) AS day,
	          AVG(%s)::float8 AS avg_val,
	          COUNT(*)::int   AS samples
	     FROM platform_health_snapshots
	    WHERE snapshot_at >= NOW() - INTERVAL '30 days'
	    GROUP BY day
	    ORDER BY day`, column)
	rows, err := pool.Pgx().Query(ctx, sql)
	if err != nil {
		return ""
	}
	defer rows.Close()

	type sample struct {
		avg float64
	}
	var samples []sample
	for rows.Next() {
		var (
			day      time.Time
			avg      *float64
			samplesN int
		)
		if err := rows.Scan(&day, &avg, &samplesN); err != nil {
			return ""
		}
		if avg == nil {
			continue
		}
		samples = append(samples, sample{avg: *avg})
	}
	if err := rows.Err(); err != nil {
		return ""
	}
	if len(samples) < 2 {
		// Less than 2 days of data — can't draw a line.
		return ""
	}

	// Simple linear regression y = mx + b, where x is the day
	// index (0..N-1) and y is the avg_val. Then solve for x
	// at y = cap.
	n := float64(len(samples))
	var sumX, sumY, sumXY, sumXX float64
	for i, s := range samples {
		x := float64(i)
		sumX += x
		sumY += s.avg
		sumXY += x * s.avg
		sumXX += x * x
	}
	denom := n*sumXX - sumX*sumX
	if denom == 0 {
		// All x values identical — undefined slope.
		return ""
	}
	slope := (n*sumXY - sumX*sumY) / denom
	if slope <= 0 {
		// Trend is flat or declining — no exhaustion predicted.
		return ""
	}
	intercept := (sumY - slope*sumX) / n
	if intercept >= float64(cap) {
		// Already at or above the cap — exhausted.
		return ""
	}
	daysToCap := (float64(cap) - intercept) / slope
	if daysToCap > 365*5 {
		// Cap is more than 5 years away — not actionable.
		return ""
	}
	exhaustAt := time.Now().UTC().Add(time.Duration(daysToCap*24) * time.Hour)
	return exhaustAt.Format(time.RFC3339)
}