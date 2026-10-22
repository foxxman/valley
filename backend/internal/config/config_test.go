package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadRequiresDatabaseAndSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected error without DATABASE_URL")
	}

	t.Setenv("DATABASE_URL", "postgres://game@localhost/game")
	t.Setenv("JWT_SECRET", "short")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for a short JWT_SECRET")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://game@localhost/game")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("ACCESS_TTL", "")
	t.Setenv("REFRESH_TTL", "")
	t.Setenv("COOKIE_SECURE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("addr %s", cfg.HTTPAddr)
	}
	if cfg.AccessTTL != 15*time.Minute {
		t.Fatalf("access ttl %s", cfg.AccessTTL)
	}
	if cfg.RefreshTTL != 30*24*time.Hour {
		t.Fatalf("refresh ttl %s", cfg.RefreshTTL)
	}
	if cfg.CookieSecure {
		t.Fatal("cookie secure should default to false")
	}
}

func TestLoadRejectsBadDurations(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://game@localhost/game")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("ACCESS_TTL", "nope")

	if _, err := Load(); err == nil {
		t.Fatal("expected duration error")
	}
}
