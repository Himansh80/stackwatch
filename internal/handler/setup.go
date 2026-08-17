// Package handler — Setup wizard (Tier 0.5).
//
// Handles first-run setup for self-hosted mode. In cloud mode, this
// endpoint always reports "already complete" since cloud users sign up
// via /auth/signup instead.
package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// SetupHandler bundles Tier 0.5 endpoints.
type SetupHandler struct {
	pool        *db.Pool
	installMode string
	issuer      *auth.Issuer
}

// NewSetupHandler constructs the handler. installMode is "cloud" or
// "self-hosted" — wired from config at startup.
func NewSetupHandler(pool *db.Pool, installMode string, issuer *auth.Issuer) *SetupHandler {
	return &SetupHandler{pool: pool, installMode: installMode, issuer: issuer}
}

// SetupStatus describes the current state of setup.
type SetupStatus struct {
	InstallMode string `json:"install_mode"`
	IsComplete  bool   `json:"is_complete"`
	NeedsSetup  bool   `json:"needs_setup"`
	AdminEmail  string `json:"admin_email,omitempty"`
	SetupDomain string `json:"setup_domain,omitempty"`
	PublicURL   string `json:"public_url,omitempty"`
	TLSMode     string `json:"tls_mode"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// GetSetupStatus returns current setup state. Always public (no auth).
func (h *SetupHandler) GetSetupStatus(c *gin.Context) {
	st := SetupStatus{InstallMode: h.installMode}

	var isComplete bool
	var adminUserID *uuid.UUID
	var domain, publicURL, tlsMode string
	var completedAt *time.Time
	err := h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT is_complete, admin_user_id, setup_domain, public_url, tls_mode, completed_at
         FROM setup_state WHERE id = TRUE`).Scan(&isComplete, &adminUserID,
		&domain, &publicURL, &tlsMode, &completedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if h.installMode == "cloud" {
				st.IsComplete = true
				kernel.RespondOK(c, st)
				return
			}
			st.NeedsSetup = true
			kernel.RespondOK(c, st)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	st.IsComplete = isComplete
	st.SetupDomain = domain
	st.PublicURL = publicURL
	st.TLSMode = tlsMode
	if completedAt != nil {
		st.CompletedAt = completedAt.UTC().Format(time.RFC3339)
	}
	st.NeedsSetup = !isComplete && h.installMode == "self-hosted"

	if adminUserID != nil {
		var email string
		if err := h.pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT email FROM users WHERE id = $1`, *adminUserID).Scan(&email); err == nil {
			st.AdminEmail = email
		}
	}

	// In cloud mode, signup is the flow — never show setup wizard
	if h.installMode == "cloud" {
		st.NeedsSetup = false
		st.IsComplete = true
	}

	kernel.RespondOK(c, st)
}

// InitializeSetupRequest is the wizard's final submit body.
type InitializeSetupRequest struct {
	AdminEmail    string `json:"admin_email" binding:"required,email"`
	AdminPassword string `json:"admin_password" binding:"required,min=8"`
	Domain        string `json:"domain"`
	TLSMode       string `json:"tls_mode"`
	PublicURL     string `json:"public_url"`
}

// InitializeSetupResponse is the success body.
type InitializeSetupResponse struct {
	AdminUserID  uuid.UUID `json:"admin_user_id"`
	AdminEmail   string    `json:"admin_email"`
	TenantID     uuid.UUID `json:"tenant_id"`
	TLSMode      string    `json:"tls_mode"`
	PublicURL    string    `json:"public_url"`
	SetupDomain  string    `json:"setup_domain"`
	AgentInstall string    `json:"agent_install"`
	LoginToken   string    `json:"login_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// InitializeSetup creates the first admin user + cert + records.
func (h *SetupHandler) InitializeSetup(c *gin.Context) {
	if h.installMode != "self-hosted" {
		kernel.RespondError(c, kernel.ErrForbidden)
		return
	}
	var isComplete bool
	err := h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT is_complete FROM setup_state WHERE id = TRUE`).Scan(&isComplete)
	if err == nil && isComplete {
		kernel.RespondError(c, kernel.ErrConflict)
		return
	}

	var req InitializeSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.AdminEmail))

	tlsMode := req.TLSMode
	if tlsMode == "" {
		tlsMode = "self_signed"
	}
	if tlsMode != "self_signed" && tlsMode != "lets_encrypt" && tlsMode != "external" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}

	domain := strings.TrimSpace(req.Domain)
	if tlsMode == "lets_encrypt" && domain == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}

	publicURL := strings.TrimSpace(req.PublicURL)
	if publicURL == "" {
		if domain != "" {
			if tlsMode == "self_signed" {
				publicURL = "https://" + domain + ":9443"
			} else {
				publicURL = "https://" + domain
			}
		} else {
			publicURL = "https://localhost:9443"
		}
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		kernel.RespondError(c, fmt.Errorf("hash password: %w", err))
		return
	}

	tenantID := uuid.Nil

	var adminUserID uuid.UUID
	err = h.pool.Pgx().QueryRow(c.Request.Context(),
		`INSERT INTO users (id, tenant_id, email, full_name, password_hash, role, status, must_change_password, created_at, updated_at)
         VALUES (gen_random_uuid(), $1, $2, $3, $4, 'super_admin', 'active', false, NOW(), NOW())
         RETURNING id`,
		tenantID, email, "Administrator", string(pwHash)).Scan(&adminUserID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			kernel.RespondError(c, kernel.ErrConflict)
			return
		}
		kernel.RespondError(c, err)
		return
	}

	certPath := ""
	keyPath := ""
	if tlsMode == "self_signed" {
		cp, kp, err := generateSelfSignedCert(domain)
		if err != nil {
			kernel.RespondError(c, fmt.Errorf("generate cert: %w", err))
			return
		}
		certPath, keyPath = cp, kp
	}

	_, err = h.pool.Pgx().Exec(c.Request.Context(),
		`UPDATE setup_state SET
            is_complete = TRUE,
            admin_user_id = $1,
            setup_domain = $2,
            public_url = $3,
            tls_mode = $4,
            tls_cert_path = $5,
            tls_key_path = $6,
            completed_at = NOW()
         WHERE id = TRUE`,
		adminUserID, domain, publicURL, tlsMode, certPath, keyPath)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}

	loginToken, expiresAt, err := h.issueSetupToken(adminUserID, tenantID, email)
	if err != nil {
		kernel.RespondError(c, fmt.Errorf("issue token: %w", err))
		return
	}

	agentInstall := buildAgentInstallCommand(publicURL, email)

	kernel.RespondOK(c, InitializeSetupResponse{
		AdminUserID:  adminUserID,
		AdminEmail:   email,
		TenantID:     tenantID,
		TLSMode:      tlsMode,
		PublicURL:    publicURL,
		SetupDomain:  domain,
		AgentInstall: agentInstall,
		LoginToken:   loginToken,
		ExpiresAt:    expiresAt,
	})
}

// RegenerateCert re-issues the self-signed cert (self-hosted only).
func (h *SetupHandler) RegenerateCert(c *gin.Context) {
	if h.installMode != "self-hosted" {
		kernel.RespondError(c, kernel.ErrForbidden)
		return
	}
	var domain string
	err := h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT setup_domain FROM setup_state WHERE id = TRUE`).Scan(&domain)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	certPath, keyPath, err := generateSelfSignedCert(domain)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	_, err = h.pool.Pgx().Exec(c.Request.Context(),
		`UPDATE setup_state SET tls_cert_path = $1, tls_key_path = $2, updated_at = NOW() WHERE id = TRUE`,
		certPath, keyPath)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"cert_path": certPath,
		"key_path":  keyPath,
		"domain":    domain,
		"status":    "regenerated",
	})
}

