// Package handler — Shared auth helper for terminal connections (Tier 3.6).
//
// dialConnection opens an SSH client to a saved connection using one of:
//   - auth_method = "key":                uses ssh_keys.ssh_key_id (the existing flow)
//   - auth_method = "password":           uses credentials.password
//   - auth_method = "key_with_passphrase": uses ssh_keys.ssh_key_id + credentials.key_passphrase
//
// On success, returns *ssh.Client (caller closes) and the auth method used (for telemetry).
package handler

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/ssh"

	"github.com/stackwatch/platform/internal/kernel"
)

// authMethod is the auth method string (matches DB CHECK constraint).
type authMethod string

const (
	authMethodKey               authMethod = "key"
	authMethodPassword          authMethod = "password"
	authMethodKeyWithPassphrase authMethod = "key_with_passphrase"
)

// ConnectionAuthResult is the result of dialConnection.
type ConnectionAuthResult struct {
	Client       *ssh.Client
	AuthMethod   string // which auth succeeded
	CredentialID *uuid.UUID
}

// dialConnection opens an SSH client to a saved connection using the appropriate auth method.
func (h *TerminalHandler) dialConnection(ctx context.Context, tenantID, connID uuid.UUID, timeout time.Duration) (*ConnectionAuthResult, error) {
	// Load connection details + auth_method + credential_id
	var host string
	var port int
	var user string
	var sshKeyID *uuid.UUID
	var method string
	var credentialID *uuid.UUID
	err := h.pool.Pgx().QueryRow(ctx,
		`SELECT host, port, user_, ssh_key_id, auth_method, credential_id
         FROM connections WHERE tenant_id = $1 AND id = $2`,
		tenantID, connID).Scan(&host, &port, &user, &sshKeyID, &method, &credentialID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, kernel.ErrNotFound
		}
		return nil, fmt.Errorf("load connection: %w", err)
	}

	switch authMethod(method) {
	case authMethodKey:
		if sshKeyID == nil {
			return nil, errors.New("connection has no ssh_key_id (auth_method=key)")
		}
		return h.dialWithKey(ctx, tenantID, host, port, user, *sshKeyID, "", timeout)
	case authMethodKeyWithPassphrase:
		if sshKeyID == nil {
			return nil, errors.New("connection has no ssh_key_id (auth_method=key_with_passphrase)")
		}
		if credentialID == nil {
			return nil, errors.New("connection has no credential_id (auth_method=key_with_passphrase)")
		}
		passphrase, err := h.getCredentialSecret(ctx, tenantID, *credentialID)
		if err != nil {
			return nil, fmt.Errorf("get passphrase: %w", err)
		}
		return h.dialWithKey(ctx, tenantID, host, port, user, *sshKeyID, passphrase, timeout)
	case authMethodPassword:
		if credentialID == nil {
			return nil, errors.New("connection has no credential_id (auth_method=password)")
		}
		password, err := h.getCredentialSecret(ctx, tenantID, *credentialID)
		if err != nil {
			return nil, fmt.Errorf("get password: %w", err)
		}
		return h.dialWithPassword(ctx, host, port, user, password, timeout)
	default:
		return nil, fmt.Errorf("unsupported auth_method: %s", method)
	}
}

// dialWithKey opens an SSH client using a stored private key (with optional passphrase).
func (h *TerminalHandler) dialWithKey(ctx context.Context, tenantID uuid.UUID, host string, port int, user string, sshKeyID uuid.UUID, passphrase string, timeout time.Duration) (*ConnectionAuthResult, error) {
	var privKeyPEM string
	err := h.pool.Pgx().QueryRow(ctx,
		`SELECT private_key FROM ssh_keys WHERE tenant_id = $1 AND id = $2`,
		tenantID, sshKeyID).Scan(&privKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("load key: %w", err)
	}

	var signer ssh.Signer
	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(privKeyPEM), []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey([]byte(privKeyPEM))
	}
	if err != nil {
		return nil, fmt.Errorf("parse key: %w", err)
	}

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}
	authMethod := authMethodKey
	if passphrase != "" {
		authMethod = authMethodKeyWithPassphrase
	}
	return &ConnectionAuthResult{
		Client:     client,
		AuthMethod: string(authMethod),
	}, nil
}

// dialWithPassword opens an SSH client using a stored password.
func (h *TerminalHandler) dialWithPassword(ctx context.Context, host string, port int, user, password string, timeout time.Duration) (*ConnectionAuthResult, error) {
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}
	return &ConnectionAuthResult{
		Client:     client,
		AuthMethod: string(authMethodPassword),
	}, nil
}

// getCredentialSecret decrypts and returns the plaintext of a credential.
func (h *TerminalHandler) getCredentialSecret(ctx context.Context, tenantID, credentialID uuid.UUID) (string, error) {
	var ct, nonce []byte
	err := h.pool.Pgx().QueryRow(ctx,
		`SELECT ciphertext, nonce FROM credentials
         WHERE tenant_id = $1 AND id = $2`, tenantID, credentialID).Scan(&ct, &nonce)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", kernel.ErrNotFound
		}
		return "", fmt.Errorf("load credential: %w", err)
	}
	return decryptSecret(ct, nonce)
}
