package truenas

// CloudCredential is one row from cloudsync.credentials.query.
type CloudCredential map[string]any

// ListCloudCredentials returns all configured provider credentials.
func (c *Client) ListCloudCredentials() ([]CloudCredential, error) {
	res, err := c.Query("cloudsync.credentials.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]CloudCredential, len(res))
	for i, r := range res {
		out[i] = CloudCredential(r.(map[string]any))
	}
	return out, nil
}

// GetCloudCredential returns one credential by id.
func (c *Client) GetCloudCredential(id int64) (CloudCredential, error) {
	res, err := c.Query("cloudsync.credentials.query", [][]any{{"id", "=", id}})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, &ErrCall{Errname: "ENOENT", Reason: "credential not found"}
	}
	return CloudCredential(res[0].(map[string]any)), nil
}

// CloudCredentialCreate matches cloudsync.credentials.create args.
// Provider values: "S3" | "GOOGLE_CLOUD_STORAGE" | "AZURE_BLOB" |
// "BACKBLAZE_B2" | "BOX" | "DROPBOX" | "FTP" | "GOOGLE_DRIVE" |
// "HTTPCLIENT" | "MEGA" | "ONEDRIVE" | "PCCLIENT" | "PROTON" |
// "PURL" | "SFTP" | "SMB" | "STORJ" | "UFILES" | "WEBDAV" | "YANDEX"
type CloudCredentialCreate struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	// Attributes is opaque, provider-specific (e.g. access_key, secret_key, etc.)
	Attributes map[string]any `json:"attributes"`
}

// CreateCloudCredential adds a new provider credential.
func (c *Client) CreateCloudCredential(req CloudCredentialCreate) (int64, error) {
	res, err := c.Call("cloudsync.credentials.create", []any{req})
	if err != nil {
		return 0, err
	}
	if id, ok := res.(float64); ok {
		return int64(id), nil
	}
	return 0, nil
}

// UpdateCloudCredential modifies credentials.
func (c *Client) UpdateCloudCredential(id int64, req CloudCredentialCreate) error {
	_, err := c.Call("cloudsync.credentials.update", []any{id, req})
	return err
}

// DeleteCloudCredential removes a credential.
func (c *Client) DeleteCloudCredential(id int64) error {
	_, err := c.Call("cloudsync.credentials.delete", []any{id})
	return err
}

// VerifyCloudCredential tests that the credentials still work.
func (c *Client) VerifyCloudCredential(id int64) (bool, error) {
	res, err := c.Call("cloudsync.credentials.verify", []any{id})
	if err != nil {
		return false, err
	}
	if b, ok := res.(bool); ok {
		return b, nil
	}
	return false, nil
}

// CloudSyncTask is one row from cloudsync.query.
type CloudSyncTask map[string]any

// ListCloudSyncTasks returns all cloud sync tasks.
func (c *Client) ListCloudSyncTasks() ([]CloudSyncTask, error) {
	res, err := c.Query("cloudsync.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]CloudSyncTask, len(res))
	for i, r := range res {
		out[i] = CloudSyncTask(r.(map[string]any))
	}
	return out, nil
}

// CloudSyncTaskCreate matches cloudsync.create args.
type CloudSyncTaskCreate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Direction   string `json:"direction"` // PUSH | PULL
	Path        string `json:"path"`
	Credential  int64  `json:"credentials"`
	Bucket      string `json:"bucket"`
	Folder      string `json:"folder,omitempty"`
	Encryption  bool   `json:"encryption"`
	Schedule    string `json:"schedule,omitempty"` // cron
	Enabled     bool   `json:"enabled"`
}

// CreateCloudSyncTask adds a new task.
func (c *Client) CreateCloudSyncTask(req CloudSyncTaskCreate) (int64, error) {
	res, err := c.Call("cloudsync.create", []any{req})
	if err != nil {
		return 0, err
	}
	if id, ok := res.(float64); ok {
		return int64(id), nil
	}
	return 0, nil
}

// UpdateCloudSyncTask changes a task.
func (c *Client) UpdateCloudSyncTask(id int64, req CloudSyncTaskCreate) error {
	_, err := c.Call("cloudsync.update", []any{id, req})
	return err
}

// DeleteCloudSyncTask removes a task.
func (c *Client) DeleteCloudSyncTask(id int64) error {
	_, err := c.Call("cloudsync.delete", []any{id})
	return err
}

// RunCloudSyncTask immediately runs a task.
func (c *Client) RunCloudSyncTask(id int64) (int64, error) {
	res, err := c.Call("cloudsync.run", []any{id})
	if err != nil {
		return 0, err
	}
	if jid, ok := res.(float64); ok {
		return int64(jid), nil
	}
	return 0, nil
}

// DryRunCloudSyncTask previews what a sync would do.
func (c *Client) DryRunCloudSyncTask(id int64) (int64, error) {
	res, err := c.Call("cloudsync.run_onetime", []any{id, true})
	if err != nil {
		return 0, err
	}
	if jid, ok := res.(float64); ok {
		return int64(jid), nil
	}
	return 0, nil
}
