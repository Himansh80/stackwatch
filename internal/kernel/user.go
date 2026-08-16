package kernel

import (
	"time"

	"github.com/google/uuid"
)

// User is a login identity. Always belongs to a Tenant.
type User struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	TenantID      uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Email         string     `json:"email" db:"email"`
	FullName      string     `json:"full_name" db:"full_name"`
	PasswordHash  string     `json:"-" db:"password_hash"`
	Role          string     `json:"role" db:"role"`
	Status        string     `json:"status" db:"status"`
	MustChangePW  bool       `json:"must_change_password" db:"must_change_password"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	EmailVerified bool       `json:"email_verified" db:"email_verified"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// UserStatus constants.
const (
	UserActive   = "active"
	UserDisabled = "disabled"
	UserPending  = "pending"
	UserInvited  = "invited"
)

// IsSuperAdmin returns true if user is the platform super-admin.
func (u *User) IsSuperAdmin() bool {
	return u != nil && u.Role == RoleSuperAdmin
}

// IsActive reports whether the user can log in.
func (u *User) IsActive() bool {
	return u != nil && u.Status == UserActive
}