// generateSelfSignedCert creates a 2048-bit RSA cert valid for 1 year.
func generateSelfSignedCert(domain string) (string, string, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{Organization: []string{"StackWatch Self-Hosted"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if domain != "" {
		template.DNSNames = append(template.DNSNames, domain)
		if ip := net.ParseIP(domain); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}
	template.IPAddresses = append(template.IPAddresses, net.IPv4(127, 0, 0, 1), net.IPv6loopback)

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", "", err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	dir := "/opt/stackwatch/certs"
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", err
	}
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return "", "", err
	}
	return certPath, keyPath, nil
}

// issueSetupToken creates a JWT for immediate login after setup.
func (h *SetupHandler) issueSetupToken(userID, tenantID uuid.UUID, email string) (string, time.Time, error) {
	if h.issuer == nil {
		return "", time.Time{}, errors.New("issuer not configured")
	}
	tok, err := h.issuer.Issue(userID, tenantID, email, "super_admin", false)
	if err != nil {
		return "", time.Time{}, err
	}
	// Decode TTL from issuer (it has TTLSeconds method)
	expiresAt := time.Now().Add(time.Duration(h.issuer.TTLSeconds()) * time.Second)
	return tok, expiresAt, nil
}

// buildAgentInstallCommand returns a curl one-liner the user can copy.
func buildAgentInstallCommand(publicURL, email string) string {
	return fmt.Sprintf(
		"curl -fsSL %s/install-agent.sh | sudo BACKEND=%s EMAIL=%s bash",
		publicURL, publicURL, email,
	)
}
