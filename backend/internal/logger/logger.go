package logger

import (
	"log/slog"
	"os"
	"strings"
)

var levels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

// New writes text logs to stdout. LOG_FORMAT=json switches to JSON.
// LOG_LEVEL is debug, info, warn, or error. Unknown values stay at info.
func New() *slog.Logger {
	level := slog.LevelInfo
	if parsed, ok := levels[strings.ToLower(os.Getenv("LOG_LEVEL"))]; ok {
		level = parsed
	}

	options := &slog.HandlerOptions{Level: level}
	if strings.EqualFold(os.Getenv("LOG_FORMAT"), "json") {
		return slog.New(slog.NewJSONHandler(os.Stdout, options))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, options))
}
