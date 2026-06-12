// Package config loads and validates application configuration from YAML + env vars.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	API     APIConfig     `yaml:"api"`
	Database DatabaseConfig `yaml:"database"`
	Auth    AuthConfig     `yaml:"auth"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

type APIConfig struct {
	Port int `yaml:"port"`
}

type DatabaseConfig struct {
	URL string `yaml:"url"`
}

type AuthConfig struct {
	SessionExpiry time.Duration `yaml:"session_expiry"`
}

type RateLimitConfig struct {
	AuthRequests int           `yaml:"auth_requests"`
	AuthWindow   time.Duration `yaml:"auth_window"`
	WriteRequests int          `yaml:"write_requests"`
	WriteWindow  time.Duration `yaml:"write_window"`
}

func Load(path string) (*Config, error) {
	var cfg Config

	data, err := os.ReadFile(path)
	if err != nil {
		// Fallback to env vars if config file not found
		cfg = Config{
			API: APIConfig{
				Port: envInt("API_PORT", 9091),
			},
			Database: DatabaseConfig{
				URL: envOr("DATABASE_URL", "postgres://sv_app:sv_dev_pass@localhost:5433/svensktvin"),
			},
			Auth: AuthConfig{
				SessionExpiry: 30 * 24 * time.Hour,
			},
			RateLimit: RateLimitConfig{
				AuthRequests:  3,
				AuthWindow:    5 * time.Minute,
				WriteRequests: 30,
				WriteWindow:   1 * time.Minute,
			},
		}
		return &cfg, nil
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.API.Port == 0 {
		cfg.API.Port = 9091
	}
	if cfg.Auth.SessionExpiry == 0 {
		cfg.Auth.SessionExpiry = 30 * 24 * time.Hour
	}
	return &cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

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
