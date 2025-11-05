# GitHub Actions Workflows

This directory contains automated workflows for building, testing, and releasing Hall Monitor.

## Workflows

### `docker-publish.yml` - Docker Image Publishing

Automatically builds and publishes multi-architecture Docker images to GitHub Container Registry (GHCR).

**Triggers:**
- Push to `main` or `master` branch → publishes with `latest` tag
- Push tags matching `v*.*.*` → publishes with version tags
- Pull requests → builds but doesn't publish (for testing)
- Manual workflow dispatch

**Published Images:**
- `ghcr.io/1broseidon/hallmonitor:latest` - Latest from main branch or latest release tag
- `ghcr.io/1broseidon/hallmonitor:v0.1.0` - Specific version tags (when tag pushed)
- `ghcr.io/1broseidon/hallmonitor:0.1` - Major.minor tags (e.g., v0.1.0 → 0.1)
- `ghcr.io/1broseidon/hallmonitor:0` - Major version tags (e.g., v0.1.0 → 0)

**When you push tag `v0.1.0`, these tags are created:**
- `v0.1.0` (exact version)
- `0.1` (major.minor)
- `0` (major)
- `latest` (always points to latest release)

**Platforms:**
- `linux/amd64` - x86_64 systems
- `linux/arm64` - ARM64 systems (Raspberry Pi, AWS Graviton, etc.)

**Image is automatically:**
- Built with caching for faster builds
- Tagged with version information
- Published to GHCR for easy consumption

### `release.yml` - Release Binaries

Automatically creates GitHub releases with pre-built binaries when version tags are pushed.

**Triggers:**
- Push tags matching `v*.*.*` (e.g., `v1.2.3`)

**Built Binaries:**
- `hallmonitor-linux-amd64` - Linux x86_64
- `hallmonitor-linux-arm64` - Linux ARM64
- `hallmonitor-darwin-amd64` - macOS Intel
- `hallmonitor-darwin-arm64` - macOS Apple Silicon
- `hallmonitor-windows-amd64.exe` - Windows x86_64

## Usage

### For Users

Simply pull and run the published image:

```bash
docker pull ghcr.io/1broseidon/hallmonitor:latest
docker run -d --network host \
  -v $(pwd)/config.yml:/etc/hallmonitor/config.yml:ro \
  ghcr.io/1broseidon/hallmonitor:latest
```

Or use Docker Compose (already configured to use GHCR):

```bash
docker compose up -d
```

### For Maintainers

**To release a new version:**

1. Update version in code (if applicable)
2. Commit your changes
3. Create and push a version tag:
   ```bash
   git tag v1.2.3
   git push origin v1.2.3
   ```
4. Workflows automatically:
   - Build and publish Docker images
   - Create GitHub release with binaries
   - Generate release notes

**To trigger a rebuild of `latest`:**

Simply push to the main branch, or manually trigger the `docker-publish` workflow from the Actions tab.

## Permissions

The workflows require the following permissions:
- `contents: write` - For creating releases
- `packages: write` - For publishing to GHCR
- `id-token: write` - For Docker buildx

These are automatically provided by GitHub's `GITHUB_TOKEN`.

## Image Availability

All published images are public and can be pulled without authentication:

```bash
docker pull ghcr.io/1broseidon/hallmonitor:latest
```

For specific versions:

```bash
docker pull ghcr.io/1broseidon/hallmonitor:v1.2.3
```

