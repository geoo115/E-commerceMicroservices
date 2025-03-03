FROM golang:1.24-bullseye AS builder
ENV CGO_ENABLED=1

RUN apt-get update && apt-get install -y \
    git \
    librdkafka-dev \
    pkg-config \
    build-essential

WORKDIR /app

# Copy the message-broker directory from build context to /message-broker in the container
COPY message-broker /message-broker

# Copy the entire payment-service directory to /app
COPY payment-service/ /app/

# Download dependencies with the local replacement available
RUN go mod download

# Build
RUN go build -o payment-service .

FROM debian:bullseye-slim
WORKDIR /root/
COPY --from=builder /app/payment-service .
EXPOSE 50054
CMD ["./payment-service"]