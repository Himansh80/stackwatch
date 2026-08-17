package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// User is the REST shape of a row in users.
type User struct {
	ID                  string  `json:"id"`
	TenantID            string  `json:"tenant_id"`
	Email               string  `json:"email"`
	FullName            string  `json:"full_name"`
	Role                string  `json:"role"`
	Status              string  `json:"status"`
	EmailVerified       bool    `json:"email_verified"`
	MustChangePassword  bool    `json:"must_change_password"`
	LastLoginAt         *string `json:"last_login_at,omitempty"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

// ListUsers returns all users in the caller's tenant.
func ListTier0Users(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id, tenant_id, email, full_name, role, status,
			        email_verified, must_change_password, last_login_at,
			        created_at, updated_at
			 FROM users WHERE tenant_id = $1
			 ORDER BY created_at DESC`, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []User{}
		for rows.Next() {
			var u User
			var lastLogin *time.Time
			var created, updated time.Time
			if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.FullName, &u.Role,
				&u.Status, &u.EmailVerified, &u.MustChangePassword,
				&lastLogin, &created, &updated); err != nil {
				kernel.RespondError(c, err)
				return
			}
			u.CreatedAt = created.UTC().Format(time.RFC3339)
			u.UpdatedAt = updated.UTC().Format(time.RFC3339)
			if lastLogin != nil {
				s := lastLogin.UTC().Format(time.RFC3339)
				u.LastLoginAt = &s
			}
			out = append(out, u)
		}
		kernel.RespondOK(c, gin.H{"users": out, "total": len(out)})
	}
}

// GetUserMe returns the calling user.
func GetTier0UserMe(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := auth.ClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		u, err := loadTier0User(c, pool, claims.UserID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{"user": u})
	}
}

// GetUser returns one user by id (scoped to tenant).
func GetTier0User(pool *db.Pool) gin.HandlerFunc {
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
		u, err := loadTier0UserInTenant(c, pool, id, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{"user": u})
	}
}

// CreateUser lets a tenant admin invite a new user.
func CreateTier0User(pool *db.Pool) gin.HandlerFunc {
	type req struct {
		Email       string `json:"email" binding:"required"`
		FullName    string `json:"full_name"`
		Role        string `json:"role"`
		Password    string `json:"password"`
	}
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		r.Email = strings.ToLower(strings.TrimSpace(r.Email))
		// Generate random password if not provided; user must change on login.
		pw := r.Password
		if pw == "" {
			pw = newRandomPassword()
		}
		hashed, err := auth.HashPassword(pw)
		if err != nil {
			kernel.RespondError(c, kernel.ErrInternal)
			return
		}
		role := r.Role
		if role == "" {
			role = "viewer"
		}
		newID := uuid.New()
		_, err = pool.Pgx().Exec(c.Request.Context(),
			`INSERT INTO users (id, tenant_id, email, full_name, password_hash, role, status, must_change_password, email_verified)
			 VALUES ($1, $2, $3, $4, $5, $6, 'active', $7, false)`,
			newID, tenantID, r.Email, r.FullName, hashed, role, pw == r.Password)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		c.JSON(201, gin.H{"id": newID.String(), "email": r.Email, "role": role})
	}
}

// UpdateUser lets a tenant admin adjust role/status/name.
func UpdateTier0User(pool *db.Pool) gin.HandlerFunc {
	type req struct {
		FullName *string `json:"full_name,omitempty"`
		Role     *string `json:"role,omitempty"`
		Status   *string `json:"status,omitempty"`
	}
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
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		_, err = pool.Pgx().Exec(c.Request.Context(),
			`UPDATE users SET
				full_name = COALESCE($1, full_name),
				role      = COALESCE($2, role),
				status    = COALESCE($3, status),
				updated_at = now()
			 WHERE id = $4 AND tenant_id = $5`,
			r.FullName, r.Role, r.Status, id, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{"updated": true})
	}
}

// DeleteUser removes a user from the tenant.
func DeleteTier0User(pool *db.Pool) gin.HandlerFunc {
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
			`DELETE FROM users WHERE id = $1 AND tenant_id = $2`, id, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{"deleted": true})
	}
}

// loadUser fetches a user by id (cross-tenant lookup, for /me).
func loadTier0User(c *gin.Context, pool *db.Pool, id uuid.UUID) (User, error) {
	var u User
	var lastLogin *time.Time
	var created, updated time.Time
	err := pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT id, tenant_id, email, full_name, role, status,
		        email_verified, must_change_password, last_login_at,
		        created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.FullName, &u.Role,
		&u.Status, &u.EmailVerified, &u.MustChangePassword,
		&lastLogin, &created, &updated)
	if err != nil {
		return User{}, err
	}
	u.CreatedAt = created.UTC().Format(time.RFC3339)
	u.UpdatedAt = updated.UTC().Format(time.RFC3339)
	if lastLogin != nil {
		s := lastLogin.UTC().Format(time.RFC3339)
		u.LastLoginAt = &s
	}
	return u, nil
}

// loadUserInTenant fetches a user by id but only if they belong to the given tenant.
func loadTier0UserInTenant(c *gin.Context, pool *db.Pool, id, tenantID uuid.UUID) (User, error) {
	var u User
	var lastLogin *time.Time
	var created, updated time.Time
	err := pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT id, tenant_id, email, full_name, role, status,
		        email_verified, must_change_password, last_login_at,
		        created_at, updated_at
		 FROM users WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.FullName, &u.Role,
		&u.Status, &u.EmailVerified, &u.MustChangePassword,
		&lastLogin, &created, &updated)
	if err != nil {
		return User{}, err
	}
	u.CreatedAt = created.UTC().Format(time.RFC3339)
	u.UpdatedAt = updated.UTC().Format(time.RFC3339)
	if lastLogin != nil {
		s := lastLogin.UTC().Format(time.RFC3339)
		u.LastLoginAt = &s
	}
	return u, nil
}

// newRandomPassword returns a 24-char random password. Pure go stdlib.
func newRandomPassword() string {
	const alphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var b [24]byte
	seed := time.Now().UnixNano()
	for i := range b {
		seed = (seed*1103515245 + 12345) & 0x7FFFFFFF
		b[i] = alphabet[seed%int64(len(alphabet))]
	}
	return string(b[:])
}
