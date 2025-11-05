# Multi-stage build for minimal Hall Monitor image with multi-architecture support
# Build stage
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.24-alpine AS builder

# Build arguments for version information and target platform
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG BUILD_DATE
ARG VCS_REF

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata make

# Set working directory
WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary with optimizations and version info
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.gitCommit=${VCS_REF} -extldflags '-static'" \
    -a -installsuffix cgo \
    -trimpath \
    -o hallmonitor \
    cmd/server/main.go

# Verify the binary was built correctly
RUN chmod +x hallmonitor && \
    ([ "${TARGETARCH}" = "$(go env GOHOSTARCH)" ] && ./hallmonitor --version || echo "Cross-compiled for ${TARGETARCH}")

# Final stage - minimal runtime image
FROM alpine:3.19

# Metadata labels following OCI image spec
ARG VERSION=dev
ARG BUILD_DATE
ARG VCS_REF

LABEL org.opencontainers.image.title="Hall Monitor" \
      org.opencontainers.image.description="Lightweight network monitoring for home labs" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.source="https://github.com/1broseidon/hallmonitor" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.authors="Hall Monitor Team" \
      org.opencontainers.image.vendor="Hall Monitor" \
      org.opencontainers.image.documentation="https://github.com/1broseidon/hallmonitor/blob/main/README.md"

# Install runtime dependencies (minimal set for network monitoring)
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    wget \
    curl \
    libcap \
    && update-ca-certificates

# Create non-root user for security
RUN addgroup -g 1000 -S hallmonitor && \
    adduser -u 1000 -S hallmonitor -G hallmonitor

# Create necessary directories with proper permissions
RUN mkdir -p /etc/hallmonitor /etc/hallmonitor/grafana /var/lib/hallmonitor /tmp && \
    chown -R hallmonitor:hallmonitor /etc/hallmonitor /var/lib/hallmonitor /tmp

# Copy the binary from builder stage
COPY --from=builder /build/hallmonitor /usr/local/bin/hallmonitor
RUN chmod +x /usr/local/bin/hallmonitor && \
    setcap cap_net_raw+ep /usr/local/bin/hallmonitor

# Ensure config directory exists and has correct permissions
RUN chown -R hallmonitor:hallmonitor /etc/hallmonitor

# Switch to non-root user
USER hallmonitor

# Set working directory
WORKDIR /var/lib/hallmonitor

# Expose port
EXPOSE 7878

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:7878/health || exit 1

# Set default command
ENTRYPOINT ["/usr/local/bin/hallmonitor"]
CMD ["--config", "/etc/hallmonitor/config.yml"]
