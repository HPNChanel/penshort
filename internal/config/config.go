// Package config provides application configuration management.
// Configuration is loaded from environment variables following 12-factor principles.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
)

// Config holds all application configuration.
// All fields are populated from environment variables.
type Config struct {
	// Application settings
	AppEnv  string `env:"APP_ENV" envDefault:"development"`
	AppPort int    `env:"APP_PORT" envDefault:"8080"`

	// Database (PostgreSQL)
	DatabaseURL string `env:"DATABASE_URL,required"`

	// Cache (Redis)
	RedisURL string `env:"REDIS_URL,required"`

	// Base URL for short links (e.g., https://pen.sh)
	BaseURL string `env:"BASE_URL" envDefault:"http://localhost:8080"`

	// Logging
	LogLevel  string `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"json"`

	// Server timeouts
	ReadTimeout     time.Duration `env:"READ_TIMEOUT" envDefault:"5s"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"30s"`

	// Rate limiting
	RateLimitAPIEnabled      bool `env:"RATE_LIMIT_API_ENABLED" envDefault:"true"`
	RateLimitRedirectEnabled bool `env:"RATE_LIMIT_REDIRECT_ENABLED" envDefault:"true"`
	RateLimitRedirectRPS     int  `env:"RATE_LIMIT_REDIRECT_RPS" envDefault:"100"`
	RateLimitRedirectBurst   int  `env:"RATE_LIMIT_REDIRECT_BURST" envDefault:"20"`

	// CORS configuration
	// Comma-separated list of allowed origins (e.g., "https://example.com,https://app.example.com")
	CORSAllowedOrigins string `env:"CORS_ALLOWED_ORIGINS" envDefault:""`

	// Request body size limit in bytes (default 1MB)
	MaxRequestBodySize int64 `env:"MAX_REQUEST_BODY_SIZE" envDefault:"1048576"`
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// GetCORSAllowedOrigins parses the comma-separated origins string into a slice.
func (c *Config) GetCORSAllowedOrigins() []string {
	if c.CORSAllowedOrigins == "" {
		return nil
	}

	origins := strings.Split(c.CORSAllowedOrigins, ",")
	result := make([]string, 0, len(origins))

	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// Load parses environment variables and returns a Config.
// Returns an error if required variables are missing.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}

