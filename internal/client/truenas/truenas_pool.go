package truenas

// Pool is a row from pool.query.
type Pool map[string]any

// PoolDisk is one disk inside a pool/vdev.
type PoolDisk map[string]any

// VDev is one vdev in a pool topology.
type VDev map[string]any

// ListPools returns all ZFS pools.
func (c *Client) ListPools() ([]Pool, error) {
	res, err := c.Query("pool.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]Pool, len(res))
	for i, r := range res {
		out[i] = Pool(r.(map[string]any))
	}
	return out, nil
}

// GetPool returns one pool by ID (int64).
func (c *Client) GetPool(id int64) (Pool, error) {
	res, err := c.Query("pool.query", [][]any{{"id", "=", id}})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, &ErrCall{Errname: "ENOENT", Reason: "pool not found"}
	}
	return Pool(res[0].(map[string]any)), nil
}

// PoolCreateOptions matches pool.create args (JSON-RPC shape).
//
//	Name           string   "tank"
//	Encryption     bool
//	Dedup          string   "ON" | "OFF"
//	Topology       map[string]any  {"vdevs": []any{ ... }}
//	AllowRoot      bool
type PoolCreateOptions struct {
	Name       string         `json:"name"`
	Encryption bool           `json:"encryption"`
	Dedup      string         `json:"deduplication,omitempty"`
	Topology   map[string]any `json:"topology"`
}

// CreatePool creates a new pool.
func (c *Client) CreatePool(opts PoolCreateOptions) (int64, error) {
	res, err := c.Call("pool.create", []any{opts})
	if err != nil {
		return 0, err
	}
	if id, ok := res.(float64); ok {
		return int64(id), nil
	}
	return 0, nil
}

// ExtendPool attaches new disks to a pool.
func (c *Client) ExtendPool(poolID int64, newVdevs []map[string]any) error {
	_, err := c.Call("pool.attach", []any{poolID, newVdevs})
	return err
}

// DetachDisk from a pool's vdev.
func (c *Client) DetachDisk(poolID int64, label string) error {
	_, err := c.Call("pool.detach", []any{poolID, label})
	return err
}

// OfflineDisk stops a disk in a pool.
func (c *Client) OfflineDisk(poolID int64, label string) error {
	_, err := c.Call("pool.offline", []any{poolID, label})
	return err
}

// OnlineDisk starts a disk that was offlined.
func (c *Client) OnlineDisk(poolID int64, label string) error {
	_, err := c.Call("pool.online", []any{poolID, label})
	return err
}

// ReplaceDisk swaps a failed disk for a new one.
func (c *Client) ReplaceDisk(poolID int64, label, newDisk string) error {
	_, err := c.Call("pool.replace", []any{poolID, label, newDisk})
	return err
}

// ScrubPool starts a scrub.
func (c *Client) ScrubPool(poolID int64) error {
	_, err := c.Call("pool.scrub.run", []any{poolID})
	return err
}

// PoolScrub returns the most recent scrub state.
func (c *Client) PoolScrub(poolID int64) (map[string]any, error) {
	res, err := c.Query("pool.scrub.query", [][]any{{"pool", "=", poolID}})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return map[string]any{"state": "none"}, nil
	}
	return res[0].(map[string]any), nil
}

// ExportPool exports/destroys a pool.
func (c *Client) ExportPool(poolID int64, destroy, force bool) error {
	opts := map[string]any{
		"destroy": destroy,
		"force":   force,
	}
	_, err := c.Call("pool.export", []any{poolID, opts})
	return err
}

// ImportPool finds + imports a pool by GUID.
func (c *Client) ImportPool(guid string) error {
	_, err := c.Call("pool.import_pool", []any{guid})
	return err
}

// FindImportablePools returns pools found on disks that can be imported.
func (c *Client) FindImportablePools() ([]map[string]any, error) {
	res, err := c.Call("pool.import_find", []any{})
	if err != nil {
		return nil, err
	}
	if arr, ok := res.([]any); ok {
		out := make([]map[string]any, len(arr))
		for i, r := range arr {
			out[i] = r.(map[string]any)
		}
		return out, nil
	}
	return nil, nil
}
