// Package config reads service configuration from environment variables.
// Environment variables are the only configuration surface, which keeps the
// services twelve-factor friendly and identical between Docker and local runs.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// String returns the value of key, or def when it is unset or empty.
func String(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// Int returns the integer value of key, or def when it is unset or invalid.
func Int(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// Duration returns the duration value of key (e.g. "5s"), or def when it is unset or invalid.
func Duration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// List returns the comma-separated values of key, or def when it is unset.
func List(key string, def []string) []string {
	v := String(key, "")
	if v == "" {
		return def
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Require returns the value of key or an error when it is unset or empty.
func Require(key string) (string, error) {
	v := String(key, "")
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}
