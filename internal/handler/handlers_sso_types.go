// Tier 9 Phase 1 — SSO Foundation (Tier 9.1).
// Shared types + allowed-provider maps for the 8 SSO endpoints.
//
// Eight endpoints back the SSO surface:
//
//	GET    /api/v1/enterprise/sso/providers        — list providers (caller's tenant)
//	POST   /api/v1/enterprise/sso/providers        — create provider (encrypts config)
//	PATCH  /api/v1/enterprise/sso/providers/:id    — update name / config
//	DELETE /api/v1/enterprise/sso/providers/:id    — soft delete (enabled=false)
//	POST   /api/v1/enterprise/sso/test             — validate IdP discovery/metadata
//	GET    /api/v1/enterprise/sso/connections      — caller's SSO connections
//
// Plus 3 PUBLIC callback routes registered on the public router
// (no JWT — IdP-issued code/assertion is the auth):
//
//	GET    /api/v1/enterprise/sso/initiate         — start OIDC/SAML flow
//	GET    /api/v1/enterprise/sso/callback         — OIDC code→token exchange
//	POST   /api/v1/enterprise/sso/callback         — SAML POST binding
//
// Every protected query honors tenant_id from the JWT — no
// cross-tenant data ever crosses the wire. SSO provider configs are
// encrypted at the application layer (AES-GCM via the existing
// internal/handler.encryptSecret helper from credentials.go) BEFORE
// they're inserted — the JSONB `config` column holds opaque ciphertext
// blobs (client_secret_ciphertext/_nonce pairs, SAML x509 cert
// ciphertext/nonce pairs). The `config_summary` field returned in
// list responses contains only NON-SECRET fields (client_id,
// discovery_url, entity_id) so admins can recognize a provider
// without us leaking secrets.
package handler

// Compile-time guard so unused types referenced only via JSON shape
// don't accidentally get pruned by linters / go mod tidy.
var _ = struct{}{}

// ------------------------------------------------------------------
// JSON row types — mirror the column shape and map directly to the
// `out` slice returned by each List* handler.
// ------------------------------------------------------------------

// ssoProviderRow is the JSON shape for a single SSO provider returned
// by GET /sso/providers + POST /sso/providers + PATCH /sso/providers/:id.
//
// `config_summary` is a non-secret preview of the `config` jsonb blob
// (OIDC: {client_id, discovery_url, scopes}; SAML: {entity_id, sso_url}).
// The actual `config` jsonb is opaque ciphertext and is NOT returned
// in API responses — the only way to see it is to decrypt locally
// (which only the server-side handler does for outbound calls).
type ssoProviderRow struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	Type           string                 `json:"type"` // "oidc" | "saml"
	Name           string                 `json:"name"`
	Enabled        bool                   `json:"enabled"`
	ConfigSummary  map[string]interface{} `json:"config_summary"`
	CreatedAt      string                 `json:"created_at"`
}

// ssoConnectionRow is the JSON shape for a single SSO connection
// returned by GET /sso/connections. `subject` is the IdP's stable
// user identifier (OIDC `sub` claim or SAML NameID).
type ssoConnectionRow struct {
	ID          string  `json:"id"`
	TenantID    string  `json:"tenant_id"`
	UserID      string  `json:"user_id"`
	ProviderID  string  `json:"provider_id"`
	ProviderName string `json:"provider_name,omitempty"`
	Subject     string  `json:"subject"`
	CreatedAt   string  `json:"created_at"`
	LastUsedAt  *string `json:"last_used_at,omitempty"`
}

// ------------------------------------------------------------------
// Request types — body shape for POST/PATCH endpoints.
// ------------------------------------------------------------------

// ssoProviderReq is the JSON body for POST /sso/providers (create).
// `type` and `name` are required. `config` is required and must
// match the shape for `type`:
//
//	oidc: {client_id, client_secret, discovery_url, scopes?: [..]}
//	saml: {metadata_xml?} OR {metadata_url?}, entity_id, sso_url,
//	      x509_cert (PEM string)
//
// Sensitive fields are encrypted by the handler before INSERT.
type ssoProviderReq struct {
	Type   string                 `json:"type"   binding:"required,oneof=oidc saml"`
	Name   string                 `json:"name"   binding:"required,min=1,max=64"`
	Config map[string]interface{} `json:"config" binding:"required"`
}

