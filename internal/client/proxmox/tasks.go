package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Task is a Proxmox task entry (background operation).
// Tasks track VM/LXC/storage/network operations.
type Task struct {
	UPID       string `json:"upid"`      // unique task id (e.g. "UPID:router:00012345:...")
	Node       string `json:"node"`      // node where the task ran
	PID        int    `json:"pid"`       // process id
	StartTime  int64  `json:"starttime"` // unix epoch seconds
	EndTime    int64  `json:"endtime,omitempty"`
	Type       string `json:"type"`                 // qmsync, vzdump, qmstart, etc.
	User       string `json:"user"`                 // user who started
	Status     string `json:"status"`               // running, OK, WARNING, ERROR, STOPPED, UNKNOWN
	ExitStatus string `json:"exitstatus,omitempty"` // OK or specific error
}

// TaskLogLine is a single line in a task's log output.
// Proxmox returns { "data": [{"t": "log text", "n": 1}, ...], "total": N }
// where the response body is { "data": [...], "total": N } and the
// array of log lines is the "data" key of the response wrapper.
type TaskLogLine struct {
	N int    `json:"n"` // line number (1-indexed)
	T string `json:"t"` // text content
}

// TaskLog is the response from GET /nodes/{node}/tasks/{upid}/log.
// Proxmox returns { total: N, data: [{t, n}, ...] } and we unmarshal
// the inner `data` field as []TaskLogLine.
//
// The wrapper includes the total line count.
type TaskLog struct {
	Total int           `json:"total"` // total lines available
	Data  []TaskLogLine `json:"data,omitempty"`
}

// ListNodeTasks returns the task history for a node (most recent first).
// Limit is 1-1000, default 50.
func (c *Client) ListNodeTasks(ctx context.Context, node string, limit int) ([]Task, error) {
	path := fmt.Sprintf("/nodes/%s/tasks", url.PathEscape(node))
	if limit > 0 {
		path += fmt.Sprintf("?limit=%d", limit)
	}
	var out []Task
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetTaskLog returns the log output of a task (tail, up to 50 lines).
// Proxmox returns {total: N, data: [{t, n}, ...]} — wrapper preserves
// total at top level, so we decode the full response (not via get()).
func (c *Client) GetTaskLog(ctx context.Context, node, upid string, lines int) (TaskLog, error) {
	u := c.baseURL + "/api2/json" + fmt.Sprintf("/nodes/%s/tasks/%s/log",
		url.PathEscape(node), url.PathEscape(upid))
	if lines > 0 {
		u += fmt.Sprintf("?limit=%d", lines)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return TaskLog{}, err
	}
	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return TaskLog{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return TaskLog{}, fmt.Errorf("proxmox HTTP %d: %s", resp.StatusCode, string(body))
	}
	// Proxmox log endpoint: { "total": N, "data": [...] }
	var wrapped struct {
		Total int           `json:"total"`
		Data  []TaskLogLine `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapped); err != nil {
		return TaskLog{}, err
	}
	return TaskLog{Total: wrapped.Total, Data: wrapped.Data}, nil
}

// StopTask cancels a running task (only works on STOPPABLE tasks).
func (c *Client) StopTask(ctx context.Context, node, upid string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/tasks/%s",
		url.PathEscape(node), url.PathEscape(upid))
	if err := c.deleteForm(ctx, path); err != nil {
		return "", err
	}
	return "", nil
}

// ListClusterTasks returns recent tasks across all nodes (cluster-level task list).
func (c *Client) ListClusterTasks(ctx context.Context, limit int) ([]Task, error) {
	path := "/cluster/tasks"
	if limit > 0 {
		path += fmt.Sprintf("?limit=%d", limit)
	}
	var out []Task
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}
