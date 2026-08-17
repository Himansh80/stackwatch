package truenas

// Snapshot is one row from pool.snapshot.query.
type Snapshot map[string]any

// ListSnapshots returns all snapshots, optionally filtered by dataset.
func (c *Client) ListSnapshots(dataset string) ([]Snapshot, error) {
	filters := [][]any{}
	if dataset != "" {
		filters = append(filters, []any{"dataset", "=", dataset})
	}
	res, err := c.Query("pool.snapshot.query", filters)
	if err != nil {
		return nil, err
	}
	out := make([]Snapshot, len(res))
	for i, r := range res {
		out[i] = Snapshot(r.(map[string]any))
	}
	return out, nil
}

// GetSnapshot returns a single snapshot by name like "tank/data@auto-2024-01-15_00-00".
func (c *Client) GetSnapshot(name string) (Snapshot, error) {
	res, err := c.Query("pool.snapshot.query", [][]any{{"id", "=", name}})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, &ErrCall{Errname: "ENOENT", Reason: "snapshot not found"}
	}
	return Snapshot(res[0].(map[string]any)), nil
}

// SnapshotCreate matches pool.snapshot.create args.
type SnapshotCreate struct {
	Dataset   string `json:"dataset"`
	Name      string `json:"name,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
	VMSync    bool   `json:"vmware_sync,omitempty"`
}

// CreateSnapshot creates a new snapshot.
func (c *Client) CreateSnapshot(req SnapshotCreate) (string, error) {
	res, err := c.Call("pool.snapshot.create", []any{req})
	if err != nil {
		return "", err
	}
	if m, ok := res.(map[string]any); ok {
		if n, ok := m["name"].(string); ok {
			return n, nil
		}
	}
	return "", nil
}

// DeleteSnapshot removes a snapshot.
func (c *Client) DeleteSnapshot(name string, recursive bool) error {
	opts := map[string]any{"recursive": recursive}
	_, err := c.Call("pool.snapshot.delete", []any{name, opts})
	return err
}

// CloneSnapshot clones a snapshot into a new dataset.
func (c *Client) CloneSnapshot(name, newDataset string, props map[string]any) error {
	args := []any{name, newDataset, props}
	_, err := c.Call("pool.snapshot.clone", args)
	return err
}

// -- Replications --

// Replication is one row from replication.query.
type Replication map[string]any

// ListReplications returns all replication tasks.
func (c *Client) ListReplications() ([]Replication, error) {
	res, err := c.Query("replication.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]Replication, len(res))
	for i, r := range res {
		out[i] = Replication(r.(map[string]any))
	}
	return out, nil
}

// ReplicationCreate matches replication.create args.
type ReplicationCreate struct {
	Name          string         `json:"name"`
	Direction     string         `json:"direction"` // PUSH | PULL
	SourceDataset []string       `json:"source_datasets"`
	TargetDataset string         `json:"target_dataset"`
	Recursive     bool           `json:"recursive"`
	Transport     string         `json:"transport"` // SSH | LOCAL | LEGACY
	SSHConnection int64          `json:"ssh_connection,omitempty"`
	Schedule      string         `json:"schedule,omitempty"` // cron string
	Enabled       bool           `json:"enabled"`
	Properties    map[string]any `json:"properties,omitempty"` // recursive=..., extra=...
}

// CreateReplication adds a new replication task.
func (c *Client) CreateReplication(req ReplicationCreate) (int64, error) {
	res, err := c.Call("replication.create", []any{req})
	if err != nil {
		return 0, err
	}
	if id, ok := res.(float64); ok {
		return int64(id), nil
	}
	return 0, nil
}

// UpdateReplication modifies a replication task.
func (c *Client) UpdateReplication(id int64, req ReplicationCreate) error {
	_, err := c.Call("replication.update", []any{id, req})
	return err
}

// DeleteReplication removes a replication task.
func (c *Client) DeleteReplication(id int64) error {
	_, err := c.Call("replication.delete", []any{id})
	return err
}

// RunReplication immediately executes a replication task.
func (c *Client) RunReplication(id int64) (int64, error) {
	res, err := c.Call("replication.run", []any{id})
	if err != nil {
		return 0, err
	}
	if jid, ok := res.(float64); ok {
		return int64(jid), nil
	}
	return 0, nil
}
