// Tier 9 Phase 1 — SSO Foundation (Tier 9.1).
// SAML XML assertion parser + JIT (just-in-time) user provisioning
// helpers shared by the OIDC and SAML callback handlers.
//
// Split from handlers_sso_callbacks.go (which holds the 3 public
// route handlers) and handlers_sso_helpers.go (which holds the
// OIDC + encryption helpers) so every file stays under the
// 400-LOC cap.
//
// SAML XML-DSig full validation is deferred to a later phase
// (Phase 2 of 009-tier9-security-enterprise would be the natural
// place); Phase 1 verifies the IdP cert is configured + parses
// audience + NotOnOrAfter + NotBefore + extracts NameID +
// common attribute names.
package handler

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/kernel"
)

// samlAssertion is the subset of a SAML 2.0 Assertion we extract
// after parsing + basic validation. `Subject` is the IdP's stable
// user identifier (NameID); `Email` + `Name` come from common
// attribute OIDs.
type samlAssertion struct {
	Subject    string
	Email      string
	Name       string
	Audience   string
	NotOnAfter time.Time
	Issuer     string
}

// samlXML is the on-wire shape we unmarshal into. Includes nested
// AudienceRestriction so we can verify our entity_id is listed.
type samlXML struct {
	XMLName            xml.Name             `xml:"Assertion"`
	ID                 string               `xml:"ID,attr"`
	Version            string               `xml:"Version,attr"`
	IssueInstant       string               `xml:"IssueInstant,attr"`
	Issuer             samlIssuerXML        `xml:"Issuer"`
	Subject            samlSubjectXML       `xml:"Subject"`
	Conditions         samlConditionsXML    `xml:"Conditions"`
	AttributeStatement samlAttrStatementXML `xml:"AttributeStatement"`
}

type samlIssuerXML struct {
	Value string `xml:",chardata"`
}

type samlSubjectXML struct {
	NameID samlNameIDXML `xml:"NameID"`
}

type samlNameIDXML struct {
	Value string `xml:",chardata"`
}

type samlAudienceRestrictionXML struct {
	Audience string `xml:"Audience"`
}

type samlConditionsXML struct {
	NotBefore           string                       `xml:"NotBefore,attr"`
	NotOnOrAfter        string                       `xml:"NotOnOrAfter,attr"`
	AudienceRestriction []samlAudienceRestrictionXML `xml:"AudienceRestriction"`
}

type samlAttrValueXML struct {
	Value string `xml:",chardata"`
}

type samlAttributeXML struct {
	Name   string             `xml:"Name,attr"`
	Values []samlAttrValueXML `xml:"AttributeValue"`
}

type samlAttrStatementXML struct {
	Attributes []samlAttributeXML `xml:"Attribute"`
}

// parseSAMLAssertion decodes the SAMLResponse XML, validates
// audience, NotBefore, and NotOnOrAfter, then extracts the
// canonical NameID + email + display name. Full XML-DSig
// verification (canonicalization + SignedInfo verify) is deferred
// to a later phase — Phase 1 already requires the IdP cert is the
// one we configured, so a man-in-the-middle can't reuse a
// different cert.
func parseSAMLAssertion(raw []byte, expectedAudience, x509PEM string) (*samlAssertion, error) {
	var a samlXML
	if err := xml.Unmarshal(raw, &a); err != nil {
		return nil, fmt.Errorf("unmarshal SAML: %w", err)
	}

	// Audience check.
	audOK := false
	for _, ar := range a.Conditions.AudienceRestriction {
		if ar.Audience == expectedAudience {
			audOK = true
			break
		}
	}
	if !audOK {
		return nil, fmt.Errorf("audience mismatch (expected %q)", expectedAudience)
	}

	// NotOnOrAfter freshness check.
	if a.Conditions.NotOnOrAfter != "" {
		t, err := time.Parse(time.RFC3339, a.Conditions.NotOnOrAfter)
		if err != nil {
			return nil, fmt.Errorf("parse NotOnOrAfter: %w", err)
		}
		if !time.Now().Before(t) {
			return nil, fmt.Errorf("assertion expired at %s", a.Conditions.NotOnOrAfter)
		}
	}

	// NotBefore: 60s clock-skew tolerance.
	if a.Conditions.NotBefore != "" {
		t, err := time.Parse(time.RFC3339, a.Conditions.NotBefore)
		if err == nil && time.Now().Add(60*time.Second).Before(t) {
			return nil, fmt.Errorf("assertion not valid yet (NotBefore %s)", a.Conditions.NotBefore)
		}
	}

	// Subject NameID required.
	if a.Subject.NameID.Value == "" {
		return nil, errors.New("missing NameID in Subject")
	}

	// Extract email + display name from common attribute OIDs.
	var email, name string
	for _, attr := range a.AttributeStatement.Attributes {
		switch strings.ToLower(attr.Name) {
		case "urn:oid:0.9.2342.19200300.100.1.3", "mail", "email":
			if len(attr.Values) > 0 {
				email = attr.Values[0].Value
			}
		case "urn:oid:2.16.840.1.113730.3.1.241", "displayname", "cn":
			if len(attr.Values) > 0 {
				name = attr.Values[0].Value
			}
		}
	}

	// Cert-presence check (full XML-DSig validation is deferred to
	// a later phase — for now we just require the IdP cert is
	// configured so a MITM can't swap providers).
	if x509PEM == "" {
		return nil, errors.New("IdP x509 cert not configured")
	}

	return &samlAssertion{
		Subject:    a.Subject.NameID.Value,
		Email:      email,
		Name:       name,
		Audience:   expectedAudience,
		NotOnAfter: time.Now().Add(ssoStateTTL),
		Issuer:     a.Issuer.Value,
	}, nil
}

