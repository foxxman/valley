package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestNewHonorsLevelAndFormat(t *testing.T) {
	t.Setenv("LOG_LEVEL", "ERROR")
	t.Setenv("LOG_FORMAT", "json")

	got := New()
	if !got.Enabled(context.Background(), slog.LevelError) {
		t.Fatal("error level should be enabled")
	}
	if got.Enabled(context.Background(), slog.LevelWarn) {
		t.Fatal("warn should be disabled at error level")
	}
}

func TestNewDefaultsToInfo(t *testing.T) {
	t.Setenv("LOG_LEVEL", "nope")
	t.Setenv("LOG_FORMAT", "")

	got := New()
	if !got.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("info should be enabled")
	}
	if got.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("debug should be disabled")
	}
}
