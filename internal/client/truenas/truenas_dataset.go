package truenas

import "encoding/json"

// Dataset is one row from pool.dataset.query.
type Dataset map[string]any

// ListDatasets returns all datasets.
func (c *Client) ListDatasets(poolName string) ([]Dataset, error) {
	filters := [][]any{}
	if poolName != "" {
		filters = append(filters, []any{"pool", "=", poolName})
	}
	res, err := c.Query("pool.dataset.query", filters)
	if err != nil {
		return nil, err
	}
	out := make([]Dataset, len(res))
	for i, r := range res {
		out[i] = Dataset(r.(map[string]any))
	}
	return out, nil
}

// GetDataset returns one dataset by id (string of "tank/data" shape, or numeric id).
func (c *Client) GetDataset(id string) (Dataset, error) {
	res, err := c.Query("pool.dataset.query", [][]any{{"id", "=", id}})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, &ErrCall{Errname: "ENOENT", Reason: "dataset not found"}
	}
	return Dataset(res[0].(map[string]any)), nil
}

// DatasetCreateOptions is the body for pool.dataset.create.
// `Name` and `Type` (FILESYSTEM or VOLUME) are required; remaining fields
// map directly to ZFS properties. Compression is a simple string
// ("LZ4" | "OFF" etc.). For full control, fill Extra with raw map entries.
// MarshalJSON omits the Extra field when nil so the request payload
// matches the TrueNAS schema (no extra keys in either FILESYSTEM or
// VOLUME union variants). When Extra is non-nil its entries are merged
// into the top-level JSON object (raw pass-through).
func (d DatasetCreateOptions) MarshalJSON() ([]byte, error) {
	if d.Extra == nil {
		// Marshal only the named fields. We avoid the `inline` tag
		// because that would still emit `"Extra":null`.
		return json.Marshal(struct {
			Name        string `json:"name"`
			Type        string `json:"type"`
			Compression string `json:"compression,omitempty"`
			Quota       *int64 `json:"quota,omitempty"`
			RefQuota    *int64 `json:"refquota,omitempty"`
			RecordSize  int64  `json:"recordsize,omitempty"`
			Encryption  *bool  `json:"encryption,omitempty"`
			ACLMode     string `json:"aclmode,omitempty"`
		}{
			Name:        d.Name,
			Type:        d.Type,
			Compression: d.Compression,
			Quota:       d.Quota,
			RefQuota:    d.RefQuota,
			RecordSize:  d.RecordSize,
			Encryption:  d.Encryption,
			ACLMode:     d.ACLMode,
		})
	}
	m := map[string]any{
		"name": d.Name,
		"type": d.Type,
	}
	if d.Compression != "" {
		m["compression"] = d.Compression
	}
	if d.Quota != nil {
		m["quota"] = *d.Quota
	}
	if d.RefQuota != nil {
		m["refquota"] = *d.RefQuota
	}
	if d.RecordSize != 0 {
		m["recordsize"] = d.RecordSize
	}
	if d.Encryption != nil {
		m["encryption"] = *d.Encryption
	}
	if d.ACLMode != "" {
		m["aclmode"] = d.ACLMode
	}
	for k, v := range d.Extra {
		m[k] = v
	}
	return json.Marshal(m)
}

type DatasetCreateOptions struct {
	Name        string
	Type        string
	Compression string
	Quota       *int64
	RefQuota    *int64
	RecordSize  int64
	Encryption  *bool
	ACLMode     string
	Extra       map[string]any // pass-through user properties; nil = omitted
}

// CreateDataset creates a dataset.
func (c *Client) CreateDataset(opts DatasetCreateOptions) (string, error) {
	res, err := c.Call("pool.dataset.create", []any{opts})
	if err != nil {
		return "", err
	}
	if m, ok := res.(map[string]any); ok {
		if id, ok := m["id"].(string); ok {
			return id, nil
		}
	}
	return "", nil
}

// DatasetUpdateOptions matches pool.dataset.update.
type DatasetUpdateOptions struct {
	Compression string `json:"compression,omitempty"`
	Quota       int64  `json:"quota,omitempty"`
	RefQuota    int64  `json:"refquota,omitempty"`
	RecordSize  int64  `json:"recordsize,omitempty"`
	ACLMode     string `json:"aclmode,omitempty"`
}

// UpdateDataset changes properties of an existing dataset.
func (c *Client) UpdateDataset(datasetID string, opts DatasetUpdateOptions) error {
	_, err := c.Call("pool.dataset.update", []any{datasetID, opts})
	return err
}

// DeleteDataset destroys a dataset. Recursive delete children too.
func (c *Client) DeleteDataset(datasetID string, recursive bool) error {
	opts := map[string]any{"recursive": recursive, "force": recursive}
	_, err := c.Call("pool.dataset.delete", []any{datasetID, opts})
	return err
}

// Mountpoint is the canonical path of a dataset.
func (d Dataset) Mountpoint() string {
	if m, ok := d["mountpoint"].(string); ok {
		return m
	}
	return ""
}

// Name returns the dataset's name.
func (d Dataset) Name() string {
	if n, ok := d["name"].(string); ok {
		return n
	}
	return ""
}
