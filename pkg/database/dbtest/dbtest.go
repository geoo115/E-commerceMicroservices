// Package dbtest provides isolated PostgreSQL databases for integration tests.
//
// Tests are skipped unless TEST_DATABASE_URL is set, so `go test ./...` passes
// on a clean checkout. Each call gets its own schema, dropped on cleanup.
package dbtest

import (
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// New returns a *gorm.DB bound to a fresh schema with the given models migrated.
func New(t testing.TB, models ...any) *gorm.DB {
	t.Helper()

	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	admin, err := gorm.Open(postgres.Open(raw), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}

	schema := "test_" + randomSuffix(t)
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()

	db, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: gormlogger.Discard, TranslateError: true})
	if err != nil {
		t.Fatalf("connect to test schema: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error
		if sqlDB, err := admin.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func randomSuffix(t testing.TB) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random: %v", err)
	}
	return hex.EncodeToString(b)
}
