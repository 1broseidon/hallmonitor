# GitHub Actions Workflows

This directory contains the CI/CD workflows for Hall Monitor.

## Active Workflows

### CI (`ci.yml`)
**Triggers:** Push to `main`, Pull Requests

Runs on every push and PR to ensure code quality:
- **Test Job**: Runs unit tests with coverage and uploads to Codecov
- **Lint Job**: Runs staticcheck, go vet, and format checking
- **Build Job**: Verifies the binary builds successfully

### Release (`release-goreleaser.yml`)
**Triggers:** Push of version tags (`v*.*.*`)

Automated release process using GoReleaser:
1. **Test Job**: Runs full test suite with coverage
2. **Release Job**:
   - Builds binaries for multiple platforms (Linux, macOS, Windows on amd64/arm64)
   - Creates Docker images for GHCR (GitHub Container Registry)
   - Publishes multi-arch Docker manifests
   - Creates GitHub Release with changelog and assets
   - Uploads checksums

## GoReleaser Configuration

The release process is configured in `.goreleaser.yml` at the project root.

### Binaries Built
- `linux/amd64`
- `linux/arm64`
- `darwin/amd64` (macOS Intel)
- `darwin/arm64` (macOS Apple Silicon)
- `windows/amd64`

### Docker Images

Multi-architecture images are pushed to GHCR:
```
ghcr.io/1broseidon/hallmonitor:latest
ghcr.io/1broseidon/hallmonitor:v0.3.0
ghcr.io/1broseidon/hallmonitor:0.3
ghcr.io/1broseidon/hallmonitor:0
```

Supports: `linux/amd64`, `linux/arm64`

## Deprecated Workflows

The following workflows have been consolidated into GoReleaser:
- `release.yml.old` - Manual binary building (replaced by GoReleaser)
- `docker-publish.yml.old` - Separate Docker workflow (integrated into GoReleaser)

These files are kept for reference but are no longer active.

## Creating a Release

1. Ensure all tests pass on `main`
2. Update `CHANGELOG.md` with release notes
3. Create and push a version tag:
   ```bash
   git tag -a v0.4.0 -m "Release v0.4.0 - Description"
   git push origin v0.4.0
   ```
4. GitHub Actions will automatically:
   - Run tests
   - Build binaries
   - Build and push Docker images
   - Create GitHub Release
   - Upload all artifacts

## Local Testing

Test GoReleaser locally before pushing:

```bash
# Install GoReleaser
brew install goreleaser

# Test release build (doesn't publish)
goreleaser release --snapshot --clean

# Validate configuration
goreleaser check
```

## Usage

### For Users

Simply pull and run the published image:

```bash
docker pull ghcr.io/1broseidon/hallmonitor:latest
docker run -d -p 7878:7878 \
  -v $(pwd)/config.yml:/app/config.yml \
  ghcr.io/1broseidon/hallmonitor:latest
```

Or use Docker Compose:

```bash
docker compose up -d
```

### Download Binaries

Download pre-built binaries from the [Releases](https://github.com/1broseidon/hallmonitor/releases) page.

Verify checksums:
```bash
sha256sum -c checksums.txt
```

## Secrets Required

- `GITHUB_TOKEN` - Automatically provided by GitHub Actions
- `CODECOV_TOKEN` - Optional, for code coverage reporting

## Workflow Permissions

- **CI**: `contents: read`, `pull-requests: read`
- **Release**: `contents: write`, `packages: write`, `id-token: write`

## Troubleshooting

### Release fails to build Docker images
- Check QEMU and Buildx setup
- Verify Dockerfile exists and is valid
- Check GHCR permissions

### Binary build fails
- Check Go version in `go.mod`
- Verify all dependencies are available
- Check ldflags syntax in `.goreleaser.yml`

### Checksums don't match
- Ensure builds are reproducible
- Check for system-dependent code
- Verify CGO is disabled

## References

- [GoReleaser Documentation](https://goreleaser.com/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GHCR Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
