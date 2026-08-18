// Package promql implements a tiny PromQL-lite query language for Tier 6.
//
// Supported syntax (subset of PromQL):
//
//   cpu_pct                       — bare metric
//   avg(cpu_pct)                  — aggregation
//   max(cpu_pct), min, sum, count
//   cpu_pct{host_id="abc"}        — label filter (eq only)
//   avg(cpu_pct{host_id="abc"})
//
// NOT supported (deferred):
//   - rate(), increase(), irate()
//   - regex labels
//   - joins
//   - subqueries
//   - arithmetic operators
//
// The parser is intentionally tiny (~200 LOC) — we don't need the
// full PromQL spec. We just need enough to query our metric_points
// table from the dashboard.
package promql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Query is the parsed representation of a PromQL-lite query.
type Query struct {
	// Metric is the bare metric name (e.g. "cpu_pct").
	Metric string
	// Labels are the label filter (AND-joined). Nil means no filter.
	Labels map[string]string
	// Agg is the aggregation function. "" means no aggregation (raw values).
	Agg string // "avg" | "max" | "min" | "sum" | "count" | ""
}

// Parse parses a query string into a Query. Returns an error on
// unsupported syntax.
func Parse(q string) (*Query, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, fmt.Errorf("empty query")
	}

	out := &Query{Labels: map[string]string{}}

	// Step 1: extract aggregation (optional)
	aggRe := regexp.MustCompile(`^(avg|max|min|sum|count)\((.+)\)$`)
	if m := aggRe.FindStringSubmatch(q); m != nil {
		out.Agg = m[1]
		q = m[2]
	}

	// Step 2: extract metric name + label filter
	// Pattern: metric_name{label="value",label2="value2"}
	metricRe := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_.]*)(?:\{(.*)\})?$`)
	m := metricRe.FindStringSubmatch(q)
	if m == nil {
		return nil, fmt.Errorf("invalid query syntax: %q", q)
	}
	out.Metric = m[1]

	// Step 3: parse labels (if present)
	if m[2] != "" {
		for _, pair := range strings.Split(m[2], ",") {
			pair = strings.TrimSpace(pair)
			if !strings.Contains(pair, "=") {
				return nil, fmt.Errorf("invalid label pair: %q", pair)
			}
			kv := strings.SplitN(pair, "=", 2)
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			// Strip quotes
			if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
				val = val[1 : len(val)-1]
			}
			out.Labels[key] = val
		}
	}

	return out, nil
}

// ToSQL converts a parsed Query into a parameterised SQL query over
// metric_points. Returns the SQL string + args.
//
// BucketSize is the time-bucket size in milliseconds (e.g. 30000).
//
// Without aggregation, we return individual samples.
// With aggregation, we return time-bucketed aggregates.
func (q *Query) ToSQL(bucketMs int64) (string, []interface{}) {
	args := []interface{}{q.Metric}
	where := "metric_name = $1"
	for k, v := range q.Labels {
		args = append(args, v)
		where += fmt.Sprintf(" AND labels->>'%s' = $%d", k, len(args))
	}

	if q.Agg == "" {
		return fmt.Sprintf(`SELECT ts, value FROM metric_points WHERE %s
			ORDER BY ts DESC LIMIT 1000`, where), args
	}

	// Bucketed aggregation
	aggFn := map[string]string{
		"avg":   "AVG(value::numeric)",
		"max":   "MAX(value::numeric)",
		"min":   "MIN(value::numeric)",
		"sum":   "SUM(value::numeric)",
		"count": "COUNT(*)",
	}[q.Agg]
	if aggFn == "" {
		aggFn = "AVG(value::numeric)"
	}

	bucketSec := bucketMs / 1000
	if bucketSec <= 0 {
		bucketSec = 30
	}

	// bucket alias referenced in GROUP BY — must come BEFORE GROUP BY in SELECT list.
	// Postgres needs the full expression in GROUP BY (aliases from SELECT
	// aren't allowed in GROUP BY in standard SQL).
	bucketExpr := fmt.Sprintf("to_timestamp(FLOOR(EXTRACT(EPOCH FROM ts) / %d) * %d)", bucketSec, bucketSec)
	return fmt.Sprintf(`SELECT
			%s AS ts_bucket,
			%s AS v
		FROM metric_points
		WHERE %s
		GROUP BY %s
		ORDER BY ts_bucket ASC`, bucketExpr, aggFn, where, bucketExpr), args
}

// ValidateAgg returns nil if agg is one of the supported aggregations.
func ValidateAgg(agg string) error {
	if agg == "" {
		return nil
	}
	switch agg {
	case "avg", "max", "min", "sum", "count":
		return nil
	}
	return fmt.Errorf("unsupported aggregation: %s", agg)
}

// FormatValue formats a float64 for Prometheus output.
func FormatValue(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
