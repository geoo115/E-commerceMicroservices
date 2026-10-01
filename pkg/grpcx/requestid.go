package grpcx

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// RequestID returns the x-request-id forwarded by the API gateway, if any.
func RequestID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-request-id"); len(v) > 0 {
			return v[0]
		}
	}
	return ""
}
