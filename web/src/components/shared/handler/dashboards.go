// Tier 6 v3 — Dashboard Builder (M10).
//
//	POST   /api/v1/dashboards            — create
//	GET    /api/v1/dashboards            — list
//	GET    /api/v1/dashboards/:id        — get
//	PATCH  /api/v1/dashboards/:id        — update
//	DELETE /api/v1/dashboards/:id        — delete
//	POST   /api/v1/dashboards/:id/eval   — evaluate all panels
//	POST   /api/v1/dashboards/:id/default — mark as default
//	DELETE /api/v1/dashboards/:id/default — unmark
//
// A dashboard is a name + JSONB layout (array of panel objects).
// Panels have {id, type, title, query, grid}. The eval endpoint runs
// each panel's query against the PromQL-lite parser and returns data.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/promql"
)

// Panel is one widget in a dashboard layout.
// Stored as JSONB array element; we type it for read access.
type Panel struct {
	ID    string                 `json:"id"`
	Type  string                 `json:"type"` // "timeseries" | "stat" | "table"
	Title string                 `json:"title"`
	Query string                 `json:"query"` // PromQL-lite
	Grid  map[string]interface{} `json:"grid"`  // {x, y, w, h}
}

// Dashboard is the row in dashboards table (plus parsed layout).
type Dashboard struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Layout      []Panel   `json:"layout"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// createDashboardRequest is the body for POST /dashboards.
type createDashboardRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Layout      []Panel `json:"layout"`
	IsDefault   bool    `json:"is_default"`
}

// patchDashboardRequest is the body for PATCH /dashboards/:id.
type patchDashboardRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Layout      []Panel `json:"layout"`
	IsDefault   *bool   `json:"is_default"`
}

// evalDashboardRequest is the body for POST /dashboards/:id/eval.
type evalDashboardRequest struct {
	From int64 `json:"from"` // unix ms
	To   int64 `json:"to"`   // unix ms
	Step int64 `json:"step"` // ms
}

// evalPanelResult is the per-panel output of /eval.
type evalPanelResult struct {
	PanelID string           `json:"panel_id"`
	Title   string           `json:"title"`
	Type    string           `json:"type"`
	Values  [][2]interface{} `json:"values"` // [[unix_sec, value], ...]
	Error   string           `json:"error,omitempty"`
}

// ListDashboards returns all dashboards for the caller's tenant
// (default first, then by updated_at DESC).
func ListDashboards(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			c.JSON(401, gin.H{"error": "tenant not in context"})
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id, tenant_id, name, description, layout, is_default, created_at, updated_at
			 FROM dashboards WHERE tenant_id = $1
			 ORDER BY is_default DESC, updated_at DESC`, tenantID)
		if err != nil {
			c.JSON(500, gin.H{"error": "query: " + err.Error()})
			return
		}
		defer rows.Close()

		out := []Dashboard{}
		for rows.Next() {
			var d Dashboard
			var layoutRaw []byte
			if err := rows.Scan(&d.ID, &d.TenantID, &d.Name, &d.Description,
				&layoutRaw, &d.IsDefault, &d.CreatedAt, &d.UpdatedAt); err != nil {
				continue
			}
			_ = json.Unmarshal(layoutRaw, &d.Layout)
			if d.Layout == nil {
				d.Layout = []Panel{}
			}
			out = append(out, d)
		}
		c.JSON(200, gin.H{"dashboards": out, "total": len(out)})
	}
}

