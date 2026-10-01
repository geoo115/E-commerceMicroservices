// Command review-service runs the reviews and wishlist gRPC service.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/geoo115/E-commerceMicroservices/pkg/app"
	"github.com/geoo115/E-commerceMicroservices/pkg/config"
	"github.com/geoo115/E-commerceMicroservices/pkg/database"
	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	"github.com/geoo115/E-commerceMicroservices/pkg/logger"
	"github.com/geoo115/E-commerceMicroservices/pkg/metrics"
	reviewv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/review/v1"
	"github.com/geoo115/E-commerceMicroservices/review-service/internal/review"
)

func main() {
	grpcAddr := config.String("GRPC_ADDR", ":50056")
	app.HandleHealthcheck(func() int { return grpcx.Probe("localhost" + grpcAddr) })

	log := logger.New("review-service")
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

	db, err := database.Open(startupCtx, database.ConfigFromEnv("reviews"), log)
	if err != nil {
		return err
	}
	defer database.Close(db)
	if err := db.AutoMigrate(review.Models...); err != nil {
		return err
	}

	brokers := events.BrokersFromEnv()
	if err := events.EnsureTopics(startupCtx, brokers, log, events.TopicReviews); err != nil {
		return err
	}
	publisher := events.NewKafkaPublisher(brokers)
	defer publisher.Close()

	server := grpcx.NewServer(log)
	reviewv1.RegisterReviewServiceServer(server.Registrar(), review.NewService(db, log))

	return app.Run(ctx,
		func(ctx context.Context) error { return server.Run(ctx, grpcAddr) },
		func(ctx context.Context) error {
			return metrics.Serve(ctx, config.String("METRICS_ADDR", ":9100"), log)
		},
		events.NewRelay(db, publisher, log).Run,
	)
}
