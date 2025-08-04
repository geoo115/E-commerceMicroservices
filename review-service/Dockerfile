# review-service Dockerfile
FROM golang:1.24-bullseye AS builder
ENV CGO_ENABLED=1

RUN apt-get update && apt-get install -y \
    git \
    librdkafka-dev \
    pkg-config \
    build-essential

WORKDIR /app

# Copy the message-broker source code to the correct location for replace directive
COPY ./message-broker ../message-broker

# Copy module files for review-service
COPY ./review-service/go.mod ./review-service/go.sum ./

# Install dependencies
RUN go mod download

# Copy review-service code
COPY ./review-service .

# Build
RUN go build -o review-service .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y \
    librdkafka1 \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /root/
COPY --from=builder /app/review-service .
EXPOSE 50056
CMD ["./review-service"]