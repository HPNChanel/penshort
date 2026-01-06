package config

import (
	"os"
	"testing"
)

func TestLoad_WithRequiredVars(t *testing.T) {
	// Set required env vars
	os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	os.Setenv("REDIS_URL", "redis://localhost:6379")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("REDIS_URL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.DatabaseURL != "postgres://test:test@localhost:5432/test" {
		t.Errorf("expected DatabaseURL to be set, got %s", cfg.DatabaseURL)
	}

	if cfg.RedisURL != "redis://localhost:6379" {
		t.Errorf("expected RedisURL to be set, got %s", cfg.RedisURL)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	// Ensure required vars are unset
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("REDIS_URL")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing required vars, got nil")
	}
}

func TestConfig_Defaults(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	os.Setenv("REDIS_URL", "redis://localhost:6379")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("REDIS_URL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.AppEnv != "development" {
		t.Errorf("expected default AppEnv 'development', got %s", cfg.AppEnv)
	}

	if cfg.AppPort != 8080 {
		t.Errorf("expected default AppPort 8080, got %d", cfg.AppPort)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("expected default LogLevel 'info', got %s", cfg.LogLevel)
	}

	if cfg.LogFormat != "json" {
		t.Errorf("expected default LogFormat 'json', got %s", cfg.LogFormat)
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	cfg := &Config{AppEnv: "development"}
	if !cfg.IsDevelopment() {
		t.Error("expected IsDevelopment to return true")
	}

	cfg.AppEnv = "production"
	if cfg.IsDevelopment() {
		t.Error("expected IsDevelopment to return false")
	}
}

func TestConfig_IsProduction(t *testing.T) {
	cfg := &Config{AppEnv: "production"}
	if !cfg.IsProduction() {
		t.Error("expected IsProduction to return true")
	}

	cfg.AppEnv = "development"
	if cfg.IsProduction() {
		t.Error("expected IsProduction to return false")
	}
}
