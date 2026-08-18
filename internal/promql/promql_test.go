package promql

import (
	"strings"
	"testing"
)

func TestParse_BareMetric(t *testing.T) {
	q, err := Parse("cpu_pct")
	if err != nil {
		t.Fatal(err)
	}
	if q.Metric != "cpu_pct" {
		t.Errorf("metric = %q, want cpu_pct", q.Metric)
	}
	if q.Agg != "" {
		t.Errorf("agg = %q, want empty", q.Agg)
	}
	if len(q.Labels) != 0 {
		t.Errorf("labels = %v, want empty", q.Labels)
	}
}

func TestParse_Agg(t *testing.T) {
	q, err := Parse("avg(cpu_pct)")
	if err != nil {
		t.Fatal(err)
	}
	if q.Metric != "cpu_pct" {
		t.Errorf("metric = %q, want cpu_pct", q.Metric)
	}
	if q.Agg != "avg" {
		t.Errorf("agg = %q, want avg", q.Agg)
	}
}

func TestParse_LabelFilter(t *testing.T) {
	q, err := Parse(`cpu_pct{host_id="abc-123"}`)
	if err != nil {
		t.Fatal(err)
	}
	if q.Metric != "cpu_pct" {
		t.Errorf("metric = %q", q.Metric)
	}
	if q.Labels["host_id"] != "abc-123" {
		t.Errorf("labels[host_id] = %q, want abc-123", q.Labels["host_id"])
	}
}

func TestParse_AggWithLabel(t *testing.T) {
	q, err := Parse(`max(cpu_pct{host_id="abc"})`)
	if err != nil {
		t.Fatal(err)
	}
	if q.Agg != "max" {
		t.Errorf("agg = %q", q.Agg)
	}
	if q.Metric != "cpu_pct" {
		t.Errorf("metric = %q", q.Metric)
	}
	if q.Labels["host_id"] != "abc" {
		t.Errorf("labels[host_id] = %q", q.Labels["host_id"])
	}
}

func TestParse_Empty(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Error("expected error for empty query")
	}
}

func TestParse_InvalidSyntax(t *testing.T) {
	_, err := Parse("avg(cpu_pct")
	if err == nil {
		t.Error("expected error for unbalanced parens")
	}
}

func TestParse_UnsupportedAgg(t *testing.T) {
	_, err := Parse("rate(cpu_pct)")
	if err == nil {
		t.Error("expected error for unsupported aggregation")
	}
}

func TestToSQL_BareMetric(t *testing.T) {
	q, _ := Parse("cpu_pct")
	sql, args := q.ToSQL(30000)
	if !strings.Contains(sql, "metric_name = $1") {
		t.Errorf("SQL missing metric filter: %s", sql)
	}
	if len(args) != 1 || args[0] != "cpu_pct" {
		t.Errorf("args = %v, want [cpu_pct]", args)
	}
}

func TestToSQL_WithLabel(t *testing.T) {
	q, _ := Parse(`cpu_pct{host_id="abc"}`)
	sql, args := q.ToSQL(30000)
	if !strings.Contains(sql, "labels->>'host_id' = $2") {
		t.Errorf("SQL missing label filter: %s", sql)
	}
	if len(args) != 2 {
		t.Errorf("args len = %d, want 2", len(args))
	}
}

func TestToSQL_Agg(t *testing.T) {
	q, _ := Parse("avg(cpu_pct)")
	sql, args := q.ToSQL(30000)
	if !strings.Contains(sql, "AVG(value::numeric)") {
		t.Errorf("SQL missing AVG: %s", sql)
	}
	// GROUP BY must contain the full bucket expression (not the alias)
	if !strings.Contains(sql, "GROUP BY to_timestamp(FLOOR(EXTRACT(EPOCH FROM ts) / 30) * 30)") {
		t.Errorf("SQL missing full bucket expression in GROUP BY: %s", sql)
	}
	if len(args) != 1 {
		t.Errorf("args len = %d", len(args))
	}
}
