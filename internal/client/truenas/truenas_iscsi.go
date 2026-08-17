package truenas

// ISCSIExtent is one row from iscsi.extent.query.
type ISCSIExtent map[string]any

// ListISCSIExtents returns all iSCSI extents.
func (c *Client) ListISCSIExtents() ([]ISCSIExtent, error) {
	res, err := c.Query("iscsi.extent.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]ISCSIExtent, len(res))
	for i, r := range res {
		out[i] = ISCSIExtent(r.(map[string]any))
	}
	return out, nil
}

// ISCSIExtentCreate matches iscsi.extent.create args.
type ISCSIExtentCreate struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // DISK | FILE
	Disk      string `json:"disk,omitempty"`
	FilePath  string `json:"path,omitempty"` // API field is `path` for FILE extents
	Blocksize int    `json:"blocksize"`
	RPM       string `json:"rpm,omitempty"`
}

// CreateISCSIExtent creates a new extent.
func (c *Client) CreateISCSIExtent(req ISCSIExtentCreate) (int64, error) {
	res, err := c.Call("iscsi.extent.create", []any{req})
	if err != nil {
		return 0, err
	}
	if id, ok := res.(float64); ok {
		return int64(id), nil
	}
	return 0, nil
}

// DeleteISCSIExtent removes an extent.
func (c *Client) DeleteISCSIExtent(id int64, remove bool) error {
	_, err := c.Call("iscsi.extent.delete", []any{id, remove})
	return err
}

// ISCSITarget is one row from iscsi.target.query.
type ISCSITarget map[string]any

// ListISCSITargets returns all iSCSI targets.
func (c *Client) ListISCSITargets() ([]ISCSITarget, error) {
	res, err := c.Query("iscsi.target.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]ISCSITarget, len(res))
	for i, r := range res {
		out[i] = ISCSITarget(r.(map[string]any))
	}
	return out, nil
}

// ISCSITargetCreate matches iscsi.target.create args.
type ISCSITargetCreate struct {
	Name        string   `json:"name"` // IQN
	Alias       string   `json:"alias,omitempty"`
	ModeIscsi   bool     `json:"mode_is"` // legacy field
	ModeFC      bool     `json:"mode_fc"`
	AuthNetwork []string `json:"auth_networks"`
}

// CreateISCSITarget creates a new target.
func (c *Client) CreateISCSITarget(req ISCSITargetCreate) (int64, error) {
	res, err := c.Call("iscsi.target.create", []any{req})
	if err != nil {
		return 0, err
	}
	if id, ok := res.(float64); ok {
		return int64(id), nil
	}
	return 0, nil
}

// DeleteISCSITarget removes a target.
func (c *Client) DeleteISCSITarget(id int64, force bool) error {
	opts := map[string]any{"force": force}
	_, err := c.Call("iscsi.target.delete", []any{id, opts})
	return err
}

// ISCSIAssociated is a target-to-extent link row.
type ISCSIAssociated map[string]any

// ListISCSIAssociated returns all target<->extent associations.
func (c *Client) ListISCSIAssociated() ([]ISCSIAssociated, error) {
	res, err := c.Query("iscsi.targetextent.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]ISCSIAssociated, len(res))
	for i, r := range res {
		out[i] = ISCSIAssociated(r.(map[string]any))
	}
	return out, nil
}

// ISCSIAssociateCreate matches iscsi.targetextent.create.
type ISCSIAssociateCreate struct {
	Target int64 `json:"target"`
	Extent int64 `json:"extent"`
	LUNID  int   `json:"lunid"`
}

// AssociateISCSI links a target to an extent.
func (c *Client) AssociateISCSI(req ISCSIAssociateCreate) (int64, error) {
	res, err := c.Call("iscsi.targetextent.create", []any{req})
	if err != nil {
		return 0, err
	}
	if id, ok := res.(float64); ok {
		return int64(id), nil
	}
	return 0, nil
}

// DissociateISCSI removes a target-to-extent link.
func (c *Client) DissociateISCSI(id int64) error {
	opts := map[string]any{"force": false}
	_, err := c.Call("iscsi.targetextent.delete", []any{id, opts})
	return err
}
