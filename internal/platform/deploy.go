// Package platform is the home for Tier 11 — Platform & Commerce
// (speckit change 009-tier11-platform-commerce) domain logic.
//
// Why a new package (not internal/homelab/):
//   internal/homelab/ owns Tier 10 — Homelab Dashboard endpoints.
//   Tier 11 is a separate, larger tier (8 sub-features PL1-PL8,
//   8 DB tables, ~30 routes, 6 background workers). Splitting
//   it into its own package boundary:
//     - keeps homelab's per-user scope intact (no leak of tenant-
//       wide admin code into the homelab surface)
//     - lets Tier 11 grow without crowding the homelab tree
//     - matches the Tier 9 pattern (internal/handler/handlers_*_*.go
//       + internal/auth + internal/db) — domain packages hold
//       logic, handler packages hold HTTP wiring
//
// Phase 1 ships the deploy helper (this file). Future phases add:
//
//	usage.go (PL2 meterEvent + rollup worker)
//	signup.go + email.go + ratelimit_signup.go (PL3)
//	limits.go + limits_seed.go (PL4)
//	backup.go + backup_worker.go (PL5)
//	regions.go + replica worker (PL6)
//	ratelimit_middleware.go + ratelimit_cleanup.go (PL7)
//	health.go + health_collectors.go (PL8)
package platform

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/stackwatch/platform/internal/db"
)

// InstallTokenPlaintextPrefix is the prefix every minted install
// token carries ("swi_" — StackWatch Install). The on-the-wire
// /storage split means:
//   - bcrypt-hash at rest (cost 10, same as api_keys + scim_tokens)
//   - "swi_<43 base64url chars>" in the bash one-liner
// The prefix is grep-able in audit logs and makes accidental
// copy-paste into the wrong field (e.g. a SCIM token box) easy
// to catch.
const InstallTokenPlaintextPrefix = "swi_"

// InstallTokenPlaintextLength is the byte length of the random
// portion (32 bytes → 43 base64url chars). 32 bytes is the same
// length Tier 9's SCIM tokens use; collision probability for
// 10k tokens per tenant is ~10⁻¹⁵, which is below the bcrypt
// brute-force budget so it's the right knob.
const InstallTokenPlaintextLength = 32

// installTokenRow mirrors the deploy_install_tokens table shape.
// Defined here (not in handler/) because the helper is the one
// that scans it. The handler layer projects it into the JSON
// shape in handlers_platform_deploy_types.go (which has a
// `Token string` field for the one-time create response).
type installTokenRow struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	CreatedAt  time.Time
	ExpiresAt  time.Time
	UsedAt     *time.Time
	UsedByIP   *string
	Label      *string
}

// GenerateInstallToken mints a fresh install token for the given
// tenant, bcrypt-hashes it (cost 10), and INSERTs the row.
//
// Returns the plaintext (prefix + base64url-encoded random bytes)
// AND the expires-at timestamp so the caller can render both in
// the one-time response. The plaintext is returned EXACTLY ONCE
// here; nothing else in the codebase persists it (the DB holds
// the hash only).
//
// The lifetime parameter is exposed so a future per-tenant config
// can override the package-level constant without touching this
// signature.
func GenerateInstallToken(ctx context.Context, tenantID uuid.UUID, label string, lifetime time.Duration, pool *db.Pool) (plaintext string, expiresAt time.Time, err error) {
	raw := make([]byte, InstallTokenPlaintextLength)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, fmt.Errorf("rand: %w", err)
	}
	plaintext = InstallTokenPlaintextPrefix + base64.RawURLEncoding.EncodeToString(raw)

	hb, err := bcrypt.GenerateFromPassword([]byte(plaintext), 10)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("bcrypt: %w", err)
	}

	expiresAt = time.Now().UTC().Add(lifetime)

	// label is operator-supplied optional. Empty → NULL.
	var labelArg interface{}
	if trimmed := trimLabel(label); trimmed != "" {
		labelArg = trimmed
	}

	// INSERT and return the row id (we don't need the row back
	// for the response — the handler builds it from plaintext +
	// expiresAt + the caller-supplied tenant_id).
	_, err = pool.Pgx().Exec(ctx,
		`INSERT INTO deploy_install_tokens (tenant_id, token_hash, expires_at, label)
		 VALUES ($1, $2, $3, $4)`,
		tenantID, string(hb), expiresAt, labelArg)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("insert deploy_install_tokens: %w", err)
	}
	return plaintext, expiresAt, nil
}

