package truenas

// Disk is one row from disk.query.
type Disk map[string]any

// ListDisks returns all physical disks on the host.
func (c *Client) ListDisks() ([]Disk, error) {
	res, err := c.Query("disk.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]Disk, len(res))
	for i, r := range res {
		out[i] = Disk(r.(map[string]any))
	}
	return out, nil
}

// GetDisk returns one disk by name (e.g. "sda").
func (c *Client) GetDisk(name string) (Disk, error) {
	res, err := c.Query("disk.query", [][]any{{"name", "=", name}})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, &ErrCall{Errname: "ENOENT", Reason: "disk not found"}
	}
	return Disk(res[0].(map[string]any)), nil
}

// UpdateDisk changes disk metadata (serial, model, rotationrate, etc.).
type DiskUpdate struct {
	Description string `json:"description,omitempty"`
}

// UpdateDisk mutates an existing disk's metadata.
func (c *Client) UpdateDisk(name string, u DiskUpdate) error {
	_, err := c.Call("disk.update", []any{name, u})
	return err
}

// WipeDisk starts a background secure-wipe of a disk.
// Pass quick=true for a non-secure quick wipe.
func (c *Client) WipeDisk(name string, quick bool) error {
	opts := map[string]any{"quick": quick}
	_, err := c.Call("disk.wipe", []any{name, opts})
	return err
}

// DiskDetails returns full S.M.A.R.T. + temp details for one disk.
func (c *Client) DiskDetails(name string) (map[string]any, error) {
	res, err := c.Call("disk.details", []any{name})
	if err != nil {
		return nil, err
	}
	return res.(map[string]any), nil
}

// DiskTemperatures returns latest temperatures as a map name->celsius.
func (c *Client) DiskTemperatures() (map[string]float64, error) {
	res, err := c.Call("disk.temperatures", []any{})
	if err != nil {
		return nil, err
	}
	m, ok := res.(map[string]any)
	if !ok {
		return nil, nil
	}
	out := map[string]float64{}
	for k, v := range m {
		if f, ok := v.(float64); ok {
			out[k] = f
		}
	}
	return out, nil
}
