// Command api-gateway exposes the platform's REST API and forwards requests
// to the backend services over gRPC.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/geoo115/E-commerceMicroservices/api-gateway/internal/handler"
	"github.com/geoo115/E-commerceMicroservices/pkg/app"
	"github.com/geoo115/E-commerceMicroservices/pkg/config"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	"github.com/geoo115/E-commerceMicroservices/pkg/logger"
	authv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/auth/v1"
	cartv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/cart/v1"
	inventoryv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/inventory/v1"
	orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"
	paymentv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/payment/v1"
	productv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/product/v1"
	reviewv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/review/v1"
)

func main() {
	httpAddr := config.String("HTTP_ADDR", ":8080")
	app.HandleHealthcheck(func() int { return probe("http://localhost" + httpAddr + "/healthz") })

	log := logger.New("api-gateway")
	if err := run(log, httpAddr); err != nil {
		log.Error("gateway stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger, httpAddr string) error {
	ctx, stop := app.SignalContext()
	defer stop()

	// grpc.NewClient connects lazily, so the gateway starts even if a backend is still booting.
	var conns []*grpc.ClientConn
	var dialErr error
	dial := func(envKey, def string) *grpc.ClientConn {
		conn, err := grpcx.Dial(config.String(envKey, def))
		if err != nil {
			dialErr = errors.Join(dialErr, err)
			return nil
		}
		conns = append(conns, conn)
		return conn
	}
	defer func() {
		for _, c := range conns {
			_ = c.Close()
		}
	}()

	clients := handler.Clients{
		Auth:      authv1.NewAuthServiceClient(dial("AUTH_SERVICE_ADDR", "localhost:50051")),
		Product:   productv1.NewProductServiceClient(dial("PRODUCT_SERVICE_ADDR", "localhost:50052")),
		Order:     orderv1.NewOrderServiceClient(dial("ORDER_SERVICE_ADDR", "localhost:50053")),
		Cart:      cartv1.NewCartServiceClient(dial("CART_SERVICE_ADDR", "localhost:50054")),
		Payment:   paymentv1.NewPaymentServiceClient(dial("PAYMENT_SERVICE_ADDR", "localhost:50055")),
		Review:    reviewv1.NewReviewServiceClient(dial("REVIEW_SERVICE_ADDR", "localhost:50056")),
		Inventory: inventoryv1.NewInventoryServiceClient(dial("INVENTORY_SERVICE_ADDR", "localhost:50057")),
	}
	if dialErr != nil {
		return dialErr
	}

	gin.SetMode(gin.ReleaseMode)
	srv := &http.Server{
		Addr:              httpAddr,
		Handler:           handler.NewRouter(handler.New(clients, log), log),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("HTTP server listening", "addr", httpAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down HTTP server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func probe(url string) int {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
