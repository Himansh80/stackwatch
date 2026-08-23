package main

import "os"

// config is the runtime configuration for the API gateway.
type config struct {
	HTTPAddr       string
	DatabaseURL    string
	JWTSecret      string
	InstallMode    string // "cloud" (default) or "self-hosted"
	WebTerminalURL string // URL of the web-terminal sidecar (Tier 4 admin SSH bridge)
}

// loadConfig reads configuration from environment with sensible defaults.
func loadConfig() config {
	mode := getenv("INSTALL_MODE", "cloud")
	if mode != "cloud" && mode != "self-hosted" {
		mode = "cloud"
	}
	return config{
		HTTPAddr:       getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:    getenv("DATABASE_URL", "postgres://ios:***@192.168.0.116:5432/ios?sslmode=disable"),
		JWTSecret:      getenv("JWT_SECRET", "dev-jwt-secret-change-me-in-production-please-32bytes"),
		InstallMode:    mode,
		WebTerminalURL: getenv("WEB_TERMINAL_URL", "http://127.0.0.1:8085"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
