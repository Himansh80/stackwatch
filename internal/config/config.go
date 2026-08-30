// Package config is the single source of truth for every environment variable
// read across the whole StackWatch process tree.
//
// Rules:
//   - The ONLY place `os.Getenv` may be called is inside this package.
//   - All binaries (api-gateway, web-terminal, truenas-connector, agent,
//     alert-engine, ai-engine, ...) call config.Load() at startup and
//     receive a populated struct.
//   - .env files are supported via godotenv (only if present — not required).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the runtime configuration shared by every StackWatch binary.
type Config struct {
	// Common (all binaries)
	InstallMode  InstallMode `json:"install_mode"`   // "cloud" (default) | "self-hosted"
	LogLevel     string      `json:"log_level"`      // "debug", "info" (default), "warn", "error"
	BuildVersion string      `json:"build_version"`  // set by -ldflags at build time

	// Postgres (all backend binaries)
	DatabaseURL string `json:"-"` // sensitive — never serialize

	// Auth/JWT (api-gateway, web-terminal, agent)
	JWTSecret         string `json:"-"` // sensitive
	Issuer            string `json:"issuer"`
	AccessTokenTTL    time.Duration `json:"access_token_ttl"`
	RefreshTokenTTL   time.Duration `json:"refresh_token_ttl"`
	SignupEnabled     bool          `json:"signup_enabled"`

	// Server / Fleet / Agent
	AgentListenAddress string `json:"agent_listen_addr"` // ":9101"
	AgentBridgeKey     string `json:"-"`                 // sensitive

	// Sidecars (api-gateway calls these)
	WebTerminalURL    string        `json:"web_terminal_url"`
	TrueNASConnectorURL string      `json:"truenas_connector_url"`
	AIEngineURL        string        `json:"ai_engine_url"`
	AlertEngineURL     string        `json:"alert_engine_url"`
	IngestServiceURL   string        `json:"ingest_service_url"`
	StatusPageURL      string        `json:"status_page_url"`
	SmartSwitchURL     string        `json:"smart_switch_url"`

	// HTTP Server
	APIAddr      string        `json:"api_addr"`       // ":8080" by default — api-gateway
	WebTerminalAddr string     `json:"web_terminal_addr"` // ":8085"
	TrueNASAddr    string      `json:"truenas_addr"`   // ":8088"
	ReadTimeout    time.Duration `json:"read_timeout"`
	WriteTimeout   time.Duration `json:"write_timeout"`
	IdleTimeout    time.Duration `json:"idle_timeout"`

	// Rate limiting (per-identity)
	RateLimit struct {
		LoginMax    int `json:"login_max"`        // attempts
		LoginWindow time.Duration `json:"login_window"` // time window
	}

	// Security headers (CORS, HSTS, etc.)
	AllowedOrigins []string `json:"allowed_origins"`

	// Backup / Restore
	BackupRootDir      string `json:"backup_root_dir"`
	BackupSchedule     string `json:"backup_schedule"`     // cron format
	BackupRetentionDay int    `json:"backup_retention_day"` // how many days to keep

	// Push (FCM)
	FCMServerKey string `json:"-"` // sensitive

	// Ollama / LLM for AI features
	OllamaURL   string `json:"ollama_url"`
	OllamaModel string `json:"ollama_model"`
}

// InstallMode is the platform-wide mode enum.
type InstallMode string

const (
	InstallModeCloud      InstallMode = "cloud"
	InstallModeSelfHosted InstallMode = "self-hosted"
)

// Load returns the current process config by reading environment variables.
// Caller should call AFTER godotenv.Load() (if present) so .env takes effect.
func Load() *Config {
	cfg := &Config{}
	cfg.applyDefaults()
	cfg.parseEnv()
	return cfg
}

func (c *Config) applyDefaults() {
	c.InstallMode = InstallModeCloud
	c.LogLevel = "info"
	c.BuildVersion = "dev"
	c.DatabaseURL = "postgres://ios:stackwatch@localhost:5432/ios?sslmode=disable"
	c.JWTSecret = "dev-jwt-secret-change-me-in-production-32bytes"
	c.Issuer = "stackwatch"
	c.AccessTokenTTL = 15 * time.Minute
	c.RefreshTokenTTL = 7 * 24 * time.Hour
	c.SignupEnabled = true
	c.AgentListenAddress = "127.0.0.1:9101"
	c.APIAddr = ":8080"
	c.WebTerminalAddr = ":8085"
	c.TrueNASAddr = ":8088"
	c.ReadTimeout = 5 * time.Second
	c.WriteTimeout = 30 * time.Second
	c.IdleTimeout = 60 * time.Second
	c.RateLimit.LoginMax = 5
	c.RateLimit.LoginWindow = 5 * time.Minute
	c.BackupRootDir = "/var/backups/stackwatch"
	c.BackupSchedule = "0 2 * * *" // 2 AM daily
	c.BackupRetentionDay = 30
	c.OllamaURL = "http://127.0.0.1:11434"
	c.OllamaModel = "qwen2.5:1.5b"
}