// CreateDashboard creates a new dashboard.
func CreateDashboard(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			c.JSON(401, gin.H{"error": "tenant not in context"})
			return
		}
		var req createDashboardRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad json: " + err.Error()})
			return
		}
		if req.Layout == nil {
			req.Layout = []Panel{}
		}
		if len(req.Layout) > 50 {
			c.JSON(400, gin.H{"error": "max 50 panels per dashboard"})
			return
		}
		layoutJSON, err := json.Marshal(req.Layout)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid layout: " + err.Error()})
			return
		}

		var id uuid.UUID
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO dashboards (tenant_id, name, description, layout, is_default)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id`,
			tenantID, req.Name, req.Description, layoutJSON, req.IsDefault).Scan(&id)
		if err != nil {
			c.JSON(500, gin.H{"error": "insert: " + err.Error()})
			return
		}
		c.JSON(201, gin.H{
			"id":          id.String(),
			"name":        req.Name,
			"description": req.Description,
			"layout":      req.Layout,
			"is_default":  req.IsDefault,
			"status":      "created",
		})
	}
}

// GetDashboard returns one dashboard.
func GetDashboard(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		tenantID, _ := tenantIDFromContext(c)
		var d Dashboard
		var layoutRaw []byte
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id, tenant_id, name, description, layout, is_default, created_at, updated_at
			 FROM dashboards WHERE id = $1 AND tenant_id = $2`,
			id, tenantID).
			Scan(&d.ID, &d.TenantID, &d.Name, &d.Description,
				&layoutRaw, &d.IsDefault, &d.CreatedAt, &d.UpdatedAt)
		if err != nil {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		_ = json.Unmarshal(layoutRaw, &d.Layout)
		if d.Layout == nil {
			d.Layout = []Panel{}
		}
		c.JSON(200, d)
	}
}

// PatchDashboard updates a dashboard.
func PatchDashboard(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		tenantID, _ := tenantIDFromContext(c)

		var req patchDashboardRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "bad json: " + err.Error()})
			return
		}

		sets := []string{}
		args := []interface{}{}
		if req.Name != nil {
			args = append(args, *req.Name)
			sets = append(sets, fmt.Sprintf("name = $%d", len(args)))
		}
		if req.Description != nil {
			args = append(args, *req.Description)
			sets = append(sets, fmt.Sprintf("description = $%d", len(args)))
		}
		if req.Layout != nil {
			if len(req.Layout) > 50 {
				c.JSON(400, gin.H{"error": "max 50 panels per dashboard"})
				return
			}
			layoutJSON, err := json.Marshal(req.Layout)
			if err != nil {
				c.JSON(400, gin.H{"error": "invalid layout: " + err.Error()})
				return
			}
			args = append(args, layoutJSON)
			sets = append(sets, fmt.Sprintf("layout = $%d", len(args)))
		}
		if req.IsDefault != nil {
			args = append(args, *req.IsDefault)
			sets = append(sets, fmt.Sprintf("is_default = $%d", len(args)))
		}
		if len(sets) == 0 {
			c.JSON(400, gin.H{"error": "no fields to update"})
			return
		}
		sets = append(sets, "updated_at = NOW()")
		args = append(args, id, tenantID)
		q := "UPDATE dashboards SET " + joinStrings(sets, ", ") +
			" WHERE id = $" + fmt.Sprint(len(args)-1) +
			" AND tenant_id = $" + fmt.Sprint(len(args))

		_, err = pool.Pgx().Exec(c.Request.Context(), q, args...)
		if err != nil {
			c.JSON(500, gin.H{"error": "update: " + err.Error()})
			return
		}
		c.JSON(200, gin.H{"id": id.String(), "status": "updated"})
	}
}

// DeleteDashboard deletes a dashboard.
func DeleteDashboard(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		tenantID, _ := tenantIDFromContext(c)
		_, err = pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM dashboards WHERE id = $1 AND tenant_id = $2`,
			id, tenantID)
		if err != nil {
			c.JSON(500, gin.H{"error": "delete: " + err.Error()})
			return
		}
		c.JSON(200, gin.H{"id": id.String(), "status": "deleted"})
	}
}

// EvalDashboard runs all panel queries and returns per-panel data.
//
// Body: {from, to, step} in unix ms (default: last 1h, step 30s).
// Returns: {panel_id, title, type, values, error} per panel.
func EvalDashboard(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		tenantID, _ := tenantIDFromContext(c)

		var req evalDashboardRequest
		_ = c.ShouldBindJSON(&req) // body optional
		step := req.Step
		if step <= 0 {
			step = 30000
		}
		now := time.Now()
		from := req.From
		to := req.To
		if from == 0 || to == 0 {
			from = now.Add(-1 * time.Hour).UnixMilli()
			to = now.UnixMilli()
		}

		// Load dashboard
		var layoutRaw []byte
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT layout FROM dashboards WHERE id = $1 AND tenant_id = $2`,
			id, tenantID).Scan(&layoutRaw)
		if err != nil {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		var panels []Panel
		_ = json.Unmarshal(layoutRaw, &panels)
		if panels == nil {
			panels = []Panel{}
		}

		// Evaluate each panel
		results := []evalPanelResult{}
		for _, p := range panels {
			results = append(results, evalOnePanel(c.Request.Context(), pool, tenantID, p, from, to, step))
		}

		c.JSON(200, gin.H{
			"id":          id.String(),
			"panels":      results,
			"from":        from,
			"to":          to,
			"step":        step,
			"panel_count": len(panels),
		})
	}
}

