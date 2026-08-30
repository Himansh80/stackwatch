// Tier 10 Phase 3 — Personal Notes + Todos (H3). List + create
// endpoints for the per-user todo surface (homelab_todos).
//
//	GET  /api/v1/homelab/todos  — ListHomelabTodos
//	POST /api/v1/homelab/todos  — CreateHomelabTodo
//
// PATCH /todos/:id and DELETE /todos/:id live in
// handlers_homelab_todos_extra.go (split for file-size discipline
// — this file stays focused on the list-and-create hot path).
// POST /todos/:id/complete and GET /todos/due-soon also live in
// the extras file.
//
// Storage: homelab_todos (migrations/041_homelab.sql). One row
// per todo; per-user (NOT per-tenant) — every WHERE clause filters
// by both tenant_id and user_id so user A can never see or modify
// user B's todos.
package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// Validation helpers — kept small so the file stays under 400 LOC.
// ------------------------------------------------------------------

// validateTodoTitle enforces a non-empty title and a sane length
// cap. Todos without titles break the dashboard's list layout.
// 200 chars is the cap — matches the typical todo title length.
func validateTodoTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("title is required")
	}
	if len(title) > todosMaxTitleChars {
		return fmt.Errorf("title must be %d characters or fewer", todosMaxTitleChars)
	}
	return nil
}

// validateTodoTags enforces sane tag bounds. Same logic as
// validateNoteTags — kept separate (not shared) because the cap
// constants are different (notesMaxTags vs todosMaxTags).
func validateTodoTags(tags []string) error {
	if len(tags) > todosMaxTags {
		return fmt.Errorf("tags must be %d or fewer (got %d)", todosMaxTags, len(tags))
	}
	for i, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			return fmt.Errorf("tags[%d] is empty", i)
		}
		if len(t) > 32 {
			return fmt.Errorf("tags[%d] must be 32 characters or fewer", i)
		}
	}
	return nil
}

// validateTodoPriority enforces the priority whitelist. Empty
// string defaults to 'medium' (the column default); any other
// value must be in allowedTodoPriorities.
func validateTodoPriority(priority string) error {
	if priority == "" {
		return nil // column default is 'medium'
	}
	if !allowedTodoPriorities[priority] {
		return fmt.Errorf("priority must be one of: low, medium, high, urgent")
	}
	return nil
}

// validateTodoDueDate enforces sane due_date bounds: must be a
// parseable RFC3339-ish string. Empty string means no due date.
// We do NOT enforce "must be in the future" — the user might
// want to log a past-due todo (or backfill one from paper).
func validateTodoDueDate(raw string) error {
	if raw == "" {
		return nil // nullable
	}
	// Light parse: try a couple of common shapes. Postgres
	// itself accepts RFC3339 (with timezone) and the SQL
	// default. We use time.Parse with RFC3339Nano as the
	// canonical form.
	_, err := parseFlexibleDate(raw)
	if err != nil {
		return fmt.Errorf("due_date must be RFC3339 (e.g. 2025-12-31T23:59:00Z): %w", err)
	}
	return nil
}

// scanTodoRow scans one row of the standard todos SELECT into a
// todoRow. Pulled into a helper so all four CRUD handlers share
// the same column order and Scan signature. Note: due_date and
// completed_at come back as nullable strings (empty when NULL)
// so the JSON response omits them instead of emitting empty
// strings.
func scanTodoRow(scan func(...any) error) (todoRow, error) {
	var (
		row                  todoRow
		desc, due, completed *string
		createdAt, updatedAt string
	)
	if err := scan(&row.ID, &row.TenantID, &row.UserID, &row.Title,
		&desc, &row.Priority, &due, &completed, &row.Tags,
		&createdAt, &updatedAt); err != nil {
		return todoRow{}, err
	}
	row.Description = desc
	row.DueDate = due
	row.CompletedAt = completed
	row.CreatedAt = createdAt
	row.UpdatedAt = updatedAt
	if row.Tags == nil {
		row.Tags = []string{}
	}
	return row, nil
}

// todosListOrderBy returns the ORDER BY clause for GET /todos.
// Active todos (completed_at IS NULL) float to the top, then
// due_date ASC (overdue first, then soonest), then created_at DESC.
// Completed todos fall to the bottom, newest-completion-first.
const todosListOrderBy = ` ORDER BY (completed_at IS NULL) DESC, due_date ASC NULLS LAST, created_at DESC`

