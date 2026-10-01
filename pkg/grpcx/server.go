// Package grpcx contains the gRPC server and client plumbing shared by all
// services: logging, panic recovery, Prometheus metrics, health checks and
// graceful shutdown.
package grpcx

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"
	"time"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Server wraps a grpc.Server with health reporting and graceful shutdown.
type Server struct {
	grpc    *grpc.Server
	health  *health.Server
	metrics *grpcprom.ServerMetrics
	log     *slog.Logger
}

// NewServer returns a server with recovery, logging and metrics interceptors.
func NewServer(log *slog.Logger) *Server {
	metrics := grpcprom.NewServerMetrics(grpcprom.WithServerHandlingTimeHistogram())
	prometheus.MustRegister(metrics)

	recoverer := recovery.WithRecoveryHandler(func(p any) error {
		log.Error("panic in gRPC handler", "panic", p, "stack", string(debug.Stack()))
		return status.Error(codes.Internal, "internal error")
	})

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(
		metrics.UnaryServerInterceptor(),
		loggingInterceptor(log),
		recovery.UnaryServerInterceptor(recoverer),
	))

	h := health.NewServer()
	healthpb.RegisterHealthServer(srv, h)
	reflection.Register(srv)

	return &Server{grpc: srv, health: h, metrics: metrics, log: log}
}

// Registrar exposes the underlying server for service registration.
func (s *Server) Registrar() grpc.ServiceRegistrar { return s.grpc }

// Run serves on addr until ctx is cancelled, then drains in-flight RPCs.
func (s *Server) Run(ctx context.Context, addr string) error {
	s.metrics.InitializeMetrics(s.grpc)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- s.grpc.Serve(lis) }()
	s.health.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	s.log.Info("gRPC server listening", "addr", addr)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	s.log.Info("shutting down gRPC server")
	s.health.Shutdown()
	stopped := make(chan struct{})
	go func() { s.grpc.GracefulStop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		s.grpc.Stop()
	}
	return nil
}

func loggingInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err)

		level := slog.LevelInfo
		switch code {
		case codes.OK, codes.NotFound, codes.InvalidArgument, codes.AlreadyExists,
			codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition:
		default:
			level = slog.LevelError
		}
		if info.FullMethod == healthpb.Health_Check_FullMethodName {
			level = slog.LevelDebug
		}
		log.Log(ctx, level, "rpc", "method", info.FullMethod, "code", code.String(),
			"duration_ms", time.Since(start).Milliseconds(), "request_id", RequestID(ctx), "error", errString(err))
		return resp, err
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return status.Convert(err).Message()
}