// ssoProviderPatchReq is the JSON body for PATCH /sso/providers/:id.
// All fields optional — only the provided ones are updated. Re-encrypts
// `config` if provided.
type ssoProviderPatchReq struct {
	Name    *string                `json:"name"    binding:"omitempty,min=1,max=64"`
	Enabled *bool                  `json:"enabled" binding:"omitempty"`
	Config  map[string]interface{} `json:"config"  binding:"omitempty"`
}

// ssoTestReq is the JSON body for POST /sso/test. Tests the IdP
// connectivity: for OIDC, fetches discovery_url; for SAML, parses
// metadata_xml/metadata_url. Returns {ok, details}.
type ssoTestReq struct {
	ProviderID string `json:"provider_id" binding:"required,uuid"`
}

// ------------------------------------------------------------------
// jsonb config shapes — type aliases for the parsed config blobs.
// We unmarshal config jsonb into these shapes at the boundary so the
// rest of the handler works with typed values rather than interface{}.
// ------------------------------------------------------------------

// oidcConfig is the parsed shape of an OIDC provider's config jsonb.
// client_secret_ciphertext + client_secret_nonce are AES-GCM blobs
// produced by encryptSecret in credentials.go.
type oidcConfig struct {
	ClientID               string   `json:"client_id"`
	ClientSecretCiphertext []byte   `json:"client_secret_ciphertext,omitempty"`
	ClientSecretNonce      []byte   `json:"client_secret_nonce,omitempty"`
	DiscoveryURL           string   `json:"discovery_url"`
	RedirectURI            string   `json:"redirect_uri,omitempty"`
	Scopes                 []string `json:"scopes,omitempty"`
	KeyID                  string   `json:"key_id"`
}

// samlConfig is the parsed shape of a SAML provider's config jsonb.
// metadata_xml is the raw XML (potentially huge), x509_cert_ciphertext
// is the encrypted PEM body used to verify inbound signatures.
type samlConfig struct {
	MetadataXML          string `json:"metadata_xml,omitempty"`
	MetadataURL          string `json:"metadata_url,omitempty"`
	EntityID             string `json:"entity_id"`
	SSOURL               string `json:"sso_url"`
	X509CertCiphertext   []byte `json:"x509_cert_ciphertext,omitempty"`
	X509CertNonce        []byte `json:"x509_cert_nonce,omitempty"`
	KeyID                string `json:"key_id"`
}

// oidcDiscovery is the OIDC Discovery 1.0 document (subset).
// Source: https://openid.net/specs/openid-connect-discovery-1_0.html
type oidcDiscovery struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserinfoEndpoint      string   `json:"userinfo_endpoint"`
	JwksURI               string   `json:"jwks_uri"`
	ScopesSupported       []string `json:"scopes_supported,omitempty"`
}

// oidcTokenResponse is the OIDC token_endpoint response (subset).
type oidcTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token,omitempty"`
}

// oidcUserinfo is the OIDC userinfo_endpoint response (subset).
type oidcUserinfo struct {
	Sub     string `json:"sub"`
	Email   string `json:"email,omitempty"`
	Name    string `json:"name,omitempty"`
	Given   string `json:"given_name,omitempty"`
	Family  string `json:"family_name,omitempty"`
}

// ------------------------------------------------------------------
// Access-control maps — whitelist valid enum values.
// ------------------------------------------------------------------

// allowedSSOProviderTypes is the whitelist for the `type` enum on
// SSO providers. The same set is enforced at the binding level
// (via `oneof`) AND at the SQL level (via sso_providers.type CHECK)
// so a hand-crafted request can't smuggle in junk.
var allowedSSOProviderTypes = map[string]bool{
	"oidc": true,
	"saml": true,
}
