// Package config loads and validates application configuration from YAML + environment variables.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// SMTPConfig holds SMTP server settings for email sending.
type SMTPConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
	From string `yaml:"from"`
}

// Config holds all application configuration.
type Config struct {
	Port          int            `yaml:"port"`
	API           APIConfig      `yaml:"api"`
	Database      DatabaseConfig `yaml:"database"`
	Auth          AuthConfig     `yaml:"auth"`
	RateLimit     RateLimitConfig `yaml:"rate_limit"`
	SessionSecret string         `yaml:"session_secret"`
	SMTP          SMTPConfig     `yaml:"smtp"`
	AppHost       string         `yaml:"app_host"`
	TemplateMode  string         `yaml:"template_mode"` // "dev" (CDN) or "prod" (prebuilt)
	Cookie        CookieConfig   `yaml:"cookie"`
}

// APIConfig holds HTTP server settings.
type APIConfig struct {
	Port int `yaml:"port"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	URL string `yaml:"url"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	SessionExpiry time.Duration `yaml:"session_expiry"`
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	AuthRequests int           `yaml:"auth_requests"`
	AuthWindow   time.Duration `yaml:"auth_window"`
}

// CookieConfig holds cookie security settings.
type CookieConfig struct {
	Secure   bool   `yaml:"secure"`
	SameSite string `yaml:"same_site"` // "Strict", "Lax", "None"
}

// ResolveCookie returns the resolved cookie configuration.
// APP_ENV overrides YAML for the Secure flag:
//   - APP_ENV=development → Secure: false
//   - APP_ENV=production  → Secure: true
//   - No APP_ENV set      → Secure: false (safe for localhost dev)
func (c *Config) ResolveCookie() CookieConfig {
	cfg := c.Cookie
	if cfg.SameSite == "" {
		cfg.SameSite = "Lax"
	}
	switch os.Getenv("APP_ENV") {
	case "development":
		cfg.Secure = false
	case "production":
		cfg.Secure = true
	// default: Secure stays false for safe localhost dev
	}
	return cfg
}

// Load reads configuration from a YAML file and falls back to environment variables.
func Load(path string) (*Config, error) {
	var cfg Config

	data, err := os.ReadFile(path)
	if err != nil {
		// Fallback to environment variables
		cfg = Config{
			Port:    envInt("PORT", 8080),
			Database: DatabaseConfig{
				URL: envOr("DATABASE_URL", "postgres://sv_app:sv_dev_pass@localhost:5432/svensktvin"),
			},
			Auth: AuthConfig{
				SessionExpiry: 30 * 24 * time.Hour,
			},
			RateLimit: RateLimitConfig{
				AuthRequests: 5,
				AuthWindow:   15 * time.Minute,
			},
			SMTP: SMTPConfig{
				Port: 587,
			},
			TemplateMode: "dev",
			Cookie:       CookieConfig{Secure: false, SameSite: "Lax"},
		}
		return &cfg, nil
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Apply environment variable overrides (env takes precedence)
	if cfg.Port == 0 {
		cfg.Port = envInt("PORT", 8080)
	}
	if cfg.Database.URL == "" {
		cfg.Database.URL = envOr("DATABASE_URL", "postgres://sv_app:sv_dev_pass@localhost:5432/svensktvin")
	}
	if cfg.Auth.SessionExpiry == 0 {
		cfg.Auth.SessionExpiry = 30 * 24 * time.Hour
	}
	if cfg.RateLimit.AuthRequests == 0 {
		cfg.RateLimit.AuthRequests = 5
	}
	if cfg.RateLimit.AuthWindow == 0 {
		cfg.RateLimit.AuthWindow = 15 * time.Minute
	}
	if cfg.SMTP.Port == 0 {
		cfg.SMTP.Port = 587
	}

	// Environment variable overrides for SMTP
	if smtpHost := os.Getenv("SMTP_HOST"); smtpHost != "" {
		cfg.SMTP.Host = smtpHost
	}
	if smtpPort := os.Getenv("SMTP_PORT"); smtpPort != "" {
		if p := envInt("SMTP_PORT", 0); p > 0 {
			cfg.SMTP.Port = p
		}
	}
	if smtpUser := os.Getenv("SMTP_USER"); smtpUser != "" {
		cfg.SMTP.User = smtpUser
	}
	if smtpPass := os.Getenv("SMTP_PASS"); smtpPass != "" {
		cfg.SMTP.Pass = smtpPass
	}
	if smtpFrom := os.Getenv("SMTP_FROM"); smtpFrom != "" {
		cfg.SMTP.From = smtpFrom
	}
	if sessionSecret := os.Getenv("SESSION_SECRET"); sessionSecret != "" {
		cfg.SessionSecret = sessionSecret
	}
	if appHost := os.Getenv("APP_HOST"); appHost != "" {
		cfg.AppHost = appHost
	}

	return &cfg, nil
}

// envOr returns the value of the environment variable named by key, or fallback if empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envInt returns the integer value of the environment variable named by key, or fallback if empty/invalid.
func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n := 0
	fmt.Sscanf(v, "%d", &n)
	if n == 0 {
		return fallback
	}
	return n
}
