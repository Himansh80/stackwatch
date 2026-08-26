// Tier 9 Phase 1 — SSO Foundation (Tier 9.1).
// Public callback HTTP handlers + SSO state-signing helpers.
//
// Three public endpoints (no JWT — IdP-issued code/assertion is the
// credential):
//
//	GET  /api/v1/enterprise/sso/initiate       — InitiateSSO  (302 → IdP)
//	GET  /api/v1/enterprise/sso/callback       — OIDCCallback (code→token→JWT)
//	POST /api/v1/enterprise/sso/callback       — SAMLCallback (POST binding)
//
// State parameter: HMAC-SHA256 signed with SSO_STATE_SECRET env
// (falls back to JWT_SECRET in dev). Payload: {provider_id, nonce,
// exp_unix_seconds}. Round-trips through IdP redirect as
// base64url(payload).base64url(signature).
//
// Split from handlers_sso.go (which holds the 6 protected endpoints)
// so both files stay under the 400-LOC cap.
package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
)

// SSOStateSecret is the env var name for the HMAC key used to sign
// SSO state parameters. Falls back to JWT_SECRET in dev so the
// feature works without a separate config (with a startup warning
// logged elsewhere).
const SSOStateSecret = "SSO_STATE_SECRET"

// ssoStateTTL is how long an OIDC/SAML state parameter stays valid.
// 10 minutes — long enough for a human click + IdP auth, short
// enough that an intercepted state can't be replayed indefinitely.
const ssoStateTTL = 10 * time.Minute

// ssoStatePayload is the JSON shape we HMAC-sign into the `state`
// query parameter on OIDC/SAML redirects.
type ssoStatePayload struct {
	ProviderID string `json:"pid"`
	Nonce      string `json:"n"`
	Exp        int64  `json:"exp"` // unix seconds
}

// ssoIssuer bundles the auth issuer + DB pool for the public
// callback handlers. Passed in by routes.go when wiring them.
type ssoIssuer struct {
	pool   *db.Pool
	issuer *auth.Issuer
}

// NewSSOIssuer constructs the public-callback wiring bundle.
// Exported so cmd/api-gateway/routes.go can build one and pass it
// into the SSO callback handler factories.
func NewSSOIssuer(pool *db.Pool, issuer *auth.Issuer) *ssoIssuer {
	return &ssoIssuer{pool: pool, issuer: issuer}
}

// InitiateSSO starts the OIDC or SAML flow. Generates a signed state
// parameter, builds the IdP authorization URL, and 302-redirects
// the browser. Public (no JWT).
func InitiateSSO(iss *ssoIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		providerIDStr := strings.TrimSpace(c.Query("provider_id"))
		pid, err := uuid.Parse(providerIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "provider_id required (uuid)", "code": "bad_request"})
			return
		}
		var typ string
		var enabled bool
		var rawConfig []byte
		err = iss.pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT type, enabled, config FROM sso_providers WHERE id = $1`, pid,
		).Scan(&typ, &enabled, &rawConfig)
		if err != nil {
			if isNoRowsErr(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "provider not found", "code": "not_found"})
			} else {
				c.JSON(http.StatusBadGateway, gin.H{"error": "lookup failed", "code": "upstream_failed", "details": err.Error()})
			}
			return
		}
		if !enabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "provider disabled", "code": "forbidden"})
			return
		}
		state, err := signSSOState(ssoStatePayload{
			ProviderID: pid.String(),
			Nonce:      uuid.NewString(),
			Exp:        time.Now().Add(ssoStateTTL).Unix(),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sign state failed", "code": "internal"})
			return
		}
		var redirectURL string
		switch typ {
		case "oidc":
			redirectURL, err = buildOIDCAuthorizeURL(rawConfig, state)
		case "saml":
			redirectURL, err = buildSAMLAuthnRequestURL(rawConfig, state)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider type", "code": "bad_request"})
			return
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "build redirect URL failed", "code": "config_invalid", "details": err.Error()})
			return
		}
		c.Redirect(http.StatusFound, redirectURL)
	}
}

// OIDCCallback exchanges the authorization code for tokens at the
// IdP's token_endpoint, fetches user info, JIT-creates the user if
// new, and issues a StackWatch JWT. Public (no JWT).
func OIDCCallback(iss *ssoIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		state := c.Query("state")
		code := c.Query("code")
		if state == "" || code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing code or state", "code": "bad_request"})
			return
		}
		payload, err := verifySSOState(state)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state: " + err.Error(), "code": "bad_request"})
			return
		}
		pid, err := uuid.Parse(payload.ProviderID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state provider_id", "code": "bad_request"})
			return
		}
		var typ string
		var tenantID uuid.UUID
		var rawConfig []byte
		err = iss.pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT type, tenant_id, config FROM sso_providers WHERE id = $1`, pid,
		).Scan(&typ, &tenantID, &rawConfig)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "provider lookup failed", "code": "upstream_failed"})
			return
		}
		if typ != "oidc" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "provider is not OIDC", "code": "bad_request"})
			return
		}
		tok, err := exchangeOIDCCode(c.Request.Context(), rawConfig, code)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "code exchange failed", "code": "upstream_failed", "details": err.Error()})
			return
		}
		userInfo, err := fetchOIDCUserinfo(c.Request.Context(), rawConfig, tok.AccessToken)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "userinfo failed", "code": "upstream_failed", "details": err.Error()})
			return
		}
		if userInfo.Sub == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "userinfo missing sub", "code": "bad_request"})
			return
		}
		jwt, err := jitOrLoginOIDC(c.Request.Context(), iss, tenantID, pid, userInfo)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "JIT login failed", "code": "upstream_failed", "details": err.Error()})
			return
		}
		c.Redirect(http.StatusFound, "/auth/sso-done?token="+url.QueryEscape(jwt))
	}
}

