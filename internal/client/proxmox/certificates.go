package proxmox

import (
	"context"
	"fmt"
	"net/url"
)

// Certificate describes an installed certificate on a node or cluster.
// Proxmox /nodes/{node}/certificates returns just slot names ({name: "acme"|"custom"|"info"}).
// The /nodes/{node}/certificates/{slot} endpoint returns the full cert.
type Certificate struct {
	Name          string   `json:"name"`                    // e.g. "acme", "custom", "info"
	Filename      string   `json:"filename,omitempty"`      // alias (some endpoints use this)
	Subject       string   `json:"subject,omitempty"`       // certificate subject
	Issuer        string   `json:"issuer,omitempty"`
	NotBefore     int64    `json:"notbefore,omitempty"`     // unix epoch
	NotAfter      int64    `json:"notafter,omitempty"`      // unix epoch
	PublicKeyBits int      `json:"public-key-bits,omitempty"`
	PublicKeyType string   `json:"public-key-type,omitempty"`
	San           []string `json:"san,omitempty"`           // subject alt names
	Fingerprint   string   `json:"fingerprint,omitempty"`
}

// ListCertificates returns available certificate slots on a node.
// Returns minimal info (just names) — actual content via the URLs.
func (c *Client) ListCertificates(ctx context.Context, node string) ([]Certificate, error) {
	path := "/nodes/" + url.PathEscape(node) + "/certificates"
	var out []Certificate
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ACMEAccount represents a registered Let's Encrypt / other CA account.
type ACMEAccount struct {
	Name           string `json:"name"`                  // account name
	Email          string `json:"email,omitempty"`        // contact email
	Directory      string `json:"directory,omitempty"`     // CA URL (Let's Encrypt staging, prod, etc.)
	TOS            string `json:"tos,omitempty"`           // terms of service URL
	RegisteredAt   int64  `json:"registered-at,omitempty"` // unix epoch
	LastUpdate      int64  `json:"last-update,omitempty"`
	Status         string `json:"status,omitempty"` // "active", "expired", etc.
}

// ListACMEAccounts returns all ACME accounts.
func (c *Client) ListACMEAccounts(ctx context.Context) ([]ACMEAccount, error) {
	var out []ACMEAccount
	if err := c.get(ctx, "/cluster/acme/account", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetACMEAccount returns one ACME account.
func (c *Client) GetACMEAccount(ctx context.Context, name string) (ACMEAccount, error) {
	var out ACMEAccount
	path := "/cluster/acme/account/" + url.PathEscape(name)
	if err := c.get(ctx, path, &out); err != nil {
		return out, err
	}
	return out, nil
}

// CreateACMEAccount registers a new ACME account.
// Required: name, email, directory (CA URL).
// tos_url required on first creation.
func (c *Client) CreateACMEAccount(ctx context.Context, name, email, directory, tosURL string) (string, error) {
	form := url.Values{}
	form.Set("name", name)
	form.Set("email", email)
	form.Set("directory", directory)
	if tosURL != "" {
		form.Set("tos_url", tosURL)
	}
	if err := c.postForm(ctx, "/cluster/acme/account", form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// UpdateACMEAccount modifies account settings.
func (c *Client) UpdateACMEAccount(ctx context.Context, name string, fields url.Values) (string, error) {
	path := "/cluster/acme/account/" + url.PathEscape(name)
	if err := c.putForm(ctx, path, fields, nil); err != nil {
		return "", err
	}
	return "", nil
}

// DeleteACMEAccount removes an ACME account.
func (c *Client) DeleteACMEAccount(ctx context.Context, name string) (string, error) {
	if err := c.deleteForm(ctx, "/cluster/acme/account/"+url.PathEscape(name)); err != nil {
		return "", err
	}
	return "", nil
}

// ACMEPlugin is a DNS validator (Cloudflare, Route53, etc.) or standalone.
type ACMEPlugin struct {
	Plugin string `json:"plugin"`           // unique plugin name
	Type   string `json:"type"`             // "standalone", "dns", "rfc2136"
	Digest string `json:"digest,omitempty"` // checksum of plugin data
}

// ListACMEPlugins returns all installed ACME plugins.
func (c *Client) ListACMEPlugins(ctx context.Context) ([]ACMEPlugin, error) {
	var out []ACMEPlugin
	if err := c.get(ctx, "/cluster/acme/plugins", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateACMEPlugin adds a new ACME plugin (standalone or DNS validator).
// For standalone: data is empty.
// For DNS plugins: data is "key1=val1\nkey2=val2".
func (c *Client) CreateACMEPlugin(ctx context.Context, name, pluginType, data string) (string, error) {
	form := url.Values{}
	form.Set("type", pluginType)
	if data != "" {
		form.Set("data", data)
	}
	// Proxmox uses PUT to create/replace by name.
	path := "/cluster/acme/plugins/" + url.PathEscape(name)
	if err := c.putForm(ctx, path, form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// DeleteACMEPlugin removes an ACME plugin.
func (c *Client) DeleteACMEPlugin(ctx context.Context, name string) (string, error) {
	if err := c.deleteForm(ctx, "/cluster/acme/plugins/"+url.PathEscape(name)); err != nil {
		return "", err
	}
	return "", nil
}

// ACMEChallengeSchemaEntry describes one supported ACME challenge type
// and the data fields needed for its DNS validator plugin.
type ACMEChallengeSchemaEntry struct {
	ID     string                 `json:"id"`     // "1984hosting", "acmedns", etc.
	Type   string                 `json:"type"`   // "dns", "standalone"
	Name   string                 `json:"name"`   // display name
	Schema map[string]interface{} `json:"schema,omitempty"`
}

// ListACMEChallengeSchema returns supported challenge plugins.
func (c *Client) ListACMEChallengeSchema(ctx context.Context) ([]ACMEChallengeSchemaEntry, error) {
	var out []ACMEChallengeSchemaEntry
	if err := c.get(ctx, "/cluster/acme/challenge-schema", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ACMEDirectory describes a supported ACME CA (Let's Encrypt, ZeroSSL, etc.).
type ACMEDirectory struct {
	Name      string `json:"name"`       // "letsencrypt", "letsencrypt-staging"
	URL       string `json:"url"`        // ACME directory URL
	TOS       string `json:"tos,omitempty"`
	Website   string `json:"website,omitempty"`
}

// ListACMEDirectories returns supported ACME directories.
func (c *Client) ListACMEDirectories(ctx context.Context) ([]ACMEDirectory, error) {
	var out []ACMEDirectory
	if err := c.get(ctx, "/cluster/acme/directories", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ACMEMetaInfo describes ACME directory metadata.
type ACMEMetaInfo struct {
	Name      string `json:"name,omitempty"`
	Website   string `json:"website,omitempty"`
	CaaIdentities string `json:"caa-identities,omitempty"`
	ExternalAccountRequired bool `json:"externalAccountRequired,omitempty"`
	TermsOfService string `json:"termsOfService,omitempty"`
}

// GetACMETAOfService returns the current ToS URL for a directory.
// dir is the URL-encoded ACME directory endpoint.
func (c *Client) GetACMETAOfService(ctx context.Context, dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("directory required")
	}
	path := "/cluster/acme/tos?directory=" + url.QueryEscape(dir)
	var out ACMEMetaInfo
	if err := c.get(ctx, path, &out); err != nil {
		return "", err
	}
	return out.TermsOfService, nil
}

// ACMEInfo describes the current ACME configuration.
type ACMEInfo struct {
	Enabled int    `json:"enabled,omitempty"`
	OldCA   string `json:"old-ca,omitempty"`
	NewCA   string `json:"new-ca,omitempty"`
	URL     string `json:"url,omitempty"`
}

// GetACMEInfo returns current ACME config status.
func (c *Client) GetACMEInfo(ctx context.Context) (ACMEInfo, error) {
	var out ACMEInfo
	if err := c.get(ctx, "/cluster/acme/info", &out); err != nil {
		return out, err
	}
	return out, nil
}