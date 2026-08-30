// Tier 7 Phase 3 — CI/CD Visibility (D8). Shared types.
//
// Pipeline runs + deployments linked to APM services. Mirrors the
// Datadog Pipelines surface: each pipeline is a single run (one SHA,
// one branch, one outcome) and each deployment ties a pipeline to an
// APM service so we can correlate deploys with downstream regressions.
package handler

import "time"

// allowedCICDProviders — defense in depth at the API edge. Spec
// defines these four values; anything else is rejected so the column
// never holds garbage.
var allowedCICDProviders = map[string]bool{
	"github": true, "gitlab": true, "jenkins": true, "circleci": true,
}

// allowedCICDStatuses — pipeline status taxonomy.
var allowedCICDStatuses = map[string]bool{
	"pending": true, "running": true, "success": true, "failed": true, "cancelled": true,
}

// cicdPipelineRow is the JSON shape returned for a single pipeline.
type cicdPipelineRow struct {
	ID         string  `json:"id"`
	Provider   string  `json:"provider"`
	Repo       string  `json:"repo"`
	Branch     string  `json:"branch"`
	CommitSHA  string  `json:"commit_sha"`
	Status     string  `json:"status"`
	StartedAt  string  `json:"started_at"`
	FinishedAt *string `json:"finished_at,omitempty"`
	DurationMS int     `json:"duration_ms"`
}

// cicdPipelineReq is the JSON shape for POST /cicd/pipelines.
type cicdPipelineReq struct {
	Provider   string `json:"provider"   binding:"required,oneof=github gitlab jenkins circleci"`
	Repo       string `json:"repo"       binding:"required,min=1,max=256"`
	Branch     string `json:"branch"     binding:"max=256"`
	CommitSHA  string `json:"commit_sha" binding:"max=64"`
	Status     string `json:"status"     binding:"required,oneof=pending running success failed cancelled"`
	FinishedAt string `json:"finished_at"`
	DurationMS int    `json:"duration_ms"`
}

// cicdDeploymentRow is the JSON shape returned for a single deployment.
type cicdDeploymentRow struct {
	ID          string `json:"id"`
	PipelineID  string `json:"pipeline_id"`
	ServiceID   string `json:"service_id"`
	ServiceName string `json:"service_name,omitempty"`
	Environment string `json:"environment"`
	Version     string `json:"version"`
	DeployedAt  string `json:"deployed_at"`
}

// cicdDeploymentReq is the JSON shape for POST /cicd/deployments.
type cicdDeploymentReq struct {
	PipelineID  string `json:"pipeline_id" binding:"required,uuid"`
	ServiceID   string `json:"service_id"  binding:"required,uuid"`
	Environment string `json:"environment" binding:"required,max=64"`
	Version     string `json:"version"     binding:"required,max=128"`
}

// ghPushEvent is the subset of GitHub's push event payload we care
// about. Real GitHub payloads include a lot more; we ignore it.
type ghPushEvent struct {
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Ref        string `json:"ref"`
	HeadCommit struct {
		ID        string `json:"id"`
		Timestamp string `json:"timestamp"`
	} `json:"head_commit"`
	Conclusion string `json:"conclusion"` // success | failure | cancelled
}

// glPushEvent is GitLab's push event shape.
type glPushEvent struct {
	Project struct {
		PathWithNamespace string `json:"path_with_namespace"`
	} `json:"project"`
	Ref         string `json:"ref"`
	CheckoutSHA string `json:"checkout_sha"`
	ObjectKind  string `json:"object_kind"`
}

// Compile-time guard: keep time import even when only used in webhooks.
var _ = time.RFC3339
