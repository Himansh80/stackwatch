// Tier 9 Phase 1 — SSO Foundation (Tier 9.1).
// Helpers for SSO config encryption, OIDC discovery/token exchange,
// SAML AuthnRequest builder, and IdP connectivity test.
//
// Split from handlers_sso.go (which holds the 6 protected handlers)
// and handlers_sso_callbacks.go (which holds the 3 public handlers)
// so each file stays under the 400-LOC cap.
//
// All crypto uses the existing internal/handler.encryptSecret /
// decryptSecret helpers (AES-GCM via CREDENTIALS_MASTER_KEY env).
// All HTTP uses net/http (no new deps).
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ----- config encryption -----

// encryptSSOConfig encrypts sensitive fields in a request body and
// returns a jsonb-shaped blob ready for INSERT.
//
// OIDC: encrypts `client_secret` → `client_secret_ciphertext/_nonce`
// SAML: encrypts `x509_cert` → `x509_cert_ciphertext/_nonce`
//
// Both: stamps `key_id` from CredentialsKeyId so a future key
// rotation can pick the right decrypt path.
func encryptSSOConfig(typ string, in map[string]interface{}) ([]byte, error) {
	switch typ {
	case "oidc":
		secret, _ := in["client_secret"].(string)
		if secret == "" {
			return nil, errors.New("oidc config missing client_secret")
		}
		ct, nonce, err := encryptSecret(secret)
		if err != nil {
			return nil, fmt.Errorf("encrypt client_secret: %w", err)
		}
		delete(in, "client_secret")
		in["client_secret_ciphertext"] = ct
		in["client_secret_nonce"] = nonce
		in["key_id"] = CredentialsKeyId
		return json.Marshal(in)

	case "saml":
		cert, _ := in["x509_cert"].(string)
		if cert == "" {
			return nil, errors.New("saml config missing x509_cert")
		}
		ct, nonce, err := encryptSecret(cert)
		if err != nil {
			return nil, fmt.Errorf("encrypt x509_cert: %w", err)
		}
		delete(in, "x509_cert")
		in["x509_cert_ciphertext"] = ct
		in["x509_cert_nonce"] = nonce
		in["key_id"] = CredentialsKeyId
		return json.Marshal(in)

	default:
		return nil, fmt.Errorf("unsupported provider type %q", typ)
	}
}

// summarizeSSOConfig extracts NON-SECRET fields from a config blob
// (whether plaintext for newly-created providers OR ciphertext for
// existing ones — we only look at fields that are never secret).
//
// For OIDC: client_id, discovery_url, scopes, redirect_uri
// For SAML: entity_id, sso_url, metadata_xml length, metadata_url
func summarizeSSOConfig(typ string, rawConfig []byte) map[string]interface{} {
	out := map[string]interface{}{}
	if len(rawConfig) == 0 {
		return out
	}
	var m map[string]interface{}
	if err := json.Unmarshal(rawConfig, &m); err != nil {
		return out
	}
	switch typ {
	case "oidc":
		if v, ok := m["client_id"].(string); ok {
			out["client_id"] = v
		}
		if v, ok := m["discovery_url"].(string); ok {
			out["discovery_url"] = v
		}
		if v, ok := m["redirect_uri"].(string); ok && v != "" {
			out["redirect_uri"] = v
		}
		if v, ok := m["scopes"].([]interface{}); ok {
			out["scopes"] = v
		}
	case "saml":
		if v, ok := m["entity_id"].(string); ok {
			out["entity_id"] = v
		}
		if v, ok := m["sso_url"].(string); ok {
			out["sso_url"] = v
		}
		if v, ok := m["metadata_url"].(string); ok && v != "" {
			out["metadata_url"] = v
		}
		if v, ok := m["metadata_xml"].(string); ok && v != "" {
			out["metadata_xml_length"] = len(v)
		}
	}
	return out
}

// ----- IdP connectivity test -----

// testSSOConfigConnectivity validates an IdP's reachability without
// going through a full auth flow.
func testSSOConfigConnectivity(typ string, rawConfig []byte) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	switch typ {
	case "oidc":
		var cfg oidcConfig
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return out, fmt.Errorf("decode config: %w", err)
		}
		if cfg.DiscoveryURL == "" {
			return out, errors.New("discovery_url missing")
		}
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(cfg.DiscoveryURL)
		if err != nil {
			return out, fmt.Errorf("discovery fetch: %w", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return out, fmt.Errorf("discovery http %d", resp.StatusCode)
		}
		var disc oidcDiscovery
		if err := json.Unmarshal(body, &disc); err != nil {
			return out, fmt.Errorf("decode discovery: %w", err)
		}
		missing := []string{}
		for _, e := range []struct{ k, v string }{
			{"issuer", disc.Issuer},
			{"authorization_endpoint", disc.AuthorizationEndpoint},
			{"token_endpoint", disc.TokenEndpoint},
			{"userinfo_endpoint", disc.UserinfoEndpoint},
			{"jwks_uri", disc.JwksURI},
		} {
			if e.v == "" {
				missing = append(missing, e.k)
			}
		}
		if len(missing) > 0 {
			return out, fmt.Errorf("discovery missing endpoints: %v", missing)
		}
		out["issuer"] = disc.Issuer
		out["authorization_endpoint"] = disc.AuthorizationEndpoint
		out["token_endpoint"] = disc.TokenEndpoint
		out["userinfo_endpoint"] = disc.UserinfoEndpoint
		out["jwks_uri"] = disc.JwksURI
		return out, nil

	case "saml":
		var cfg samlConfig
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return out, fmt.Errorf("decode config: %w", err)
		}
		if cfg.EntityID == "" {
			return out, errors.New("entity_id missing")
		}
		if cfg.SSOURL == "" {
			return out, errors.New("sso_url missing")
		}
		if len(cfg.X509CertCiphertext) > 0 {
			pem, err := decryptSecret(cfg.X509CertCiphertext, cfg.X509CertNonce)
			if err != nil {
				return out, fmt.Errorf("decrypt cert: %w", err)
			}
			out["x509_cert_length"] = len(pem)
		}
		out["entity_id"] = cfg.EntityID
		out["sso_url"] = cfg.SSOURL
		return out, nil

	default:
		return out, fmt.Errorf("unsupported type %q", typ)
	}
}

