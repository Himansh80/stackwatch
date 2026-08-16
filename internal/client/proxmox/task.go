package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// taskResponse is the shape Proxmox returns for action endpoints (UPID).
type taskResponse struct {
	Data string `json:"data"`
}

// newFormRequestWithContext builds a POST/PUT/DELETE with form-encoded body.
func newFormRequestWithContext(ctx context.Context, method, urlStr string, form url.Values, apiToken string) (*http.Request, error) {
	body := strings.NewReader(form.Encode())
	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// doTask executes a request that returns a Proxmox task UPID.
// The response is wrapped in {"data":"UPID:..."}.
func (c *Client) doTask(req *http.Request) (string, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 401 {
		return "", fmt.Errorf("proxmox: unauthorized (check API token)")
	}
	if resp.StatusCode == 403 {
		return "", fmt.Errorf("proxmox: forbidden (check token privileges): %s", string(body))
	}
	if resp.StatusCode == 400 {
		return "", fmt.Errorf("proxmox: bad request: %s", string(body))
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("proxmox: HTTP %d: %s", resp.StatusCode, string(body))
	}
	var tr taskResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		// Some endpoints return plain data, fall back to raw string.
		return strings.Trim(string(body), "\"' "), nil
	}
	return tr.Data, nil
}

// TaskStatus represents the current state of an async Proxmox task.
type TaskStatus struct {
	Status     string `json:"status"`     // running, stopped
	ExitStatus string `json:"exitstatus"` // OK, ERROR
	UPID       string `json:"upid"`
}

// GetTaskStatus polls a task UPID. status: running|stopped; exitstatus: OK|ERROR.
func (c *Client) GetTaskStatus(ctx context.Context, node, upid string) (*TaskStatus, error) {
	path := fmt.Sprintf("/nodes/%s/tasks/%s/status",
		url.PathEscape(node), url.PathEscape(upid))
	var ts TaskStatus
	if err := c.get(ctx, path, &ts); err != nil {
		return nil, err
	}
	return &ts, nil
}
