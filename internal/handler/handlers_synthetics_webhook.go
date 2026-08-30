// Tier 7 Phase 2 — Synthetics Full (D5) — Webhook receiver for external
// test runs (e.g. external probe systems posting results back).
//
//	POST /api/v1/synthetics/webhook
//	  body: {test_id, external_id, status, response_ms, error_message}
//
// Looks up the test by id, verifies tenant ownership, and inserts a
// synthetics_test_runs row carrying the external_id in the
// error_message column (no dedicated column for it in the spec).
//
// NOTE: Unlike other synthetics routes, the webhook accepts an internal
// auth shortcut — it requires the JWT like all other protected routes,
// but does NOT need the test to be enabled (external systems may
// report historical / paused-test results).
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// synthWebhookReq is the JSON shape for POST /synthetics/webhook.
type synthWebhookReq struct {
	TestID       string `json:"test_id" binding:"required,uuid"`
	ExternalID   string `json:"external_id" binding:"max=128"`
	Status       string `json:"status" binding:"required,oneof=pass fail timeout error"`
	ResponseMs   int    `json:"response_ms" binding:"min=0"`
	ResponseCode int    `json:"response_code" binding:"min=0"`
	ErrorMessage string `json:"error_message"`
}

// IngestSynthWebhook stores one externally-reported test run.
func IngestSynthWebhook(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r synthWebhookReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		testID, perr := uuid.Parse(r.TestID)
		if perr != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Verify the test belongs to this tenant before recording.
		var tenantCheck uuid.UUID
		if err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT tenant_id FROM synthetics_tests WHERE id = $1`, testID,
		).Scan(&tenantCheck); err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		if tenantCheck != tenantID {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		// Encode the external_id into error_message so the row carries
		// provenance. Format: "external:<external_id> | <error_message>"
		// when both present; "external:<external_id>" when only id;
		// or just error_message.
		encoded := ""
		if r.ExternalID != "" {
			encoded = "external:" + strings.TrimSpace(r.ExternalID)
		}
		if r.ErrorMessage != "" {
			if encoded != "" {
				encoded += " | "
			}
			encoded += r.ErrorMessage
		}
		var runID uuid.UUID
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO synthetics_test_runs
			   (tenant_id, test_id, started_at, duration_ms, status,
			    response_code, error_message, assertions_passed,
			    assertions_failed, response_body_excerpt)
			 VALUES ($1, $2, now(), $3, $4, $5, $6, 0, 0, '')
			 RETURNING id`,
			tenantID, testID, r.ResponseMs, r.Status,
			r.ResponseCode, nullIfEmpty(encoded),
		).Scan(&runID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"recorded":  true,
			"result_id": runID.String(),
			"test_id":   testID.String(),
			"status":    r.Status,
		})
	}
}
