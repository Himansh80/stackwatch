package truenas

// NFSShare is one row from sharing.nfs.query.
type NFSShare map[string]any

// ListNFS returns all NFS shares.
func (c *Client) ListNFS() ([]NFSShare, error) {
	res, err := c.Query("sharing.nfs.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]NFSShare, len(res))
	for i, r := range res {
		out[i] = NFSShare(r.(map[string]any))
	}
	return out, nil
}

// NFSShareCreate matches sharing.nfs.create args.
type NFSShareCreate struct {
	Path     string   `json:"path"`
	Comment  string   `json:"comment,omitempty"`
	Enabled  bool     `json:"enabled"`
	ReadOnly bool     `json:"ro"`
	Networks []string `json:"networks"`
	Hosts    []string `json:"hosts"`
	// Maproot user/group controlled via maproot/maproot_user elsewhere
}

// CreateNFS adds an NFS share.
func (c *Client) CreateNFS(req NFSShareCreate) (int64, error) {
	res, err := c.Call("sharing.nfs.create", []any{req})
	if err != nil {
		return 0, err
	}
	if id, ok := res.(float64); ok {
		return int64(id), nil
	}
	return 0, nil
}

// UpdateNFS modifies an existing NFS share.
func (c *Client) UpdateNFS(id int64, req NFSShareCreate) error {
	_, err := c.Call("sharing.nfs.update", []any{id, req})
	return err
}

// DeleteNFS removes an NFS share by ID.
func (c *Client) DeleteNFS(id int64) error {
	_, err := c.Call("sharing.nfs.delete", []any{id})
	return err
}

// SMBShare is one row from sharing.smb.query.
type SMBShare map[string]any

// ListSMB returns all SMB shares.
func (c *Client) ListSMB() ([]SMBShare, error) {
	res, err := c.Query("sharing.smb.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]SMBShare, len(res))
	for i, r := range res {
		out[i] = SMBShare(r.(map[string]any))
	}
	return out, nil
}

// SMBShareCreate matches sharing.smb.create args.
type SMBShareCreate struct {
	Name       string   `json:"name"`
	Path       string   `json:"path"`
	Comment    string   `json:"comment"`
	Enabled    bool     `json:"enabled"`
	ReadOnly   bool     `json:"ro"`
	Browseable bool     `json:"browseable"`
	GuestOk    bool     `json:"guestok"`
	HostsAllow []string `json:"hostsallow"`
	HostsDeny  []string `json:"hostsdeny"`
}

// CreateSMB adds an SMB share.
func (c *Client) CreateSMB(req SMBShareCreate) (int64, error) {
	res, err := c.Call("sharing.smb.create", []any{req})
	if err != nil {
		return 0, err
	}
	if id, ok := res.(float64); ok {
		return int64(id), nil
	}
	return 0, nil
}

// UpdateSMB modifies an existing SMB share.
func (c *Client) UpdateSMB(id int64, req SMBShareCreate) error {
	_, err := c.Call("sharing.smb.update", []any{id, req})
	return err
}

// DeleteSMB removes an SMB share by ID.
func (c *Client) DeleteSMB(id int64) error {
	_, err := c.Call("sharing.smb.delete", []any{id})
	return err
}