// evalOnePanel runs one panel's query and returns its data.
func evalOnePanel(ctx context.Context, pool *db.Pool, tenantID uuid.UUID,
	p Panel, fromMs, toMs, stepMs int64) evalPanelResult {
	res := evalPanelResult{
		PanelID: p.ID,
		Title:   p.Title,
		Type:    p.Type,
		Values:  [][2]interface{}{},
	}
	if p.Query == "" {
		res.Error = "empty query"
		return res
	}

	q, err := promql.Parse(p.Query)
	if err != nil {
		res.Error = "parse: " + err.Error()
		return res
	}

	baseSQL, args := q.ToSQL(stepMs)
	from := time.UnixMilli(fromMs)
	to := time.UnixMilli(toMs)
	args = append(args, tenantID, from, to)
	tenantIdx := len(args) - 2
	fromIdx := len(args) - 1
	toIdx := len(args)
	extras := fmt.Sprintf(" AND tenant_id = $%d AND ts >= $%d AND ts < $%d",
		tenantIdx, fromIdx, toIdx)
	var fullSQL string
	if containsString(baseSQL, "GROUP BY") {
		fullSQL = replaceOne(baseSQL, "GROUP BY", extras+" GROUP BY")
	} else {
		fullSQL = replaceOne(baseSQL, "ORDER BY", extras+" ORDER BY")
	}

	rows, err := pool.Pgx().Query(ctx, fullSQL, args...)
	if err != nil {
		res.Error = "query: " + err.Error()
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var ts time.Time
		var v float64
		if err := rows.Scan(&ts, &v); err == nil {
			res.Values = append(res.Values, [2]interface{}{
				float64(ts.UnixMilli()) / 1000, promql.FormatValue(v),
			})
		}
	}
	return res
}

// SetDashboardDefault marks a dashboard as default (and unmarks others).
func SetDashboardDefault(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return
		}
		tenantID, _ := tenantIDFromContext(c)
		// Unset others in same tenant first
		_, _ = pool.Pgx().Exec(c.Request.Context(),
			`UPDATE dashboards SET is_default = false
			 WHERE tenant_id = $1 AND id <> $2`, tenantID, id)
		// Set this one
		_, err = pool.Pgx().Exec(c.Request.Context(),
			`UPDATE dashboards SET is_default = true, updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2`, id, tenantID)
		if err != nil {
			c.JSON(500, gin.H{"error": "update: " + err.Error()})
			return
		}
		c.JSON(200, gin.H{"id": id.String(), "is_default": true})
	}
}

// UnsetDashboardDefault unmarks any default for the tenant.
func UnsetDashboardDefault(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := tenantIDFromContext(c)
		_, err := pool.Pgx().Exec(c.Request.Context(),
			`UPDATE dashboards SET is_default = false
			 WHERE tenant_id = $1 AND is_default = true`, tenantID)
		if err != nil {
			c.JSON(500, gin.H{"error": "update: " + err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "default_unset"})
	}
}

// joinStrings joins a string slice with a separator (avoids extra import).
func joinStrings(s []string, sep string) string {
	out := ""
	for i, x := range s {
		if i > 0 {
			out += sep
		}
		out += x
	}
	return out
}

// containsString is a simple substring check.
func containsString(s, sub string) bool {
	return indexOf(s, sub) >= 0
}

// replaceOne replaces the first occurrence of old with new in s.
func replaceOne(s, old, new string) string {
	idx := indexOf(s, old)
	if idx < 0 {
		return s
	}
	return s[:idx] + new + s[idx+len(old):]
}

// indexOf returns the index of sub in s, or -1 if not found.
func indexOf(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
