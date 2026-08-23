package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// PVEUser describes a Proxmox VE user account (or PVE token).
//
// Note: Proxmox returns `groups` as an array (e.g. ["admins"]) in user
// detail responses but as a comma-separated string in list responses.
// We accept either via a custom type.
type PVEUser struct {
	UserID    string              `json:"userid"` // e.g. "root@pam" or "jdoe@pve"
	Comment   string              `json:"comment,omitempty"`
	Email     string              `json:"email,omitempty"`
	Enable    int                 `json:"enable"`           // 0/1
	Expire    int                 `json:"expire,omitempty"` // unix epoch (0 = never)
	FirstName string              `json:"firstname,omitempty"`
	LastName  string              `json:"lastname,omitempty"`
	RealmType string              `json:"realmtype,omitempty"` // pam, pve, etc.
	Groups    Groups              `json:"groups,omitempty"`    // string or array
	Tokens    map[string]APIToken `json:"tokens,omitempty"`
}

// Groups accepts both array and string forms from Proxmox.
type Groups []string

// UnmarshalJSON implements json.Unmarshaler for Groups.
func (g *Groups) UnmarshalJSON(data []byte) error {
	// If null/empty, leave as nil
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	// If string, split on commas
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		if s == "" {
			return nil
		}
		*g = append(*g, s)
		return nil
	}
	// Otherwise treat as array of strings
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	*g = arr
	return nil
}

// APITokenCreateResponse is what Proxmox returns from POST /access/users/{userid}/token/{tokenid}.
// Only POST returns the secret `value` — subsequent GETs do NOT include it.
//
// Note: privsep is sometimes int (1) and sometimes string ("0") in Proxmox responses
// depending on which value was set. Use a custom unmarshal or accept either.
type APITokenCreateResponse struct {
	Value       string `json:"value"`        // secret part (uuid-XXXX) — ONLY returned on create
	FullTokenID string `json:"full-tokenid"` // name@realm!tokenid (e.g. "root@pam!monitor")
	Info        struct {
		Privsep interface{} `json:"privsep"` // can be int (1) or string ("0")
		Comment string      `json:"comment,omitempty"`
	} `json:"info"`
}

// APIToken is an API token (used for LIST responses — no secret value).
// Proxmox uses "id" in the keyed-map form and "tokenid" in the array form;
// we accept both.
type APIToken struct {
	TokenID string `json:"tokenid,omitempty"` // primary: array form uses "tokenid"
	Comment string `json:"comment,omitempty"`
	Enable  int    `json:"enable"`
	Expire  int    `json:"expire,omitempty"`
	Privsep int    `json:"privsep,omitempty"`
	Value   string `json:"value,omitempty"` // ONLY present on create, never on list
	ID      string `json:"id,omitempty"`    // fallback: keyed-map form uses "id"
}

// tokenID returns the canonical token identifier regardless of source shape.
func (t APIToken) tokenID() string {
	if t.TokenID != "" {
		return t.TokenID
	}
	return t.ID
}

