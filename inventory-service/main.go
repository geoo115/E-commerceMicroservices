// Command inventory-service runs the stock gRPC service and the reservation step of the order saga.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/geoo115/E-commerceMicroservices/inventory-service/internal/inventory"
	"github.com/geoo115/E-commerceMicroservices/pkg/app"
	"github.com/geoo115/E-commerceMicroservices/pkg/config"
	"github.com/geoo115/E-commerceMicroservices/pkg/database"
	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	"github.com/geoo115/E-commerceMicroservices/pkg/logger"
	"github.com/geoo115/E-commerceMicroservices/pkg/metrics"
	inventoryv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/inventory/v1"
)

func main() {
	grpcAddr := config.String("GRPC_ADDR", ":50057")
	app.HandleHealthcheck(func() int { return grpcx.Probe("localhost" + grpcAddr) })

	log := logger.New("inventory-service")
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

	db, err := database.Open(startupCtx, database.ConfigFromEnv("inventory"), log)
	if err != nil {
		return err
	}
	defer database.Close(db)
	if err := db.AutoMigrate(inventory.Models...); err != nil {
		return err
	}

	brokers := events.BrokersFromEnv()
	if err := events.EnsureTopics(startupCtx, brokers, log,
		events.TopicInventory, events.TopicOrders, events.TopicPayments, events.TopicProducts); err != nil {
		return err
	}
	publisher := events.NewKafkaPublisher(brokers)
	defer publisher.Close()

	svc := inventory.NewService(db, log)
	server := grpcx.NewServer(log)
	inventoryv1.RegisterInventoryServiceServer(server.Registrar(), svc)

	const service = "inventory-service"
	return app.Run(ctx,
		func(ctx context.Context) error { return server.Run(ctx, grpcAddr) },
		func(ctx context.Context) error {
			return metrics.Serve(ctx, config.String("METRICS_ADDR", ":9100"), log)
		},
		events.NewRelay(db, publisher, log).Run,
		events.NewConsumer(brokers, service, events.TopicProducts, svc.HandleProductEvent, publisher, log).Run,
		events.NewConsumer(brokers, service, events.TopicOrders, svc.HandleOrderEvent, publisher, log).Run,
		events.NewConsumer(brokers, service, events.TopicPayments, svc.HandlePaymentEvent, publisher, log).Run,
	)
}