// ----- OIDC: discovery + token + userinfo -----

// fetchOIDCDiscovery performs a single GET on the discovery URL and
// returns the parsed document. Reused by InitiateSSO, OIDCCallback,
// and TestSSOProvider so we don't repeat the fetch logic.
func fetchOIDCDiscovery(ctx context.Context, discoveryURL string) (*oidcDiscovery, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", discoveryURL, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch discovery: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery http %d: %s", resp.StatusCode, string(body))
	}
	var disc oidcDiscovery
	if err := json.Unmarshal(body, &disc); err != nil {
		return nil, fmt.Errorf("decode discovery: %w", err)
	}
	return &disc, nil
}

// buildOIDCAuthorizeURL composes the IdP authorization URL with the
// OIDC standard params + signed state.
func buildOIDCAuthorizeURL(rawConfig []byte, state string) (string, error) {
	var cfg oidcConfig
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return "", fmt.Errorf("decode config: %w", err)
	}
	if cfg.DiscoveryURL == "" {
		return "", errors.New("discovery_url missing")
	}
	disc, err := fetchOIDCDiscovery(context.Background(), cfg.DiscoveryURL)
	if err != nil {
		return "", err
	}
	if disc.AuthorizationEndpoint == "" {
		return "", errors.New("authorization_endpoint missing in discovery")
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}
	u, err := url.Parse(disc.AuthorizationEndpoint)
	if err != nil {
		return "", fmt.Errorf("parse authorize URL: %w", err)
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", cfg.RedirectURI)
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// exchangeOIDCCode performs the authorization-code → token exchange
// at the IdP's token_endpoint.
func exchangeOIDCCode(ctx context.Context, rawConfig []byte, code string) (*oidcTokenResponse, error) {
	var cfg oidcConfig
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	secret, err := decryptSecret(cfg.ClientSecretCiphertext, cfg.ClientSecretNonce)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: %w", err)
	}
	disc, err := fetchOIDCDiscovery(ctx, cfg.DiscoveryURL)
	if err != nil {
		return nil, err
	}
	if disc.TokenEndpoint == "" {
		return nil, errors.New("token_endpoint missing in discovery")
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", cfg.RedirectURI)
	req, _ := http.NewRequestWithContext(ctx, "POST", disc.TokenEndpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(cfg.ClientID, secret)
	client := &http.Client{Timeout: 10 * time.Second}
	tokResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token POST: %w", err)
	}
	defer tokResp.Body.Close()
	tokBody, _ := io.ReadAll(tokResp.Body)
	if tokResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token http %d: %s", tokResp.StatusCode, string(tokBody))
	}
	var tok oidcTokenResponse
	if err := json.Unmarshal(tokBody, &tok); err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, errors.New("token response missing access_token")
	}
	return &tok, nil
}

// fetchOIDCUserinfo calls the userinfo_endpoint with the bearer
// access token and returns the parsed JSON.
func fetchOIDCUserinfo(ctx context.Context, rawConfig []byte, accessToken string) (*oidcUserinfo, error) {
	var cfg oidcConfig
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	disc, err := fetchOIDCDiscovery(ctx, cfg.DiscoveryURL)
	if err != nil {
		return nil, err
	}
	if disc.UserinfoEndpoint == "" {
		return nil, errors.New("userinfo_endpoint missing in discovery")
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", disc.UserinfoEndpoint, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	uiResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo GET: %w", err)
	}
	defer uiResp.Body.Close()
	uiBody, _ := io.ReadAll(uiResp.Body)
	if uiResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo http %d: %s", uiResp.StatusCode, string(uiBody))
	}
	var ui oidcUserinfo
	if err := json.Unmarshal(uiBody, &ui); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}
	return &ui, nil
}

// ----- SAML AuthnRequest URL builder -----

// buildSAMLAuthnRequestURL composes an HTTP-Redirect SAML AuthnRequest
// URL. We sign state as the RelayState so the SAML POST binding can
// recover the provider_id. The actual AuthnRequest XML body is
// deferred to the IdP landing — Phase 1 only handles the redirect
// flow with state + entity_id.
func buildSAMLAuthnRequestURL(rawConfig []byte, state string) (string, error) {
	var cfg samlConfig
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return "", fmt.Errorf("decode config: %w", err)
	}
	if cfg.SSOURL == "" {
		return "", errors.New("sso_url missing")
	}
	if cfg.EntityID == "" {
		return "", errors.New("entity_id missing")
	}
	u, err := url.Parse(cfg.SSOURL)
	if err != nil {
		return "", fmt.Errorf("parse sso URL: %w", err)
	}
	q := u.Query()
	q.Set("SAMLRequest", "") // empty marker; full XML-DSig deferred to Phase 2
	q.Set("RelayState", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// decodeSAMLConfig is a thin wrapper around json.Unmarshal that
// returns errors with the same shape as the other decoders.
func decodeSAMLConfig(raw []byte) (*samlConfig, error) {
	var cfg samlConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return &cfg, nil
}
