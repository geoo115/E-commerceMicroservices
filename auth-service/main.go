// Command auth-service runs the authentication gRPC service.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/geoo115/E-commerceMicroservices/auth-service/internal/auth"
	"github.com/geoo115/E-commerceMicroservices/pkg/app"
	"github.com/geoo115/E-commerceMicroservices/pkg/cache"
	"github.com/geoo115/E-commerceMicroservices/pkg/config"
	"github.com/geoo115/E-commerceMicroservices/pkg/database"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	"github.com/geoo115/E-commerceMicroservices/pkg/logger"
	"github.com/geoo115/E-commerceMicroservices/pkg/metrics"
	authv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/auth/v1"
)

func main() {
	grpcAddr := config.String("GRPC_ADDR", ":50051")
	app.HandleHealthcheck(func() int { return grpcx.Probe("localhost" + grpcAddr) })

	log := logger.New("auth-service")
	if err := run(log, grpcAddr); err != nil {
		log.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger, grpcAddr string) error {
	ctx, stop := app.SignalContext()
	defer stop()

	secret, err := config.Require("JWT_SECRET")
	if err != nil {
		return err
	}
	tokens, err := auth.NewTokenManager(secret, config.Duration("JWT_TTL", 24*time.Hour))
	if err != nil {
		return err
	}

	startupCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	db, err := database.Open(startupCtx, database.ConfigFromEnv("auth"), log)
	if err != nil {
		return err
	}
	defer database.Close(db)
	if err := db.AutoMigrate(auth.Models...); err != nil {
		return err
	}

	rdb, err := cache.OpenFromEnv(startupCtx)
	if err != nil {
		return err
	}
	defer rdb.Close()

	if password := config.String("ADMIN_PASSWORD", ""); password != "" {
		created, err := auth.SeedAdmin(startupCtx, db,
			config.String("ADMIN_USERNAME", "admin"),
			config.String("ADMIN_EMAIL", "admin@example.com"),
			password)
		if err != nil {
			return err
		}
		if created {
			log.Info("admin account created")
		}
	}

	svc := auth.NewService(db, auth.NewRedisCodeStore(rdb), auth.LogMailer{Log: log}, tokens, log)
	server := grpcx.NewServer(log)
	authv1.RegisterAuthServiceServer(server.Registrar(), svc)

	return app.Run(ctx,
		func(ctx context.Context) error { return server.Run(ctx, grpcAddr) },
		func(ctx context.Context) error {
			return metrics.Serve(ctx, config.String("METRICS_ADDR", ":9100"), log)
		},
	)
}
