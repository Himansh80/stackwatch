package handler

import "testing"

func TestPublicHostRedactsCredentials(t *testing.T) {
	got := publicHost(Host{
		ID:       "h1",
		Name:     "nas",
		BaseURL:  "https://nas.example",
		APIKey:   "secret-key",
		Password: "secret-password",
		Username: "admin",
	})
	if got.APIKey != "" || got.Password != "" {
		t.Fatalf("credentials leaked in public host: api_key=%q password=%q", got.APIKey, got.Password)
	}
	if got.Username != "admin" || got.Name != "nas" {
		t.Fatalf("public host metadata changed: %+v", got)
	}
}