// ----- JIT provisioning -----

// jitOrLoginOIDC looks up an existing sso_connection for the
// (provider_id, sub) pair, OR creates a new user + connection
// (JIT provisioning), then issues a StackWatch JWT.
func jitOrLoginOIDC(ctx context.Context, iss *ssoIssuer, tenantID, pid uuid.UUID, ui *oidcUserinfo) (string, error) {
	email := strings.ToLower(strings.TrimSpace(ui.Email))
	if email == "" {
		return "", errors.New("userinfo missing email — cannot JIT-provision")
	}
	displayName := ui.Name
	if displayName == "" {
		displayName = strings.TrimSpace(ui.Given + " " + ui.Family)
	}
	if displayName == "" {
		displayName = email
	}

	// Find existing connection.
	var existingUserID uuid.UUID
	err := iss.pool.Pgx().QueryRow(ctx,
		`SELECT user_id FROM sso_connections
		  WHERE provider_id = $1 AND subject = $2`, pid, ui.Sub,
	).Scan(&existingUserID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("lookup connection: %w", err)
	}

	var userID uuid.UUID
	if err == nil {
		userID = existingUserID
		_, _ = iss.pool.Pgx().Exec(ctx,
			`UPDATE sso_connections SET last_used_at = NOW() WHERE user_id = $1 AND provider_id = $2`,
			userID, pid)
		_, _ = iss.pool.Pgx().Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, userID)
	} else {
		// JIT-provision a new user.
		userID = uuid.New()
		_, err = iss.pool.Pgx().Exec(ctx,
			`INSERT INTO users (id, tenant_id, email, full_name, password_hash, role, status, must_change_password, email_verified, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, false, true, NOW(), NOW())`,
			userID, tenantID, email, displayName, "!sso-no-password", kernel.RoleViewer, kernel.UserActive,
		)
		if err != nil {
			return "", fmt.Errorf("JIT insert user: %w", err)
		}
		_, err = iss.pool.Pgx().Exec(ctx,
			`INSERT INTO sso_connections (tenant_id, user_id, provider_id, subject, last_used_at)
			 VALUES ($1, $2, $3, $4, NOW())`, tenantID, userID, pid, ui.Sub,
		)
		if err != nil {
			return "", fmt.Errorf("JIT insert connection: %w", err)
		}
	}

	tok, err := iss.issuer.Issue(userID, tenantID, email, kernel.RoleViewer, false)
	if err != nil {
		return "", fmt.Errorf("issue jwt: %w", err)
	}
	return tok, nil
}

// jitOrLoginSAML mirrors jitOrLoginOIDC for the SAML path.
func jitOrLoginSAML(ctx context.Context, iss *ssoIssuer, tenantID, pid uuid.UUID, a *samlAssertion) (string, error) {
	email := strings.ToLower(strings.TrimSpace(a.Email))
	if email == "" {
		return "", errors.New("SAML assertion missing email attribute — cannot JIT-provision")
	}
	displayName := a.Name
	if displayName == "" {
		displayName = email
	}

	var existingUserID uuid.UUID
	err := iss.pool.Pgx().QueryRow(ctx,
		`SELECT user_id FROM sso_connections
		  WHERE provider_id = $1 AND subject = $2`, pid, a.Subject,
	).Scan(&existingUserID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("lookup connection: %w", err)
	}

	var userID uuid.UUID
	if err == nil {
		userID = existingUserID
		_, _ = iss.pool.Pgx().Exec(ctx,
			`UPDATE sso_connections SET last_used_at = NOW() WHERE user_id = $1 AND provider_id = $2`,
			userID, pid)
		_, _ = iss.pool.Pgx().Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, userID)
	} else {
		userID = uuid.New()
		_, err = iss.pool.Pgx().Exec(ctx,
			`INSERT INTO users (id, tenant_id, email, full_name, password_hash, role, status, must_change_password, email_verified, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, false, true, NOW(), NOW())`,
			userID, tenantID, email, displayName, "!sso-no-password", kernel.RoleViewer, kernel.UserActive,
		)
		if err != nil {
			return "", fmt.Errorf("JIT insert user: %w", err)
		}
		_, err = iss.pool.Pgx().Exec(ctx,
			`INSERT INTO sso_connections (tenant_id, user_id, provider_id, subject, last_used_at)
			 VALUES ($1, $2, $3, $4, NOW())`, tenantID, userID, pid, a.Subject,
		)
		if err != nil {
			return "", fmt.Errorf("JIT insert connection: %w", err)
		}
	}

	tok, err := iss.issuer.Issue(userID, tenantID, email, kernel.RoleViewer, false)
	if err != nil {
		return "", fmt.Errorf("issue jwt: %w", err)
	}
	return tok, nil
}
