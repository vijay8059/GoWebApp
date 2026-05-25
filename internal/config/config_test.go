package config

import (
	"testing"
	"time"
)

func TestLoad_DefaultsInDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.ShutdownTimeout != 15*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 15s", cfg.ShutdownTimeout)
	}
}

func TestLoad_ProductionRequiresSessionKey(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_AUTH_KEY", "")
	t.Setenv("LOG_FORMAT", "json")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when SESSION_AUTH_KEY is missing in production")
	}
}

func TestLoad_ParsesDuration(t *testing.T) {
	t.Setenv("HTTP_READ_TIMEOUT", "3s")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ReadTimeout != 3*time.Second {
		t.Errorf("ReadTimeout = %v, want 3s", cfg.ReadTimeout)
	}
}
