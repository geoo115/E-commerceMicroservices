FROM golang:1.23.6-alpine AS builder
RUN apk update && apk add --no-cache git

WORKDIR /app

# Copy module files from service directory
COPY go.mod go.sum ./
RUN go mod download

# Copy ALL service files (no subdirectory)
COPY . .

RUN go build -o auth-service .

FROM alpine:3.20
WORKDIR /root/
COPY --from=builder /app/auth-service .
EXPOSE 50051
CMD ["./auth-service"]