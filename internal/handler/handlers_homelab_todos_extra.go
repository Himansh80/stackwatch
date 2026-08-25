// Tier 10 Phase 3 — Personal Notes + Todos (H3). The remaining
// 4 todo handlers split out of handlers_homelab_todos.go to keep
// both files under 400 LOC.
//
//	PATCH  /api/v1/homelab/todos/:id      — PatchHomelabTodo
//	DELETE /api/v1/homelab/todos/:id      — DeleteHomelabTodo
//	POST   /api/v1/homelab/todos/:id/complete — CompleteHomelabTodo
//	GET    /api/v1/homelab/todos/due-soon     — ListDueSoonHomelabTodos
//
// List + Create live in handlers_homelab_todos.go (the hot path).
// All four honor tenant_id + user_id from the JWT — never touches
// another user's todos.
package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// PATCH /api/v1/homelab/todos/:id
// ------------------------------------------------------------------

// PatchHomelabTodo updates a todo. All fields optional — the
// handler builds a dynamic UPDATE with only the supplied fields,
// matching the services/notes PATCH pattern.
//
// 404 when the row does not belong to the caller (or doesn't exist).
// The frontend sees 404 and silently removes the todo from the
// list (the row was deleted by another device).
//
// The MarkComplete field is special: it's a tri-state *bool:
//
//	nil       → don't change completed_at
//	*true     → set completed_at = NOW()  (mark complete)
//	*false    → set completed_at = NULL   (re-open todo)
func PatchHomelabTodo(pool *db.Pool) gin.HandlerFunc {
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
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "id must be a uuid")
			return
		}

		var req todoPatchReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		sets := []string{}
		args := []interface{}{}
		if req.Title != nil {
			t := strings.TrimSpace(*req.Title)
			if err := validateTodoTitle(t); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			args = append(args, t)
			sets = append(sets, fmt.Sprintf("title = $%d", len(args)))
		}
		if req.Description != nil {
			args = append(args, *req.Description)
			sets = append(sets, fmt.Sprintf("description = $%d", len(args)))
		}
		if req.Priority != nil {
			if err := validateTodoPriority(*req.Priority); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			args = append(args, *req.Priority)
			sets = append(sets, fmt.Sprintf("priority = $%d", len(args)))
		}
		if req.DueDate != nil {
			if err := validateTodoDueDate(*req.DueDate); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			args = append(args, *req.DueDate)
			sets = append(sets, fmt.Sprintf("due_date = $%d::timestamptz", len(args)))
		}
		if req.Tags != nil {
			if err := validateTodoTags(*req.Tags); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			tags := *req.Tags
			if tags == nil {
				tags = []string{}
			}
			args = append(args, tags)
			sets = append(sets, fmt.Sprintf("tags = $%d", len(args)))
		}
		if req.MarkComplete != nil {
			if *req.MarkComplete {
				sets = append(sets, "completed_at = NOW()")
			} else {
				sets = append(sets, "completed_at = NULL")
			}
		}
		if len(sets) == 0 {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "no fields to update")
			return
		}
		sets = append(sets, "updated_at = NOW()")
		args = append(args, id, tenantID, userID)
		q := "UPDATE homelab_todos SET " + joinStrings(sets, ", ") +
			" WHERE id = $" + fmt.Sprint(len(args)-2) +
			" AND tenant_id = $" + fmt.Sprint(len(args)-1) +
			" AND user_id = $" + fmt.Sprint(len(args))

		tag, err := pool.Pgx().Exec(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}

		// Return the refreshed row so the UI doesn't have to
		// refetch.
		var (
			rowID, tid, uid, title, priorityStr, createdAt, updatedAt string
			desc, due, completed                                       *string
			tags                                                       []string
		)
		if err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id::text, tenant_id::text, user_id::text, title, description,
			        priority, due_date::text, completed_at::text, tags,
			        created_at::text, updated_at::text
			   FROM homelab_todos
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		).Scan(&rowID, &tid, &uid, &title, &desc, &priorityStr, &due, &completed, &tags, &createdAt, &updatedAt); err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tags == nil {
			tags = []string{}
		}
		kernel.RespondOK(c, todoRow{
			ID:          rowID,
			TenantID:    tid,
			UserID:      uid,
			Title:       title,
			Description: desc,
			Priority:    priorityStr,
			DueDate:     due,
			CompletedAt: completed,
			Tags:        tags,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})
	}
}

// ------------------------------------------------------------------
// DELETE /api/v1/homelab/todos/:id
// ------------------------------------------------------------------

