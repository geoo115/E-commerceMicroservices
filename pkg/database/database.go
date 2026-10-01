// Package database opens PostgreSQL connections through GORM.
package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/geoo115/E-commerceMicroservices/pkg/config"
)

// Config holds PostgreSQL connection settings.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// ConfigFromEnv reads DATABASE_* variables. defaultName is used when DATABASE_NAME is unset.
func ConfigFromEnv(defaultName string) Config {
	return Config{
		Host:            config.String("DATABASE_HOST", "localhost"),
		Port:            config.String("DATABASE_PORT", "5432"),
		User:            config.String("DATABASE_USER", "postgres"),
		Password:        config.String("DATABASE_PASSWORD", "postgres"),
		Name:            config.String("DATABASE_NAME", defaultName),
		SSLMode:         config.String("DATABASE_SSLMODE", "disable"),
		MaxOpenConns:    config.Int("DATABASE_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    config.Int("DATABASE_MAX_IDLE_CONNS", 10),
		ConnMaxLifetime: config.Duration("DATABASE_CONN_MAX_LIFETIME", 5*time.Minute),
	}
}

// DSN returns the libpq-style connection string.
func (c Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

// Open connects to PostgreSQL, retrying until ctx is done so that services
// tolerate the database starting after them.
func Open(ctx context.Context, cfg Config, log *slog.Logger) (*gorm.DB, error) {
	gcfg := &gorm.Config{
		Logger: gormlogger.New(slogWriter{log}, gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true, // "not found" is a normal outcome, not an error
		}),
		TranslateError: true, // map unique violations to gorm.ErrDuplicatedKey
	}

	var lastErr error
	for {
		db, err := gorm.Open(postgres.Open(cfg.DSN()), gcfg)
		if err == nil {
			sqlDB, err := db.DB()
			if err != nil {
				return nil, err
			}
			sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
			sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
			sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
			if err = sqlDB.PingContext(ctx); err == nil {
				log.Info("connected to database", "host", cfg.Host, "database", cfg.Name)
				return db, nil
			}
		}
		lastErr = err
		log.Warn("database not ready, retrying", "error", err)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connect to database: %w", lastErr)
		case <-time.After(2 * time.Second):
		}
	}
}

// slogWriter adapts slog to gorm's logger.Writer (slow queries, SQL errors).
type slogWriter struct{ log *slog.Logger }

func (w slogWriter) Printf(format string, args ...any) {
	w.log.Warn("gorm", "detail", fmt.Sprintf(format, args...))
}

// Close closes the underlying connection pool.
func Close(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
