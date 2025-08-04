FROM golang:1.24-bullseye AS builder
ENV CGO_ENABLED=1

RUN apt-get update && apt-get install -y \
    git \
    librdkafka-dev \
    pkg-config \
    build-essential

WORKDIR /app

# Copy the message-broker directory to the correct location for replace directive
COPY ./message-broker ../message-broker

# Copy payment-service go.mod and go.sum first
COPY ./payment-service/go.mod ./payment-service/go.sum ./

# Download dependencies with the local replacement available
RUN go mod download

# Copy the entire payment-service directory to /app
COPY ./payment-service .

# Build
RUN go build -o payment-service .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y \
    librdkafka1 \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /root/
COPY --from=builder /app/payment-service .
EXPOSE 50055
CMD ["./payment-service"]