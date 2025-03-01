# Build Stage
FROM golang:1.23.6-alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN go build -o inventory-service .

# Final Stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/inventory-service .
EXPOSE 50057
CMD ["./inventory-service"]
