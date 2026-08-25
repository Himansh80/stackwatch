// Tier 13 Phase 2 — Mobile handler types (shared).
package handler

import "time"

// pushDeviceRow mirrors the push_devices DB row (returned by Phase 2 routes).
type pushDeviceRow struct {
	ID           string     `json:"id"`
	Platform     string     `json:"platform"`
	FCMToken     string     `json:"-"` // never serialize the token back
	AppVersion   string     `json:"app_version"`
	DeviceName   string     `json:"device_name"`
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
	Enabled      bool       `json:"enabled"`
}

// pushLogRow mirrors the push_log DB row.
type pushLogRow struct {
	ID           string     `json:"id"`
	DeviceID     string     `json:"device_id"`
	AlertID      string     `json:"alert_id,omitempty"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	Status       string     `json:"status"`
	FCMMessageID string     `json:"fcm_message_id,omitempty"`
	APNsID       string     `json:"apns_id,omitempty"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// registerDeviceReq is the POST body for /push/register.
type registerDeviceReq struct {
	Platform   string `json:"platform"   binding:"required,oneof=android ios web"`
	FCMToken   string `json:"fcm_token"   binding:"required,min=10,max=512"`
	AppVersion string `json:"app_version" binding:"max=64"`
	DeviceName string `json:"device_name" binding:"max=128"`
}

// heartbeatReq is the POST body for /devices/heartbeat.
type heartbeatReq struct {
	AppVersion string `json:"app_version" binding:"max=64"`
	DeviceName string `json:"device_name" binding:"max=128"`
}

// refreshTokenReq is the POST body for /devices/refresh-token.
type refreshTokenReq struct {
	OldFCMToken string `json:"old_fcm_token" binding:"required"`
	NewFCMToken string `json:"new_fcm_token" binding:"required,min=10,max=512"`
}

// mobileSessionRow mirrors the mobile_sessions DB row.
type mobileSessionRow struct {
	ID           string    `json:"id"`
	DeviceName   string    `json:"device_name"`
	AppVersion   string    `json:"app_version"`
	IPAddress    string    `json:"ip_address,omitempty"`
	LastActiveAt time.Time `json:"last_active_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// testPushReq is the POST body for /test-push (super_admin only).
type testPushReq struct {
	Title string `json:"title" binding:"required,min=1,max=128"`
	Body  string `json:"body"  binding:"max=512"`
}
