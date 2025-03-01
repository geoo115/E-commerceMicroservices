# Build Stage
FROM golang:1.23.6-alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# If review-service uses message-broker for publishing events, copy the module:
COPY ../message-broker ./message-broker

# Copy source code
COPY . .

# Build the binary
RUN go build -o review-service .

# Final Stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/review-service .
EXPOSE 50056
CMD ["./review-service"]
