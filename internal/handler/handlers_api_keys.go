package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// APIKey is the REST shape of a row in api_keys.
// The hash + raw token are NEVER returned — only the prefix and metadata.
type APIKey struct {
	ID         string  `json:"id"`
	TenantID   string  `json:"tenant_id"`
	UserID     *string `json:"user_id,omitempty"`
	Name       string  `json:"name"`
	Prefix     string  `json:"prefix"`
	Scopes     []string `json:"scopes"`
	Revoked    bool    `json:"revoked"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

// apiKeyPrefix is stamped on every key for dashboard identification.
// Format: sw_<6 hex chars>_... — the prefix makes it scannable in a long list.
const apiKeyPrefix = "sw_"

// ListAPIKeys returns the caller's API keys (metadata only, never the hash).
func ListAPIKeys(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id, tenant_id, user_id, name, prefix, scopes,
			        revoked_at IS NOT NULL AS revoked,
			        last_used_at, created_at
			 FROM api_keys WHERE tenant_id = $1
			 ORDER BY created_at DESC`, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []APIKey{}
		for rows.Next() {
			var k APIKey
			var userID *uuid.UUID
			var lastUsed *time.Time
			var created time.Time
			if err := rows.Scan(&k.ID, &k.TenantID, &userID, &k.Name, &k.Prefix, &k.Scopes,
				&k.Revoked, &lastUsed, &created); err != nil {
				kernel.RespondError(c, err)
				return
			}
			if userID != nil {
				s := userID.String()
				k.UserID = &s
			}
			k.CreatedAt = created.UTC().Format(time.RFC3339)
			if lastUsed != nil {
				s := lastUsed.UTC().Format(time.RFC3339)
				k.LastUsedAt = &s
			}
			out = append(out, k)
		}
		kernel.RespondOK(c, gin.H{"api_keys": out, "total": len(out)})
	}
}

// CreateAPIKey body. The response includes the FULL token (only at creation).
type createAPIKeyReq struct {
	Name   string   `json:"name" binding:"required"`
	Scopes []string `json:"scopes"`
}

// CreateAPIKey issues a new key. The plaintext is returned ONCE — callers
// must store it themselves; we only store the sha256 hash.
func CreateAPIKey(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		claims, ok := auth.ClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r createAPIKeyReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		r.Name = strings.TrimSpace(r.Name)
		if r.Name == "" {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		scopes := r.Scopes
		if scopes == nil {
			scopes = []string{}
		}
		// Generate a strong random token: sw_<32 bytes base64>
		rawSuffix, err := randomB64(32)
		if err != nil {
			kernel.RespondError(c, kernel.ErrInternal)
			return
		}
		prefix := apiKeyPrefix + shortPrefix() + "_"
		token := prefix + rawSuffix

		h := sha256.Sum256([]byte(token))
		hash := hex.EncodeToString(h[:])

		newID := uuid.New()
		_, err = pool.Pgx().Exec(c.Request.Context(),
			`INSERT INTO api_keys (id, tenant_id, user_id, name, key_hash, prefix, scopes)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			newID, tenantID, claims.UserID, r.Name, hash, prefix, scopes)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":    newID.String(),
			"name":  r.Name,
			"prefix": prefix,
			"token": token, // returned ONCE
			"scopes": scopes,
		})
	}
}

// RevokeAPIKey marks a key as revoked. Soft delete (revoked_at), not hard.
func RevokeAPIKey(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		_, err = pool.Pgx().Exec(c.Request.Context(),
			`UPDATE api_keys SET revoked_at = now()
			 WHERE id = $1 AND tenant_id = $2`, id, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{"revoked": true})
	}
}

// randomB64 returns n random bytes, base64-url-encoded.
func randomB64(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// shortPrefix returns a 6-hex char prefix identifier stamped in the
// public-facing token (e.g. sw_a1b2c3_xxxx). 24 bits of randomness is
// enough to differentiate keys in a UI list.
func shortPrefix() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano()&0xFFFFFF, 16)
	}
	return fmt.Sprintf("%06x", int(b[0])<<16|int(b[1])<<8|int(b[2]))
}
