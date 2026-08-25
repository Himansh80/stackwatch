package mobile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// fcmLegacyEndpoint is FCM's classic HTTP endpoint (FCM v1 service-account
// JSON is more complex; the legacy endpoint just needs a server key).
const fcmLegacyEndpoint = "https://fcm.googleapis.com/fcm/send"

// dispatchToFCM sends one message via FCM Legacy HTTP. Returns the FCM
// message id on success, or (empty, error) on failure.
//
// Skipped (returns ErrFCMDisabled) if FCM_SERVER_KEY is not set — this
// lets Phase 1 ship without requiring the operator to wire FCM.
func dispatchToFCM(ctx context.Context, client *http.Client, msg fcmMessage) (string, error) {
	serverKey := os.Getenv("FCM_SERVER_KEY")
	if serverKey == "" {
		return "", ErrFCMDisabled
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("fcm: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fcmLegacyEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("fcm: new request: %w", err)
	}
	req.Header.Set("Authorization", "key="+serverKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fcm: do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("fcm: http %d", resp.StatusCode)
	}

	var out struct {
		MessageID int64  `json:"message_id"`
		Error     string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("fcm: decode: %w", err)
	}
	if out.Error != "" {
		return "", fmt.Errorf("fcm: %s", out.Error)
	}
	return fmt.Sprintf("%d", out.MessageID), nil
}

// dispatchToAPNs is a Phase 2 stub. iOS push requires a JWT signed with
// APNs auth key + a device token per app. We log + return ErrAPNsDisabled
// for now; the worker still records push_log rows with status='skipped'.
func dispatchToAPNs(ctx context.Context, client *http.Client, msg fcmMessage) (string, error) {
	return "", ErrAPNsDisabled
}

// newHTTPClient returns a shared *http.Client with a 10s timeout (sufficient
// for FCM/APNs; the dispatcher itself ticks every 30s so this is the
// per-message upper bound).
func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// Errors returned by the dispatch functions.
var (
	ErrFCMDisabled  = fmt.Errorf("fcm: FCM_SERVER_KEY not set — push dispatch skipped")
	ErrAPNsDisabled = fmt.Errorf("apns: not implemented in Phase 1 — push dispatch skipped")
)
