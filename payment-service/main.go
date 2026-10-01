// Command payment-service runs the payment gRPC service.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/geoo115/E-commerceMicroservices/payment-service/internal/payment"
	"github.com/geoo115/E-commerceMicroservices/pkg/app"
	"github.com/geoo115/E-commerceMicroservices/pkg/config"
	"github.com/geoo115/E-commerceMicroservices/pkg/database"
	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	"github.com/geoo115/E-commerceMicroservices/pkg/logger"
	"github.com/geoo115/E-commerceMicroservices/pkg/metrics"
	orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"
	paymentv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/payment/v1"
)

func main() {
	grpcAddr := config.String("GRPC_ADDR", ":50055")
	app.HandleHealthcheck(func() int { return grpcx.Probe("localhost" + grpcAddr) })

	log := logger.New("payment-service")
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

	db, err := database.Open(startupCtx, database.ConfigFromEnv("payments"), log)
	if err != nil {
		return err
	}
	defer database.Close(db)
	if err := payment.Migrate(db); err != nil {
		return err
	}

	brokers := events.BrokersFromEnv()
	if err := events.EnsureTopics(startupCtx, brokers, log, events.TopicPayments); err != nil {
		return err
	}
	publisher := events.NewKafkaPublisher(brokers)
	defer publisher.Close()

	orderConn, err := grpcx.Dial(config.String("ORDER_SERVICE_ADDR", "localhost:50053"))
	if err != nil {
		return err
	}
	defer orderConn.Close()

	svc := payment.NewService(db, orderv1.NewOrderServiceClient(orderConn), payment.SimulatedProvider{},
		config.String("PAYMENT_CURRENCY", "USD"), log)
	server := grpcx.NewServer(log)
	paymentv1.RegisterPaymentServiceServer(server.Registrar(), svc)

	return app.Run(ctx,
		func(ctx context.Context) error { return server.Run(ctx, grpcAddr) },
		func(ctx context.Context) error {
			return metrics.Serve(ctx, config.String("METRICS_ADDR", ":9100"), log)
		},
		events.NewRelay(db, publisher, log).Run,
	)
}
