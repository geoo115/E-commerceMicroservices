# Use Debian-based image for better compatibility with C libraries
FROM golang:1.23-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    librdkafka-dev \
    gcc \
    g++ \
    make

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o message-broker .

# Final stage
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y \
    librdkafka-dev \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /root/
COPY --from=builder /app/message-broker .
CMD ["./message-broker"]