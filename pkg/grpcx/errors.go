package grpcx

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Internal logs err with context and returns a generic INTERNAL status, so
// implementation details never leak to callers.
func Internal(ctx context.Context, log *slog.Logger, msg string, err error) error {
	log.ErrorContext(ctx, msg, "error", err)
	return status.Error(codes.Internal, "internal error")
}