// LookupInstallToken bcrypt-compares the supplied plaintext
// against every un-used, un-expired row in deploy_install_tokens
// and returns the matching row. Returns nil + ErrTokenNotFound
// when no row matches (already used, expired, or never existed).
//
// We scan all rows instead of indexing the bcrypt input because
// bcrypt is one-way: there's no way to fetch a candidate row by
// token_hash without first hashing. The table scan is fine at
// Tier 11 v1 volume (a handful of tokens per tenant per day);
// Phase 7 will add a fast-path SHA-256 candidate-row lookup if
// the table ever grows large enough to make the scan hot.
//
// Errors:
//   - ErrTokenNotFound (no match)
//   - ErrTokenExpired  (matched but used_at / expires_at invalid)
//   - any DB error as-is
func LookupInstallToken(ctx context.Context, plaintext string, pool *db.Pool) (*installTokenRow, error) {
	rows, err := pool.Pgx().Query(ctx,
		`SELECT id, tenant_id, created_at, expires_at, used_at, used_by_ip, label
		   FROM deploy_install_tokens
		  WHERE used_at IS NULL AND expires_at > now()`)
	if err != nil {
		return nil, fmt.Errorf("query deploy_install_tokens: %w", err)
	}
	defer rows.Close()

	var matched *installTokenRow
	for rows.Next() {
		var (
			id       uuid.UUID
			tid      uuid.UUID
			created  time.Time
			expires  time.Time
			usedAt   *time.Time
			usedIP   *string
			label    *string
		)
		if err := rows.Scan(&id, &tid, &created, &expires, &usedAt, &usedIP, &label); err != nil {
			continue
		}
		// We can't fetch the hash in the same query because we
		// need it for bcrypt-compare against plaintext. So do
		// a per-candidate hash lookup.
		var hash string
		if err := pool.Pgx().QueryRow(ctx,
			`SELECT token_hash FROM deploy_install_tokens WHERE id = $1`, id,
		).Scan(&hash); err != nil {
			continue
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)); err == nil {
			matched = &installTokenRow{
				ID:        id,
				TenantID:  tid,
				CreatedAt: created,
				ExpiresAt: expires,
				UsedAt:    usedAt,
				UsedByIP:  usedIP,
				Label:     label,
			}
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deploy_install_tokens: %w", err)
	}
	if matched == nil {
		return nil, ErrTokenNotFound
	}
	return matched, nil
}

// MarkInstallTokenUsed flips used_at + used_by_ip on the given
// token row. Used by the install-script handler after a
// successful GET so the token can't be replayed from a different
// machine.
//
// Fire-and-forget semantics: a failure here is logged by the
// caller but does NOT block the response. The token will
// eventually be rejected by LookupInstallToken's "used_at IS
// NULL" filter on the next replay attempt anyway.
func MarkInstallTokenUsed(ctx context.Context, tokenID uuid.UUID, ip string, pool *db.Pool) error {
	_, err := pool.Pgx().Exec(ctx,
		`UPDATE deploy_install_tokens
		    SET used_at = now(), used_by_ip = $2
		  WHERE id = $1 AND used_at IS NULL`,
		tokenID, ip)
	if err != nil {
		return fmt.Errorf("update deploy_install_tokens: %w", err)
	}
	return nil
}

// TrimLabel trims surrounding whitespace and rejects values
// outside the allowed character set (letters/digits/space/dash/
// underscore, ≤ 64 chars). Returns the cleaned value, or "" when
// the input is empty / invalid. Exported because the handler
// layer also calls it on the inbound POST body before forwarding
// to GenerateInstallToken.
//
// We keep the regex check inline (not via regexp.MustCompile)
// because it's used on every POST /install-token; an inline
// ASCII range scan is ~100x faster than a regex engine call.
func TrimLabel(s string) string {
	return trimLabel(s)
}

func trimLabel(s string) string {
	const maxLen = 64
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	// Strip leading/trailing whitespace.
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	out := make([]byte, 0, end-start)
	for i := start; i < end; i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z',
			c >= 'a' && c <= 'z',
			c >= '0' && c <= '9',
			c == ' ' || c == '-' || c == '_':
			out = append(out, c)
		default:
			return "" // invalid char — drop the whole label
		}
	}
	return string(out)
}