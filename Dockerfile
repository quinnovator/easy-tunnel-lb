# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o easy-tunnel-lb ./cmd

# Final stage
FROM alpine:3.19

# Install WireGuard tools and dependencies
RUN apk add --no-cache \
    wireguard-tools \
    iptables \
    ip6tables \
    iproute2

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/easy-tunnel-lb .

# Create directory for WireGuard configuration
RUN mkdir -p /etc/wireguard

# Set necessary capabilities for WireGuard and network operations
ENV CAP_NET_ADMIN=1
ENV CAP_NET_RAW=1

# Expose ports
EXPOSE 8080
EXPOSE 8081

# Run the application
ENTRYPOINT ["/app/easy-tunnel-lb"] 