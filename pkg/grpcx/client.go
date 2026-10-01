package grpcx

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Dial creates a lazily-connecting client connection to target. Connections
// are long-lived and multiplexed (HTTP/2), so create one per dependency at
// startup and share it.
func Dial(target string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", target, err)
	}
	return conn, nil
}

// Probe performs a gRPC health check against addr and returns a process exit
// code. Services call it from `<binary> healthcheck`, which Docker uses as the
// container health check (the runtime image has no shell or curl).
func Probe(addr string) int {
	conn, err := Dial(addr)
	if err != nil {
		return 1
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil || resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		return 1
	}
	return 0
}
