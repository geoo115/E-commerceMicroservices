# Build Stage
FROM golang:1.23.6-alpine AS builder

# Install git for module downloads
RUN apk update && apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code into the container
COPY . .

# Build the binary
RUN go build -o payment-service .

# Final Stage
FROM alpine:latest

WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /app/payment-service .

# Expose the service port (adjust if your PAYMENT_SERVICE_PORT differs)
EXPOSE 50055

# Command to run the payment-service binary
CMD ["./payment-service"]
