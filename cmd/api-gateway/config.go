package main

import "os"

// config is the runtime configuration for the API gateway.
type config struct {
	HTTPAddr    string
	DatabaseURL string
	JWTSecret   string
}

// loadConfig reads configuration from environment with sensible defaults.
func loadConfig() config {
	return config{
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://ios:3d5cb43fba1f82283a2ba02c79e116cf@192.168.0.116:5432/ios?sslmode=disable"),
		JWTSecret:   getenv("JWT_SECRET", "dev-jwt-secret-change-me-in-production-please-32bytes"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
