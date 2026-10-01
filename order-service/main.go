// Command order-service runs the order gRPC service and the order side of the saga.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/geoo115/E-commerceMicroservices/order-service/internal/order"
	"github.com/geoo115/E-commerceMicroservices/pkg/app"
	"github.com/geoo115/E-commerceMicroservices/pkg/config"
	"github.com/geoo115/E-commerceMicroservices/pkg/database"
	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	"github.com/geoo115/E-commerceMicroservices/pkg/logger"
	"github.com/geoo115/E-commerceMicroservices/pkg/metrics"
	orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"
	productv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/product/v1"
)

func main() {
	grpcAddr := config.String("GRPC_ADDR", ":50053")
	app.HandleHealthcheck(func() int { return grpcx.Probe("localhost" + grpcAddr) })

	log := logger.New("order-service")
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

	db, err := database.Open(startupCtx, database.ConfigFromEnv("orders"), log)
	if err != nil {
		return err
	}
	defer database.Close(db)
	if err := db.AutoMigrate(order.Models...); err != nil {
		return err
	}

	brokers := events.BrokersFromEnv()
	if err := events.EnsureTopics(startupCtx, brokers, log,
		events.TopicOrders, events.TopicInventory, events.TopicPayments); err != nil {
		return err
	}
	publisher := events.NewKafkaPublisher(brokers)
	defer publisher.Close()

	productConn, err := grpcx.Dial(config.String("PRODUCT_SERVICE_ADDR", "localhost:50052"))
	if err != nil {
		return err
	}
	defer productConn.Close()

	svc := order.NewService(db, productv1.NewProductServiceClient(productConn), log)
	server := grpcx.NewServer(log)
	orderv1.RegisterOrderServiceServer(server.Registrar(), svc)

	const service = "order-service"
	return app.Run(ctx,
		func(ctx context.Context) error { return server.Run(ctx, grpcAddr) },
		func(ctx context.Context) error {
			return metrics.Serve(ctx, config.String("METRICS_ADDR", ":9100"), log)
		},
		events.NewRelay(db, publisher, log).Run,
		events.NewConsumer(brokers, service, events.TopicInventory, svc.HandleInventoryEvent, publisher, log).Run,
		events.NewConsumer(brokers, service, events.TopicPayments, svc.HandlePaymentEvent, publisher, log).Run,
	)
}
