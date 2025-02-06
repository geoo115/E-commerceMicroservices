# Build Stage
FROM golang:1.23.6-alpine AS builder

# Install git (required for module downloads)
RUN apk update && apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum to cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary
RUN go build -o cart-service .

# Final Stage: Create a minimal runtime image
FROM alpine:latest
WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /app/cart-service .

# Expose the service port (should match CART_SERVICE_PORT env variable)
EXPOSE 50054

# Command to run the service
CMD ["./cart-service"]