func (c *Config) parseEnv() {
	c.InstallMode = InstallMode(getenv("INSTALL_MODE", string(c.InstallMode)))
	c.LogLevel = getenv("LOG_LEVEL", c.LogLevel)
	c.BuildVersion = getenv("BUILD_VERSION", c.BuildVersion)
	c.DatabaseURL = getenv("DATABASE_URL", c.DatabaseURL)
	c.JWTSecret = getenv("JWT_SECRET", c.JWTSecret)
	c.Issuer = getenv("JWT_ISSUER", c.Issuer)
	c.AccessTokenTTL = getDur("JWT_ACCESS_TTL", c.AccessTokenTTL)
	c.RefreshTokenTTL = getDur("JWT_REFRESH_TTL", c.RefreshTokenTTL)
	c.SignupEnabled = getBool("SIGNUP_ENABLED", c.SignupEnabled)
	c.AgentListenAddress = getenv("AGENT_LISTEN", c.AgentListenAddress)
	c.AgentBridgeKey = getenv("AGENT_BRIDGE_KEY", "")
	c.WebTerminalURL = getenv("WEB_TERMINAL_URL", c.WebTerminalURL)
	c.TrueNASConnectorURL = getenv("TRUENAS_CONNECTOR_URL", c.TrueNASConnectorURL)
	c.AIEngineURL = getenv("AI_ENGINE_URL", c.AIEngineURL)
	c.AlertEngineURL = getenv("ALERT_ENGINE_URL", c.AlertEngineURL)
	c.IngestServiceURL = getenv("INGEST_SERVICE_URL", c.IngestServiceURL)
	c.StatusPageURL = getenv("STATUS_PAGE_URL", c.StatusPageURL)
	c.SmartSwitchURL = getenv("SMART_SWITCH_URL", c.SmartSwitchURL)
	c.APIAddr = getenv("API_ADDR", c.APIAddr)
	c.WebTerminalAddr = getenv("WEB_TERMINAL_ADDR", c.WebTerminalAddr)
	c.TrueNASAddr = getenv("TRUENAS_ADDR", c.TrueNASAddr)
	c.ReadTimeout = getDur("READ_TIMEOUT", c.ReadTimeout)
	c.WriteTimeout = getDur("WRITE_TIMEOUT", c.WriteTimeout)
	c.IdleTimeout = getDur("IDLE_TIMEOUT", c.IdleTimeout)
	c.RateLimit.LoginMax = getInt("LOGIN_MAX", c.RateLimit.LoginMax)
	c.RateLimit.LoginWindow = getDur("LOGIN_WINDOW", c.RateLimit.LoginWindow)

	origins := getenv("ALLOWED_ORIGINS", "https://stackwatch.smarthomelab.fun")
	c.AllowedOrigins = splitAndTrim(origins, ",")

	c.BackupRootDir = getenv("BACKUP_ROOT_DIR", c.BackupRootDir)
	c.BackupSchedule = getenv("BACKUP_SCHEDULE", c.BackupSchedule)
	c.BackupRetentionDay = getInt("BACKUP_RETENTION_DAY", c.BackupRetentionDay)
	c.FCMServerKey = getenv("FCM_SERVER_KEY", "")
	c.OllamaURL = getenv("OLLAMA_URL", c.OllamaURL)
	c.OllamaModel = getenv("OLLAMA_MODEL", c.OllamaModel)
}

// Validate panics if configuration is invalid (called at boot).
// Only call in binaries that can't tolerate silent misconfig.
func (c *Config) Validate() error {
	if c.InstallMode != InstallModeCloud && c.InstallMode != InstallModeSelfHosted {
		return fmt.Errorf("invalid INSTALL_MODE %q: must be 'cloud' or 'self-hosted'", c.InstallMode)
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET too short (%d chars): must be ≥32 chars", len(c.JWTSecret))
	}
	return nil
}

// IsCloud returns true iff INSTALL_MODE=cloud.
func (c *Config) IsCloud() bool { return c.InstallMode == InstallModeCloud }

// IsSelfHosted returns true iff INSTALL_MODE=self-hosted.
func (c *Config) IsSelfHosted() bool { return c.InstallMode == InstallModeSelfHosted }

// helpers

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getDur(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func getBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