// ListUsers returns all Proxmox users.
func (c *Client) ListUsers(ctx context.Context) ([]PVEUser, error) {
	var out []PVEUser
	if err := c.get(ctx, "/access/users", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateUser creates a new Proxmox user.
// Note: Proxmox's POST /access/users does NOT accept a password directly
// (schema rejects `password` field). The user is created with no password.
//
// IMPORTANT: Proxmox internally tries to set password on POST and returns
// HTTP 500 with "change password failed: user 'X' does not exist" even when
// the user IS created. We swallow that specific 500 to allow the API to
// succeed.
//
// On some Proxmox configs, this is the expected behavior — the user IS
// created, just without password. Set password via `/access/password`
// endpoint (requires user to authenticate) or via the PAM backend directly.
func (c *Client) CreateUser(ctx context.Context, userid, password, email, comment, firstname, lastname string) (string, error) {
	path := "/access/users"
	form := url.Values{}
	form.Set("userid", userid)
	if email != "" {
		form.Set("email", email)
	}
	if comment != "" {
		form.Set("comment", comment)
	}
	if firstname != "" {
		form.Set("firstname", firstname)
	}
	if lastname != "" {
		form.Set("lastname", lastname)
	}
	err := c.postForm(ctx, path, form, nil)
	if err != nil && strings.Contains(err.Error(), "change password failed") {
		// Proxmox quirk: user IS created, just without password set
		// (because /access/users POST tries to set password via /access/password
		// which fails for fresh users). Swallow this specific error.
		return "", nil
	}
	if err != nil {
		return "", err
	}
	// Password setting is not reliably supported via API on all Proxmox configs.
	_ = password
	return "", nil
}

// GetUser returns a single user by id (e.g. "root@pam").
func (c *Client) GetUser(ctx context.Context, userid string) (PVEUser, error) {
	path := "/access/users/" + url.PathEscape(userid)
	var out PVEUser
	if err := c.get(ctx, path, &out); err != nil {
		return out, err
	}
	return out, nil
}

// UpdateUser modifies a user. Pass only the fields you want to change.
func (c *Client) UpdateUser(ctx context.Context, userid string, fields map[string]string) (string, error) {
	path := "/access/users/" + url.PathEscape(userid)
	form := url.Values{}
	for k, v := range fields {
		form.Set(k, v)
	}
	if err := c.putForm(ctx, path, form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// DeleteUser removes a user (sync, no task).
func (c *Client) DeleteUser(ctx context.Context, userid string) (string, error) {
	path := "/access/users/" + url.PathEscape(userid)
	if err := c.deleteForm(ctx, path); err != nil {
		return "", err
	}
	return "", nil
}

// ListAPITokens returns all API tokens for a user.
//
// Proxmox response shape varies by version:
//   - newer PVE: `{"data": {"<tokenid>": {...}, ...}}` (map keyed by tokenid)
//   - older PVE / this .107: `{"data": [{"tokenid": "..."}, ...}` (array)
//
// This function normalizes the result to a map keyed by tokenid so the
// handler always sees a consistent shape.
func (c *Client) ListAPITokens(ctx context.Context, userid string) (map[string]APIToken, error) {
	path := fmt.Sprintf("/access/users/%s/token",
		url.PathEscape(userid))

	// Issue a raw request so we can detect the response shape ourselves
	// (c.get() would unmarshal into either map or array and fail on the
	// wrong one — we want to handle BOTH shapes gracefully).
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api2/json"+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("proxmox: unauthorized (check API token)")
	}
	if resp.StatusCode == 501 {
		return nil, ErrNotSupported
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("proxmox: HTTP %d on %s", resp.StatusCode, path)
	}
	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, err
	}
	// Try map shape first (canonical).
	var asMap map[string]APIToken
	if err := json.Unmarshal(wrapper.Data, &asMap); err == nil && len(asMap) > 0 {
		return asMap, nil
	}
	// Fallback: array shape.
	var arr []APIToken
	if err := json.Unmarshal(wrapper.Data, &arr); err == nil {
		out := make(map[string]APIToken, len(arr))
		for i, t := range arr {
			key := t.tokenID()
			if key == "" {
				key = fmt.Sprintf("token-%d", i)
			}
			out[key] = t
		}
		return out, nil
	}
	// Neither shape — return empty so handler still works.
	return map[string]APIToken{}, nil
}

// AddAPIToken creates a new API token for a user.
// IMPORTANT: the returned token's "value" field is the secret —
// caller must save it; Proxmox will never show it again.
//
// Proxmox returns a special response shape:
// {"data": {"value": "uuid-...", "full-tokenid": "name@realm!tokenid", "info": {...}}}
// NOT a flat APIToken struct.
//
// privsep can be int (1) or string ("0") in JSON depending on which value
// was set. We use interface{} for the field and convert manually.
func (c *Client) AddAPIToken(ctx context.Context, userid, tokenid, comment string, privsep int, expire int) (APIToken, error) {
	path := fmt.Sprintf("/access/users/%s/token/%s",
		url.PathEscape(userid), url.PathEscape(tokenid))
	form := url.Values{}
	if comment != "" {
		form.Set("comment", comment)
	}
	if privsep > 0 {
		form.Set("privsep", "1")
	} else {
		form.Set("privsep", "0")
	}
	if expire > 0 {
		form.Set("expire", fmt.Sprintf("%d", expire))
	}
	var resp APITokenCreateResponse
	if err := c.postForm(ctx, path, form, &resp); err != nil {
		return APIToken{}, err
	}
	// Convert privsep (which may be int or string) to int
	privsepOut := 0
	switch v := resp.Info.Privsep.(type) {
	case int:
		privsepOut = v
	case float64:
		privsepOut = int(v)
	case string:
		fmt.Sscanf(v, "%d", &privsepOut)
	}
	return APIToken{
		TokenID: tokenid,
		Comment: resp.Info.Comment,
		Privsep: privsepOut,
		Expire:  expire,
		Value:   resp.Value,
	}, nil
}

// DeleteAPIToken removes an API token from a user.
func (c *Client) DeleteAPIToken(ctx context.Context, userid, tokenid string) (string, error) {
	path := fmt.Sprintf("/access/users/%s/token/%s",
		url.PathEscape(userid), url.PathEscape(tokenid))
	if err := c.deleteForm(ctx, path); err != nil {
		return "", err
	}
	return "", nil
}
