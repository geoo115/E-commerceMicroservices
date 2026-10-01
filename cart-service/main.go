// Command cart-service runs the shopping cart gRPC service.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/geoo115/E-commerceMicroservices/cart-service/internal/cart"
	"github.com/geoo115/E-commerceMicroservices/pkg/app"
	"github.com/geoo115/E-commerceMicroservices/pkg/cache"
	"github.com/geoo115/E-commerceMicroservices/pkg/config"
	"github.com/geoo115/E-commerceMicroservices/pkg/database"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	"github.com/geoo115/E-commerceMicroservices/pkg/logger"
	"github.com/geoo115/E-commerceMicroservices/pkg/metrics"
	cartv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/cart/v1"
)

func main() {
	grpcAddr := config.String("GRPC_ADDR", ":50054")
	app.HandleHealthcheck(func() int { return grpcx.Probe("localhost" + grpcAddr) })

	log := logger.New("cart-service")
	if err := run(log, grpcAddr); err != nil {
		log.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger, grpcAddr string) error {
	ctx, stop := app.SignalContext()
	defer stop()

	startupCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	db, err := database.Open(startupCtx, database.ConfigFromEnv("carts"), log)
	if err != nil {
		return err
	}
	defer database.Close(db)
	if err := db.AutoMigrate(cart.Models...); err != nil {
		return err
	}

	rdb, err := cache.OpenFromEnv(startupCtx)
	if err != nil {
		return err
	}
	defer rdb.Close()

	server := grpcx.NewServer(log)
	cartv1.RegisterCartServiceServer(server.Registrar(), cart.NewService(db, cache.Redis{Client: rdb}, log))

	return app.Run(ctx,
		func(ctx context.Context) error { return server.Run(ctx, grpcAddr) },
		func(ctx context.Context) error {
			return metrics.Serve(ctx, config.String("METRICS_ADDR", ":9100"), log)
		},
	)
}
