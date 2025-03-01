# Use Debian-based image for builder
FROM golang:1.23-bookworm AS builder

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

# Final stage with runtime library
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y \
    librdkafka1 \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /root/
COPY --from=builder /app/message-broker .
CMD ["./message-broker"]