# inventory-service Dockerfile
FROM golang:1.24-bullseye AS builder
ENV CGO_ENABLED=1

RUN apt-get update && apt-get install -y \
    git \
    librdkafka-dev \
    pkg-config \
    build-essential

WORKDIR /app

# Copy module files first
COPY go.mod go.sum ./

# Install dependencies
RUN go mod download

# Copy service code
COPY . .

# Build
RUN go build -o inventory-service .

FROM debian:bullseye-slim
WORKDIR /root/
COPY --from=builder /app/inventory-service .
EXPOSE 50057
CMD ["./inventory-service"]
