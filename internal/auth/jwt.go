package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims is the JWT payload.
type Claims struct {
	UserID       uuid.UUID `json:"uid"`
	TenantID     uuid.UUID `json:"tid"`
	Role         string    `json:"role"`
	MustChangePW bool      `json:"mcp,omitempty"`
	Email        string    `json:"email"`
	jwt.RegisteredClaims
}

// Issuer signs and verifies JWTs.
type Issuer struct {
	secret []byte
	ttl    time.Duration
}

// NewIssuer creates an Issuer with the given secret and token lifetime.
func NewIssuer(secret string, ttl time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), ttl: ttl}
}

// Issue creates a signed token for the user.
func (i *Issuer) Issue(userID, tenantID uuid.UUID, email, role string, mustChange bool) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID:       userID,
		TenantID:     tenantID,
		Role:         role,
		MustChangePW: mustChange,
		Email:        email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(i.ttl)),
			Issuer:    "stackwatch",
			Subject:   userID.String(),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(i.secret)
}

// Verify parses and validates a token string.
func (i *Issuer) Verify(tokenStr string) (*Claims, error) {
	c := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, c, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return i.secret, nil
	})
	if err != nil || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return c, nil
}

// TTLSeconds returns the issuer TTL in seconds (helper for testing).
func (i *Issuer) TTLSeconds() int64 {
	return int64(i.ttl / time.Second)
}
