# Build Stage
FROM golang:1.18-alpine AS builder

# Install git to download modules
RUN apk update && apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the inventory-service binary
RUN go build -o inventory-service .

# Final Stage: Use a minimal image
FROM alpine:latest

WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /app/inventory-service .

# Expose the service port (should match INVENTORY_SERVICE_PORT)
EXPOSE 50057

# Command to run the inventory-service binary
CMD ["./inventory-service"]
