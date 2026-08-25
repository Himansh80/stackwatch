// Package mobile implements Tier 13 — Mobile (React Native + Push).
//
// Files:
//   - types.go          : row + payload types
//   - push_stack.go     : per-platform HTTP dispatcher (FCM + APNs stubs)
//   - push_dispatch.go  : 30s ticker that polls alerts + queues pushes
package mobile

import "time"

// pushDeviceRow mirrors the push_devices table.
type pushDeviceRow struct {
	ID           string
	TenantID     string
	UserID       string
	Platform     string // 'android' | 'ios' | 'web'
	FCMToken     string
	AppVersion   string
	DeviceName   string
	LastActiveAt *time.Time
	Enabled      bool
}

// pushLogRow mirrors the push_log table.
type pushLogRow struct {
	ID            string
	TenantID      string
	DeviceID      string
	AlertID       string
	Title         string
	Body          string
	Data          map[string]any
	Status        string // 'queued'|'sent'|'failed'|'skipped'
	FCMMessageID  string
	APNsID        string
	SentAt        *time.Time
	ErrorMessage  string
	CreatedAt     time.Time
}

// fcmMessage is the FCM legacy HTTP payload (we use legacy for simplicity
// — FCM v1 requires service-account JSON; legacy only needs the server key).
type fcmMessage struct {
	To           string            `json:"to"`
	Notification fcmNotification   `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
	Priority     string            `json:"priority,omitempty"` // 'normal' | 'high'
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon,omitempty"`
	Sound string `json:"sound,omitempty"`
}

// alertRow is the slice of alerts/push_log we need for dispatch decisions.
type alertRow struct {
	ID       string
	TenantID string
	UserID   string
	RuleName string
	Severity string
	Message  string
}
