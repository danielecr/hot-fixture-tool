# Multi-stage build for Hot Fixture Tool Daemon
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the main server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hfitd .

# Build the CLI tool
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hfitd-cli ./cmd/hfitd-cli

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates redis

# Create app user
RUN addgroup -g 1001 -S hfitd && \
    adduser -u 1001 -S hfitd -G hfitd

# Create necessary directories
RUN mkdir -p /app/data /tmp/hfitd && \
    chown -R hfitd:hfitd /app /tmp/hfitd

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/hfitd .
COPY --from=builder /app/hfitd-cli .

# Copy configuration files
COPY --from=builder /app/.env.example .
COPY --from=builder /app/generate_keys.sh .

# Make scripts executable
RUN chmod +x hfitd hfitd-cli generate_keys.sh

# Switch to app user
USER hfitd

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
CMD ["./hfitd"]