# GoReleaser v2 compatible Dockerfile
FROM alpine:latest

LABEL org.opencontainers.image.authors="daniele@smartango.com"

# Install runtime dependencies
RUN apk --no-cache add ca-certificates redis

# Create app user
RUN addgroup -g 1001 -S hfitd && \
    adduser -u 1001 -S hfitd -G hfitd

# Create necessary directories
RUN mkdir -p /app/data /tmp/hfitd && \
    chown -R hfitd:hfitd /app /tmp/hfitd

WORKDIR /app

# Copy pre-built binaries from GoReleaser build context
# GoReleaser v2 provides binaries in $TARGETPLATFORM directory
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/hfitd .
COPY $TARGETPLATFORM/hfitd-cli .

# Copy configuration files (add to extra_files in goreleaser if needed)
COPY hfitd/.env.example .env.example

# Make scripts executable
RUN chmod +x hfitd hfitd-cli

# Switch to app user
USER hfitd

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
CMD ["./hfitd"]