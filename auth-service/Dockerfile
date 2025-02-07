# Use an official lightweight Go image as the base image
FROM golang:1.23.6-alpine AS builder

# Install git (required for go modules) and other necessary packages
RUN apk update && apk add --no-cache git

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the workspace
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -o auth-service .

# Use a smaller base image for production
FROM alpine:latest

# Set the working directory
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/auth-service .

# Expose the port (make sure this matches AUTH_SERVICE_PORT in your env)
EXPOSE 50051

# Command to run the executable
CMD ["./auth-service"]
