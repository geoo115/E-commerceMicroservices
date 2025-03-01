# Build Stage
FROM golang:1.23.6-alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Corrected COPY path
COPY message-broker ./message-broker
COPY payment-service ./payment-service

# Build from payment-service directory
RUN go build -o payment-service ./payment-service

# Final Stage
FROM alpine:3.20
WORKDIR /root/
COPY --from=builder /app/payment-service .
EXPOSE 50055
CMD ["./payment-service"]