// SAMLCallback handles the SAML POST binding (the IdP POSTs a signed
// SAMLResponse here). Validates audience + NotOnOrAfter, JIT-creates
// the user if new, and issues a StackWatch JWT. Public (no JWT).
func SAMLCallback(iss *ssoIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var samlResp string
		if err := c.Request.ParseForm(); err == nil {
			samlResp = c.Request.FormValue("SAMLResponse")
		}
		if samlResp == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing SAMLResponse", "code": "bad_request"})
			return
		}
		state := c.Request.FormValue("RelayState")
		var pid uuid.UUID
		var tenantID uuid.UUID
		if state != "" {
			payload, err := verifySSOState(state)
			if err == nil {
				if parsed, perr := uuid.Parse(payload.ProviderID); perr == nil {
					pid = parsed
				}
			}
		}
		if pid == uuid.Nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid RelayState", "code": "bad_request"})
			return
		}
		var rawConfig []byte
		err := iss.pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT tenant_id, config FROM sso_providers WHERE id = $1`, pid,
		).Scan(&tenantID, &rawConfig)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "provider lookup failed", "code": "upstream_failed"})
			return
		}
		cfg, err := decodeSAMLConfig(rawConfig)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "decode config: " + err.Error(), "code": "config_invalid"})
			return
		}
		x509PEM, err := decryptSecret(cfg.X509CertCiphertext, cfg.X509CertNonce)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "decrypt cert failed", "code": "internal"})
			return
		}
		raw, err := base64.StdEncoding.DecodeString(samlResp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "base64 decode failed", "code": "bad_request"})
			return
		}
		assertion, err := parseSAMLAssertion(raw, cfg.EntityID, x509PEM)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "SAML validation failed: " + err.Error(), "code": "bad_request"})
			return
		}
		jwt, err := jitOrLoginSAML(c.Request.Context(), iss, tenantID, pid, assertion)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "JIT login failed", "code": "upstream_failed", "details": err.Error()})
			return
		}
		c.Redirect(http.StatusFound, "/auth/sso-done?token="+url.QueryEscape(jwt))
	}
}

// ----- state HMAC helpers -----

// signSSOState signs a state payload with HMAC-SHA256 and returns
// "base64url(payload).base64url(signature)".
func signSSOState(payload ssoStatePayload) (string, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, ssoStateKeyBytes())
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	mac.Write([]byte(payloadB64))
	sigB64 := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payloadB64 + "." + sigB64, nil
}

// verifySSOState validates the signature + checks the exp is in the
// future. Returns the payload on success.
func verifySSOState(state string) (*ssoStatePayload, error) {
	parts := strings.SplitN(state, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("malformed state")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}
	mac := hmac.New(sha256.New, ssoStateKeyBytes())
	mac.Write([]byte(parts[0]))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, errors.New("signature mismatch")
	}
	var payload ssoStatePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	if payload.Exp > 0 && time.Now().Unix() > payload.Exp {
		return nil, errors.New("state expired")
	}
	return &payload, nil
}

// ssoStateKeyBytes returns the HMAC key for signing SSO state.
// Tries SSO_STATE_SECRET first; falls back to JWT_SECRET in dev.
func ssoStateKeyBytes() []byte {
	if k := strings.TrimSpace(getEnv(SSOStateSecret, "")); k != "" {
		return []byte(k)
	}
	return []byte(getEnv("JWT_SECRET", "dev-only-do-not-use-in-prod"))
}

// isNoRowsErr reports whether err is a pgx "no rows" error without
// forcing every handler file to import pgx directly.
func isNoRowsErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no rows")
}
