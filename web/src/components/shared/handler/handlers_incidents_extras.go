// Tier 7 Phase 3 — Service Management (D10). War rooms + postmortems.
//
// Routes:
//
//	POST /api/v1/incidents/:id/war-room     — open a comms channel
//	POST /api/v1/incidents/:id/postmortem   — attach a blameless write-up
//
// War rooms store the channel URL plus a jsonb participants array
// (user ids + optional roles). Postmortems hold free-form text plus
// author_id + published_at so the UI can flag "draft" vs "published".
//
// Both endpoints verify the parent incident belongs to the caller's
// tenant before inserting, so cross-tenant FK smuggling is impossible.
package handler

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// CreateWarRoom opens a war room for an incident. Returns 404 if the
// incident isn't in the caller's tenant.
func CreateWarRoom(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var r warRoomReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Tenant ownership check on the parent incident.
		var ownCount int
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT COUNT(*) FROM incidents WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&ownCount)
		if err != nil || ownCount != 1 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		// Default participants to [] if omitted.
		participants := r.Participants
		if participants == nil {
			participants = []any{}
		}
		partsJSON, err := json.Marshal(participants)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var row warRoomRow
		var rawParts []byte
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO war_rooms (tenant_id, incident_id, channel_url, participants)
			 VALUES ($1, $2, $3, $4::jsonb)
			 RETURNING id::text, incident_id::text, channel_url, participants, created_at::text`,
			tenantID, id, strings.TrimSpace(r.ChannelURL), string(partsJSON),
		).Scan(&row.ID, &row.IncidentID, &row.ChannelURL, &rawParts, &row.CreatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		// Decode the jsonb back into a slice for the response.
		row.Participants = []any{}
		_ = json.Unmarshal(rawParts, &row.Participants)
		kernel.RespondCreated(c, gin.H{"war_room": row})
	}
}

// CreatePostmortem attaches a postmortem to an incident. author_id is
// taken from the JWT; published_at stays NULL until the author
// explicitly publishes the write-up (Phase 3 doesn't expose a
// publish endpoint yet — the UI uses PATCH or a follow-up).
func CreatePostmortem(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		claims, ok := auth.ClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var r postmortemReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Tenant ownership check on the parent incident.
		var ownCount int
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT COUNT(*) FROM incidents WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&ownCount)
		if err != nil || ownCount != 1 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		var row postmortemRow
		var author *string
		var published *string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO postmortems (tenant_id, incident_id, content, author_id)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id::text, incident_id::text, content,
			           author_id::text, published_at::text, created_at::text`,
			tenantID, id, r.Content, claims.UserID,
		).Scan(&row.ID, &row.IncidentID, &row.Content,
			&author, &published, &row.CreatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		row.AuthorID = author
		row.PublishedAt = published
		// Backfill the incident.postmortem_id pointer so the detail
		// page can show "Postmortem →" without a follow-up query.
		_, _ = pool.Pgx().Exec(c.Request.Context(),
			`UPDATE incidents SET postmortem_id = $1
			 WHERE id = $2 AND tenant_id = $3`,
			uuid.MustParse(row.ID), id, tenantID)
		kernel.RespondCreated(c, gin.H{"postmortem": row})
	}
}
