package proxmox

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// BackupJob is a Proxmox scheduled vzdump job.
type BackupJob struct {
	ID        string `json:"id"`             // unique job id (e.g. "daily-2am")
	Schedule  string `json:"schedule"`       // calendar expression (e.g. "mon..fri 02:00")
	Storage   string `json:"storage"`        // target storage (local, nfs, etc.)
	Type      string `json:"type,omitempty"` // "vzdump" always
	VMID      string `json:"vmid,omitempty"`  // comma-separated VMIDs or "all"
	Mode      string `json:"mode,omitempty"` // "snapshot" | "suspend" | "stop"
	Enabled   int    `json:"enabled,omitempty"`  // 1 or 0
	NextRun   int64  `json:"next-run,omitempty"` // unix epoch
	Compress  int    `json:"compress,omitempty"`
	MaxDays   int    `json:"maxdays,omitempty"`   // retention in days
	Notify    string `json:"notification,omitempty"` // notification mode
	Comment   string `json:"comment,omitempty"`
}

// ListBackupJobs returns all scheduled backup jobs.
func (c *Client) ListBackupJobs(ctx context.Context) ([]BackupJob, error) {
	var out []BackupJob
	if err := c.get(ctx, "/cluster/backup", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetBackupJob returns one scheduled backup job.
func (c *Client) GetBackupJob(ctx context.Context, jobID string) (BackupJob, error) {
	var out BackupJob
	path := "/cluster/backup/" + url.PathEscape(jobID)
	if err := c.get(ctx, path, &out); err != nil {
		return out, err
	}
	return out, nil
}

// CreateBackupJob creates a new scheduled backup job.
func (c *Client) CreateBackupJob(ctx context.Context, job BackupJob) (string, error) {
	form := url.Values{}
	if job.ID == "" {
		return "", fmt.Errorf("id required")
	}
	form.Set("id", job.ID)
	form.Set("schedule", job.Schedule)
	form.Set("storage", job.Storage)
	if job.VMID != "" {
		form.Set("vmid", job.VMID)
	}
	if job.Mode != "" {
		form.Set("mode", job.Mode)
	} else {
		form.Set("mode", "snapshot")
	}
	if job.Compress > 0 {
		form.Set("compress", strconv.Itoa(job.Compress))
	}
	if job.MaxDays > 0 {
		form.Set("maxdays", strconv.Itoa(job.MaxDays))
	}
	if job.Notify != "" {
		form.Set("notification", job.Notify)
	}
	if job.Enabled == 0 {
		form.Set("enabled", "0")
	}
	if job.Comment != "" {
		form.Set("comment", job.Comment)
	}
	if err := c.postForm(ctx, "/cluster/backup", form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// UpdateBackupJob modifies a backup job's settings.
func (c *Client) UpdateBackupJob(ctx context.Context, jobID string, job BackupJob) (string, error) {
	form := url.Values{}
	if job.Schedule != "" {
		form.Set("schedule", job.Schedule)
	}
	if job.Storage != "" {
		form.Set("storage", job.Storage)
	}
	if job.Mode != "" {
		form.Set("mode", job.Mode)
	}
	if job.VMID != "" {
		form.Set("vmid", job.VMID)
	}
	if job.Compress > 0 {
		form.Set("compress", strconv.Itoa(job.Compress))
	}
	if job.MaxDays > 0 {
		form.Set("maxdays", strconv.Itoa(job.MaxDays))
	}
	if job.Comment != "" {
		form.Set("comment", job.Comment)
	}
	// enabled is a special case — always send when explicit
	// Proxmox schema rejects "disable" property but accepts "enabled".
	// To disable, omit nothing — the call accepts partial fields.
	if job.Enabled == 1 {
		form.Set("enabled", "1")
	}
	// job.Enabled == 0: Proxmox PUT requires no "disable" field; omit it.
	// (For full disable, use DELETE + re-create with enabled=0.)
	if err := c.putForm(ctx, "/cluster/backup/"+url.PathEscape(jobID), form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// DeleteBackupJob removes a backup job.
func (c *Client) DeleteBackupJob(ctx context.Context, jobID string) (string, error) {
	if err := c.deleteForm(ctx, "/cluster/backup/"+url.PathEscape(jobID)); err != nil {
		return "", err
	}
	return "", nil
}

// BackupNow starts an immediate backup of a VM/LXC.
// Returns UPID for polling status.
// VMID can be a single int or comma-separated list.
// All other fields are optional; defaults to snapshot/suspend+stop.
func (c *Client) BackupNow(ctx context.Context, node string, vmid int, storage string, mode string, options url.Values) (string, error) {
	form := url.Values{}
	form.Set("vmid", strconv.Itoa(vmid))
	if storage != "" {
		form.Set("storage", storage)
	}
	if mode != "" {
		form.Set("mode", mode)
	}
	for k, v := range options {
		for _, val := range v {
			form.Add(k, val)
		}
	}
	u := c.baseURL + "/api2/json/nodes/" + url.PathEscape(node) + "/vzdump"
	req, err := newFormRequestWithContext(ctx, "POST", u, form, c.apiToken)
	if err != nil {
		return "", err
	}
	return c.doTask(req)
}