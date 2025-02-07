# Build Stage
FROM golang:1.23.6-alpine AS builder

# Install git to fetch dependencies.
RUN apk update && apk add --no-cache git

# Set the working directory inside the container.
WORKDIR /app

# Copy go.mod and go.sum to leverage Docker cache.
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code.
COPY . .

# Build the binary.
RUN go build -o review-service .

# Final Stage: Use a minimal image.
FROM alpine:latest

WORKDIR /root/

# Copy the binary from the builder stage.
COPY --from=builder /app/review-service .

# Expose the service port.
EXPOSE 50056

# Command to run the binary.
CMD ["./review-service"]
