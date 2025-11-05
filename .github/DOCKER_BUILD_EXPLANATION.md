# Docker Multi-Architecture Build Explanation

## How It Works

This project uses **Docker Buildx** to create multi-architecture images from a single Dockerfile.

### Single Dockerfile for All Architectures

The `Dockerfile` supports multiple architectures through build arguments:

```dockerfile
# Build stage uses cross-compilation
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.24-alpine AS builder

# These are automatically set by Docker Buildx
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Go cross-compilation
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build ...
```

### GitHub Actions Build Process

When the workflow runs, Docker Buildx:

1. **Sets up QEMU** for ARM emulation on AMD64 runners
2. **Uses buildx** to build for multiple platforms simultaneously
3. **Automatically passes** the correct `TARGETARCH` and `TARGETOS` values
4. **Creates a multi-arch manifest** that references both images

### Supported Platforms

- `linux/amd64` - x86_64 systems (Intel/AMD)
- `linux/arm64` - ARM64 systems (Raspberry Pi 4/5, AWS Graviton, Apple Silicon in Docker)

### For Users

Users don't need to worry about architecture:

```bash
docker pull ghcr.io/1broseidon/hallmonitor:latest
```

Docker automatically pulls the correct architecture image for their system.

### Build Locally (Multi-Arch)

To build multi-arch locally:

```bash
# Create a buildx builder
docker buildx create --name hallmonitor-builder --use

# Build for multiple platforms
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/1broseidon/hallmonitor:latest \
  --push \
  .
```

### Build Locally (Single Arch)

To build for your current platform only:

```bash
docker build -t hallmonitor:latest .
```

## Why This Approach?

✅ **Single Dockerfile**: Easier to maintain
✅ **Automatic**: No manual per-architecture builds
✅ **Fast**: Buildx parallelizes builds and uses caching
✅ **Standard**: Uses official Docker best practices
✅ **User-Friendly**: Transparent to end users

## Technical Details

- **BUILDPLATFORM**: The platform doing the building (usually linux/amd64 on GitHub runners)
- **TARGETPLATFORM**: The platform being built for (linux/amd64 or linux/arm64)
- **TARGETOS**: The target OS (linux)
- **TARGETARCH**: The target architecture (amd64 or arm64)

These are automatically set by Docker Buildx during multi-platform builds.

