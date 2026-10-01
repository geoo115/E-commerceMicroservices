// Command product-service runs the catalog gRPC service.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/geoo115/E-commerceMicroservices/pkg/app"
	"github.com/geoo115/E-commerceMicroservices/pkg/cache"
	"github.com/geoo115/E-commerceMicroservices/pkg/config"
	"github.com/geoo115/E-commerceMicroservices/pkg/database"
	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	"github.com/geoo115/E-commerceMicroservices/pkg/logger"
	"github.com/geoo115/E-commerceMicroservices/pkg/metrics"
	productv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/product/v1"
	"github.com/geoo115/E-commerceMicroservices/product-service/internal/product"
)

func main() {
	grpcAddr := config.String("GRPC_ADDR", ":50052")
	app.HandleHealthcheck(func() int { return grpcx.Probe("localhost" + grpcAddr) })

	log := logger.New("product-service")
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

	db, err := database.Open(startupCtx, database.ConfigFromEnv("products"), log)
	if err != nil {
		return err
	}
	defer database.Close(db)
	if err := db.AutoMigrate(product.Models...); err != nil {
		return err
	}

	rdb, err := cache.OpenFromEnv(startupCtx)
	if err != nil {
		return err
	}
	defer rdb.Close()

	brokers := events.BrokersFromEnv()
	if err := events.EnsureTopics(startupCtx, brokers, log, events.TopicProducts, events.TopicReviews); err != nil {
		return err
	}
	publisher := events.NewKafkaPublisher(brokers)
	defer publisher.Close()

	svc := product.NewService(db, cache.Redis{Client: rdb}, log)
	server := grpcx.NewServer(log)
	productv1.RegisterProductServiceServer(server.Registrar(), svc)

	reviews := events.NewConsumer(brokers, "product-service", events.TopicReviews, svc.HandleReviewEvent, publisher, log)

	return app.Run(ctx,
		func(ctx context.Context) error { return server.Run(ctx, grpcAddr) },
		func(ctx context.Context) error {
			return metrics.Serve(ctx, config.String("METRICS_ADDR", ":9100"), log)
		},
		events.NewRelay(db, publisher, log).Run,
		reviews.Run,
	)
}
