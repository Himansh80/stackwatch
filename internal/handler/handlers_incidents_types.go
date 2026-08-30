// Tier 7 Phase 3 — Service Management (D10). Shared types.
//
// All queries honor tenant_id from the JWT. Severity / status
// enums are enforced at the API edge so the column never holds
// garbage. The four tables backing this surface (incidents,
// war_rooms, postmortems, tasks) are created by the idempotent
// migration migrations/037_servicemgmt.sql.
package handler

import "time"

// Compile-time guard: keep time import even when not used here.
var _ = time.RFC3339

// allowedIncidentSeverities — defense in depth at the API edge.
// Spec defines these four values; anything else is rejected.
var allowedIncidentSeverities = map[string]bool{
	"sev1": true, "sev2": true, "sev3": true, "sev4": true,
}

// allowedIncidentStatuses — incident lifecycle taxonomy.
var allowedIncidentStatuses = map[string]bool{
	"open": true, "acknowledged": true, "resolved": true,
}

// allowedTaskStatuses — task progress taxonomy.
var allowedTaskStatuses = map[string]bool{
	"todo": true, "in_progress": true, "done": true,
}

// incidentRow is the JSON shape returned for a single incident.
type incidentRow struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Description  string  `json:"description,omitempty"`
	Severity     string  `json:"severity"`
	Status       string  `json:"status"`
	CommanderID  *string `json:"commander_id,omitempty"`
	StartedAt    string  `json:"started_at"`
	ResolvedAt   *string `json:"resolved_at,omitempty"`
	PostmortemID *string `json:"postmortem_id,omitempty"`
}

// incidentReq is the JSON shape for POST /incidents.
type incidentReq struct {
	Title       string `json:"title"       binding:"required,min=1,max=256"`
	Description string `json:"description" binding:"max=8192"`
	Severity    string `json:"severity"    binding:"required,oneof=sev1 sev2 sev3 sev4"`
}

// warRoomRow is the JSON shape returned for a war room.
type warRoomRow struct {
	ID           string `json:"id"`
	IncidentID   string `json:"incident_id"`
	ChannelURL   string `json:"channel_url"`
	Participants []any  `json:"participants"`
	CreatedAt    string `json:"created_at"`
}

// warRoomReq is the JSON shape for POST /incidents/:id/war-room.
type warRoomReq struct {
	ChannelURL   string `json:"channel_url"   binding:"required,min=1,max=1024"`
	Participants []any  `json:"participants"`
}

// postmortemRow is the JSON shape returned for a postmortem.
type postmortemRow struct {
	ID          string  `json:"id"`
	IncidentID  string  `json:"incident_id"`
	Content     string  `json:"content"`
	AuthorID    *string `json:"author_id,omitempty"`
	PublishedAt *string `json:"published_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

// postmortemReq is the JSON shape for POST /incidents/:id/postmortem.
type postmortemReq struct {
	Content string `json:"content" binding:"required,min=1,max=65536"`
}

// taskRow is the JSON shape returned for a single task.
type taskRow struct {
	ID         string  `json:"id"`
	IncidentID *string `json:"incident_id,omitempty"`
	Title      string  `json:"title"`
	AssigneeID *string `json:"assignee_id,omitempty"`
	Status     string  `json:"status"`
	DueAt      *string `json:"due_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

// taskReq is the JSON shape for POST /incidents/:id/tasks.
type taskReq struct {
	Title      string `json:"title"       binding:"required,min=1,max=256"`
	AssigneeID string `json:"assignee_id"`
	Status     string `json:"status"      binding:"omitempty,oneof=todo in_progress done"`
	DueAt      string `json:"due_at"`
}

// taskUpdateReq is the JSON shape for PUT /incidents/:id/tasks/:task_id.
// All fields optional — partial updates allowed.
type taskUpdateReq struct {
	Title      *string `json:"title"`
	AssigneeID *string `json:"assignee_id"`
	Status     *string `json:"status" binding:"omitempty,oneof=todo in_progress done"`
	DueAt      *string `json:"due_at"`
}