// DeleteHomelabTodo removes a todo for the caller. Idempotent:
// 200 with `{deleted: 0}` when the row doesn't exist (or isn't the
// caller's) — the UI treats "it doesn't exist" and "I deleted it"
// as the same outcome.
//
// We do NOT cascade to other tables — todos are isolated by design.
func DeleteHomelabTodo(pool *db.Pool) gin.HandlerFunc {
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
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "id must be a uuid")
			return
		}

		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM homelab_todos
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondOK(c, gin.H{
			"deleted":        tag.RowsAffected(),
			"todo_id":        id.String(),
			"idempotent_ok": true,
		})
	}
}

// ------------------------------------------------------------------
// POST /api/v1/homelab/todos/:id/complete
// ------------------------------------------------------------------

// CompleteHomelabTodo marks a todo as completed. Sets
// completed_at = NOW() — the handler does not accept a custom
// timestamp (the frontend shouldn't be re-stamping server-side
// events).
//
// Idempotent: re-marking a completed todo refreshes completed_at
// to NOW() (the new "completion moment"). This matches the
// typical homelab-dashboard UX where the user clicks the
// checkbox, navigates away, and the next refresh should reflect
// the most-recent action.
//
// 404 when the row does not belong to the caller (or doesn't exist).
func CompleteHomelabTodo(pool *db.Pool) gin.HandlerFunc {
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
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "id must be a uuid")
			return
		}

		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`UPDATE homelab_todos
			    SET completed_at = NOW(), updated_at = NOW()
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}

		// Return the refreshed row.
		var (
			rowID, tid, uid, title, priorityStr, createdAt, updatedAt string
			desc, due, completed                                       *string
			tags                                                       []string
		)
		if err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id::text, tenant_id::text, user_id::text, title, description,
			        priority, due_date::text, completed_at::text, tags,
			        created_at::text, updated_at::text
			   FROM homelab_todos
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		).Scan(&rowID, &tid, &uid, &title, &desc, &priorityStr, &due, &completed, &tags, &createdAt, &updatedAt); err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tags == nil {
			tags = []string{}
		}
		kernel.RespondOK(c, todoRow{
			ID:          rowID,
			TenantID:    tid,
			UserID:      uid,
			Title:       title,
			Description: desc,
			Priority:    priorityStr,
			DueDate:     due,
			CompletedAt: completed,
			Tags:        tags,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})
	}
}

// ------------------------------------------------------------------
// GET /api/v1/homelab/todos/due-soon
// ------------------------------------------------------------------

// ListDueSoonHomelabTodos returns the caller's active (not yet
// completed) todos with due_date in [now, now + todosDueSoonDays].
// Powers the "Todos due soon" KPI card. Order: due_date ASC, then
// priority DESC (urgent > high > medium > low). Active todos only
// (completed_at IS NULL).
func ListDueSoonHomelabTodos(pool *db.Pool) gin.HandlerFunc {
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

		limit := todosListLimit
		if l, perr := parsePositiveInt(c.Query("limit"), 1, 200, todosListLimit); perr == nil {
			limit = l
		}

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, tenant_id::text, user_id::text, title, description,
			        priority, due_date::text, completed_at::text, tags,
			        created_at::text, updated_at::text
			   FROM homelab_todos
			  WHERE tenant_id = $1 AND user_id = $2
			    AND completed_at IS NULL
			    AND due_date IS NOT NULL
			    AND due_date <= NOW() + ($3::text || ' days')::interval
			  ORDER BY due_date ASC,
			           CASE priority
			               WHEN 'urgent' THEN 4
			               WHEN 'high'   THEN 3
			               WHEN 'medium' THEN 2
			               WHEN 'low'    THEN 1
			               ELSE 0
			           END DESC
			  LIMIT $4`,
			tenantID, userID, todosDueSoonDays, limit,
		)
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

		// Overdue count: a separate cheap query on the same
		// (user_id, due_date) index — cheaper than fetching the
		// timestamps in Go.
		var overdueCount int
		_ = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT COUNT(*) FROM homelab_todos
			  WHERE tenant_id = $1 AND user_id = $2
			    AND completed_at IS NULL
			    AND due_date IS NOT NULL
			    AND due_date < NOW()`,
			tenantID, userID,
		).Scan(&overdueCount)

		kernel.RespondOK(c, gin.H{
			"todos":         out,
			"count":         len(out),
			"overdue_count": overdueCount,
			"window_days":   todosDueSoonDays,
		})
	}
}
