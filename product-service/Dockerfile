# Use an official lightweight Go image as the base image
FROM golang:1.23.6-alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR /app

# Copy go.mod and go.sum for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN go build -o product-service .

# Use a minimal image for the final container
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/product-service .
EXPOSE 50052
CMD ["./product-service"]
