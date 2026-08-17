package truenas

// User is one row from user.query.
type User map[string]any

// ListUsers returns all users.
func (c *Client) ListUsers() ([]User, error) {
	res, err := c.Query("user.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]User, len(res))
	for i, r := range res {
		out[i] = User(r.(map[string]any))
	}
	return out, nil
}

// GetUser returns one user by ID (int) or username string.
func (c *Client) GetUser(idOrName any) (User, error) {
	res, err := c.Query("user.query", [][]any{{"id|username", "=", idOrName}})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, &ErrCall{Errname: "ENOENT", Reason: "user not found"}
	}
	return User(res[0].(map[string]any)), nil
}

// UserCreate matches user.create args.
type UserCreate struct {
	Username    string `json:"username"`
	FullName    string `json:"full_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Password    string `json:"password,omitempty"`
	UID         int    `json:"uid,omitempty"`
	Group       int    `json:"group,omitempty"`
	Home        string `json:"home,omitempty"`
	Shell       string `json:"shell,omitempty"`
	Locked      bool   `json:"locked"`
	SSHPassword bool   `json:"ssh_password_enabled"`
}

// CreateUser creates a new user.
func (c *Client) CreateUser(req UserCreate) (int64, error) {
	res, err := c.Call("user.create", []any{req})
	if err != nil {
		return 0, err
	}
	if id, ok := res.(float64); ok {
		return int64(id), nil
	}
	return 0, nil
}

// UpdateUser modifies an existing user.
func (c *Client) UpdateUser(id int64, req UserCreate) error {
	_, err := c.Call("user.update", []any{id, req})
	return err
}

// DeleteUser removes a user.
func (c *Client) DeleteUser(id int64) error {
	_, err := c.Call("user.delete", []any{id})
	return err
}

// SetUserPassword changes a user's password.
func (c *Client) SetUserPassword(username, password string) error {
	_, err := c.Call("user.set_password", []any{username, password})
	return err
}

// -- Groups --

// Group is one row from group.query.
type Group map[string]any

// ListGroups returns all groups.
func (c *Client) ListGroups() ([]Group, error) {
	res, err := c.Query("group.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]Group, len(res))
	for i, r := range res {
		out[i] = Group(r.(map[string]any))
	}
	return out, nil
}

// GetGroup returns one group by ID or name.
func (c *Client) GetGroup(idOrName any) (Group, error) {
	res, err := c.Query("group.query", [][]any{{"id|name", "=", idOrName}})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, &ErrCall{Errname: "ENOENT", Reason: "group not found"}
	}
	return Group(res[0].(map[string]any)), nil
}

// GroupCreate matches group.create args.
type GroupCreate struct {
	Name string `json:"group"`
	GID  int    `json:"gid,omitempty"`
	Perm string `json:"permissions,omitempty"` // one of USER, FULL etc
	// sudo / smb / other flags handled via separate call
}

// CreateGroup creates a group.
func (c *Client) CreateGroup(req GroupCreate) (int64, error) {
	res, err := c.Call("group.create", []any{req})
	if err != nil {
		return 0, err
	}
	if id, ok := res.(float64); ok {
		return int64(id), nil
	}
	return 0, nil
}

// UpdateGroup modifies a group.
func (c *Client) UpdateGroup(id int64, req GroupCreate) error {
	_, err := c.Call("group.update", []any{id, req})
	return err
}

// DeleteGroup removes a group.
func (c *Client) DeleteGroup(id int64) error {
	_, err := c.Call("group.delete", []any{id})
	return err
}

// -- ACL (filesystem.acl) --

// ACLPermissions are NFSv4 or POSIX ACL entries.
//
// filesystem.getacl returns entries with keys like:
//
//	tag, id, type, perms (map), flags, inherit
type ACL map[string]any

// GetACL returns the ACL on a filesystem path.
func (c *Client) GetACL(path string, simplified bool) ([]map[string]any, error) {
	opts := map[string]any{
		"path":       path,
		"simplified": simplified,
	}
	res, err := c.Call("filesystem.getacl", []any{opts})
	if err != nil {
		return nil, err
	}
	switch r := res.(type) {
	case []any:
		out := make([]map[string]any, len(r))
		for i, e := range r {
			out[i] = e.(map[string]any)
		}
		return out, nil
	case map[string]any:
		// newer SCALE wraps the entries under "acl_entries" or similar
		if entries, ok := r["acl_entries"].([]any); ok {
			out := make([]map[string]any, len(entries))
			for i, e := range entries {
				out[i] = e.(map[string]any)
			}
			return out, nil
		}
		return []map[string]any{r}, nil
	}
	return nil, nil
}

// SetACLOptions are filesystem.setacl args.
type SetACLOptions struct {
	Path      string           `json:"path"`
	Entries   []map[string]any `json:"acl"`
	Recurse   bool             `json:"recursive"`
	StripACL  bool             `json:"strip"`
	Canonical bool             `json:"canonicalize"`
	Traversal string           `json:"traversal,omitempty"` // "default" | "current"
}

// SetACL replaces the ACL on a path.
func (c *Client) SetACL(opts SetACLOptions) error {
	_, err := c.Call("filesystem.setacl", []any{opts})
	return err
}

// FileStat mirrors filesystem.stat result.
func (c *Client) FileStat(path string) (map[string]any, error) {
	res, err := c.Call("filesystem.stat", []any{path})
	if err != nil {
		return nil, err
	}
	return res.(map[string]any), nil
}