// ------------------------------------------------------------------
// GET /api/v1/homelab/todos
// ------------------------------------------------------------------

// ListHomelabTodos returns todos for the caller. Honors
// tenant_id + user_id from the JWT — never returns another
// user's todos.
//
// Filters:
//
//	?priority=high|medium|low|urgent   filter by priority
//	?completed=true|false             filter by completion state
//	?due_before=2025-12-31T...        only todos due before this
//	                                  timestamp (open-ended)
//	?limit=N                          cap rows (default 50, max 200)
//
// Default sort: active first, then due date ascending (overdue
// first, soonest after), then creation date descending.
func ListHomelabTodos(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		priority := strings.TrimSpace(c.Query("priority"))
		if priority != "" && !allowedTodoPriorities[priority] {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"priority must be one of: low, medium, high, urgent")
			return
		}
		completedFilter := strings.TrimSpace(c.Query("completed"))
		dueBefore := strings.TrimSpace(c.Query("due_before"))

		limit := todosListLimit
		if l, perr := parsePositiveInt(c.Query("limit"), 1, 200, todosListLimit); perr == nil {
			limit = l
		}

		conds := []string{"tenant_id = $1", "user_id = $2"}
		args := []interface{}{tenantID, userID}
		if priority != "" {
			args = append(args, priority)
			conds = append(conds, fmt.Sprintf("priority = $%d", len(args)))
		}
		if completedFilter == "true" {
			conds = append(conds, "completed_at IS NOT NULL")
		} else if completedFilter == "false" {
			conds = append(conds, "completed_at IS NULL")
		}
		if dueBefore != "" {
			if err := validateTodoDueDate(dueBefore); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			args = append(args, dueBefore)
			conds = append(conds, fmt.Sprintf("due_date <= $%d", len(args)))
		}
		args = append(args, limit)
		q := `SELECT id::text, tenant_id::text, user_id::text, title, description,
		             priority, due_date::text, completed_at::text, tags,
		             created_at::text, updated_at::text
		        FROM homelab_todos
		       WHERE ` + joinStrings(conds, " AND ") +
			todosListOrderBy +
			fmt.Sprintf(" LIMIT $%d", len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := []todoRow{}
		for rows.Next() {
			r, serr := scanTodoRow(rows.Scan)
			if serr != nil {
				kernel.RespondError(c, serr)
				return
			}
			out = append(out, r)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"todos": out,
			"count": len(out),
		})
	}
}

// ------------------------------------------------------------------
// POST /api/v1/homelab/todos
// ------------------------------------------------------------------

// CreateHomelabTodo inserts a new todo for the caller. Validates:
//
//   - title is non-empty (after trim) and ≤ todosMaxTitleChars
//   - priority is empty (→ medium) or in allowedTodoPriorities
//   - due_date is empty (→ NULL) or parseable as RFC3339
//   - tags is ≤ todosMaxTags entries, each non-empty and ≤ 32 chars
//
// Returns 201 with the created row so the UI can render the new
// todo without an extra GET.
func CreateHomelabTodo(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		var req todoReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		req.Title = strings.TrimSpace(req.Title)
		if err := validateTodoTitle(req.Title); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if err := validateTodoPriority(req.Priority); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if req.DueDate != nil {
			if err := validateTodoDueDate(*req.DueDate); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
		}
		if err := validateTodoTags(req.Tags); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if req.Tags == nil {
			req.Tags = []string{}
		}
		priority := req.Priority
		if priority == "" {
			priority = "medium"
		}

		var (
			id, createdAt, updatedAt string
		)
		var dueOut *string
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO homelab_todos
			        (tenant_id, user_id, title, description, priority, due_date, tags)
			 VALUES ($1, $2, $3, $4, $5, $6::timestamptz, $7)
			 RETURNING id::text, due_date::text, created_at::text, updated_at::text`,
			tenantID, userID, req.Title, req.Description, priority,
			nullableTime(req.DueDate), req.Tags,
		).Scan(&id, &dueOut, &createdAt, &updatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondCreated(c, todoRow{
			ID:          id,
			TenantID:    tenantID.String(),
			UserID:      userID.String(),
			Title:       req.Title,
			Description: req.Description,
			Priority:    priority,
			DueDate:     dueOut,
			CompletedAt: nil,
			Tags:        req.Tags,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})
	}
}
