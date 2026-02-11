# 1. Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy Go modules first to leverage caching
COPY go.mod go.sum ./
RUN go mod download

# Copy all project files
COPY . .

# Build the Go binary
RUN go build -o hci-asset ./cmd/api

# 2. Run stage (smaller final image)
FROM alpine:latest

RUN apk add --no-cache nmap

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/hci-asset .

# Expose the port your API listens on
EXPOSE 8080

# Default command to run your API
CMD ["./hci-asset"]
