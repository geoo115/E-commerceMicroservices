// Package logger configures structured JSON logging with log/slog.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a JSON logger tagged with the service name and installs it as
// the slog default. The level is read from LOG_LEVEL (debug, info, warn, error).
func New(service string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})).
		With("service", service)
	slog.SetDefault(log)
	return log
}
