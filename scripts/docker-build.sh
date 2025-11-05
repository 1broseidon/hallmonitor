#!/bin/bash
# Build Docker images with proper tagging and multi-arch support

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Build configuration
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
BUILD_DATE=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
VCS_REF=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

REGISTRY=${REGISTRY:-docker.io}
IMAGE_NAME=${IMAGE_NAME:-hallmonitor}
PLATFORMS=${PLATFORMS:-linux/amd64,linux/arm64}
PUSH=${PUSH:-false}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --push)
      PUSH=true
      shift
      ;;
    --registry)
      REGISTRY="$2"
      shift 2
      ;;
    --platforms)
      PLATFORMS="$2"
      shift 2
      ;;
    --version)
      VERSION="$2"
      shift 2
      ;;
    -h|--help)
      cat << EOF
Usage: $0 [OPTIONS]

Build Hall Monitor Docker images with multi-architecture support

Options:
  --push              Push images to registry after build
  --registry REGISTRY Docker registry (default: docker.io)
  --platforms ARCH    Target platforms (default: linux/amd64,linux/arm64)
  --version VERSION   Version tag (default: git describe or 'dev')
  -h, --help          Show this help message

Examples:
  $0                                    # Build locally for current platform
  $0 --push --registry ghcr.io        # Build and push to GitHub Container Registry
  $0 --platforms linux/amd64           # Build only for amd64
  $0 --version v1.0.0 --push          # Build version 1.0.0 and push

Environment Variables:
  VERSION             Version tag (can also use --version flag)
  REGISTRY            Docker registry
  PLATFORMS           Target platforms
  PUSH                Whether to push (true/false)
EOF
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown option: $1${NC}"
      exit 1
      ;;
  esac
done

echo -e "${CYAN}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║        Hall Monitor - Docker Build                       ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Build Configuration:${NC}"
echo -e "  Version:     ${GREEN}$VERSION${NC}"
echo -e "  Build Date:  ${GREEN}$BUILD_DATE${NC}"
echo -e "  Git Commit:  ${GREEN}$VCS_REF${NC}"
echo -e "  Registry:    ${GREEN}$REGISTRY${NC}"
echo -e "  Image Name:  ${GREEN}$IMAGE_NAME${NC}"
echo -e "  Platforms:   ${GREEN}$PLATFORMS${NC}"
echo -e "  Push:        ${GREEN}$PUSH${NC}"
echo ""

# Check if buildx is available
if ! docker buildx version >/dev/null 2>&1; then
  echo -e "${RED}Error: docker buildx is not available${NC}"
  echo "Please install Docker Buildx to build multi-architecture images"
  exit 1
fi

# Create or use existing buildx builder
BUILDER_NAME="hallmonitor-builder"
if ! docker buildx inspect "$BUILDER_NAME" >/dev/null 2>&1; then
  echo -e "${YELLOW}Creating new buildx builder: $BUILDER_NAME${NC}"
  docker buildx create --name "$BUILDER_NAME" --use
else
  echo -e "${YELLOW}Using existing buildx builder: $BUILDER_NAME${NC}"
  docker buildx use "$BUILDER_NAME"
fi

# Build arguments
BUILD_ARGS=(
  --build-arg "VERSION=$VERSION"
  --build-arg "BUILD_DATE=$BUILD_DATE"
  --build-arg "VCS_REF=$VCS_REF"
  --platform "$PLATFORMS"
  -t "$REGISTRY/$IMAGE_NAME:$VERSION"
  -t "$REGISTRY/$IMAGE_NAME:latest"
)

# Add push flag if requested
if [ "$PUSH" = "true" ]; then
  BUILD_ARGS+=(--push)
  echo -e "${YELLOW}Building and pushing images...${NC}"
else
  BUILD_ARGS+=(--load)
  echo -e "${YELLOW}Building images locally...${NC}"
fi

# Build the image
if docker buildx build "${BUILD_ARGS[@]}" .; then
  echo ""
  echo -e "${GREEN}╔═══════════════════════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║        ✅ Build Complete!                                 ║${NC}"
  echo -e "${GREEN}╚═══════════════════════════════════════════════════════════╝${NC}"
  echo ""
  echo -e "${CYAN}Images built:${NC}"
  echo -e "  ${GREEN}$REGISTRY/$IMAGE_NAME:$VERSION${NC}"
  echo -e "  ${GREEN}$REGISTRY/$IMAGE_NAME:latest${NC}"
  echo ""

  if [ "$PUSH" = "true" ]; then
    echo -e "${GREEN}Images pushed to registry: $REGISTRY${NC}"
  else
    echo -e "${YELLOW}To push images, run with --push flag${NC}"
    echo ""
    echo -e "${CYAN}Run the image:${NC}"
    echo -e "  ${GREEN}docker run --network host --cap-add NET_RAW --cap-add NET_ADMIN -v \$(pwd)/config.yml:/etc/hallmonitor/config.yml:ro $REGISTRY/$IMAGE_NAME:$VERSION${NC}"
  fi
else
  echo ""
  echo -e "${RED}╔═══════════════════════════════════════════════════════════╗${NC}"
  echo -e "${RED}║        ❌ Build Failed!                                   ║${NC}"
  echo -e "${RED}╚═══════════════════════════════════════════════════════════╝${NC}"
  exit 1
fi
