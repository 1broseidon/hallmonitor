# Hall Monitor Enhanced Makefile
.PHONY: help build test clean docker k8s helm all

# ============================================================================
# Variables
# ============================================================================

APP_NAME := hallmonitor
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE) -X main.gitCommit=$(GIT_COMMIT)"

# Docker
DOCKER_REGISTRY ?= docker.io
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(APP_NAME)
PLATFORMS := linux/amd64,linux/arm64

# Kubernetes
K8S_NAMESPACE := hallmonitor
K8S_ENV ?= development

# Helm
HELM_RELEASE := hallmonitor
HELM_CHART := k8s/helm/hallmonitor

# Colors
CYAN := \033[0;36m
GREEN := \033[0;32m
RED := \033[0;31m
YELLOW := \033[1;33m
RESET := \033[0m

# ============================================================================
# Help
# ============================================================================

.DEFAULT_GOAL := help

## help: Show this help message
help:
	@echo "$(CYAN)╔═══════════════════════════════════════════════════════════╗$(RESET)"
	@echo "$(CYAN)║        Hall Monitor - Enhanced Makefile                  ║$(RESET)"
	@echo "$(CYAN)╚═══════════════════════════════════════════════════════════╝$(RESET)"
	@echo ""
	@echo "$(YELLOW)Build & Development:$(RESET)"
	@grep -E '^## build|^## test|^## clean|^## dev' $(MAKEFILE_LIST) | sed 's/^## /  /'
	@echo ""
	@echo "$(YELLOW)Code Quality:$(RESET)"
	@grep -E '^## fmt|^## imports|^## vet|^## cyclo|^## staticcheck|^## lint|^## check|^## fix' $(MAKEFILE_LIST) | sed 's/^## /  /'
	@echo ""
	@echo "$(YELLOW)Docker:$(RESET)"
	@grep -E '^## docker-' $(MAKEFILE_LIST) | sed 's/^## /  /'
	@echo ""
	@echo "$(YELLOW)Kubernetes:$(RESET)"
	@grep -E '^## k8s-' $(MAKEFILE_LIST) | sed 's/^## /  /'
	@echo ""
	@echo "$(YELLOW)Helm:$(RESET)"
	@grep -E '^## helm-' $(MAKEFILE_LIST) | sed 's/^## /  /'
	@echo ""
	@echo "$(YELLOW)Utility:$(RESET)"
	@grep -E '^## version|^## deps|^## status' $(MAKEFILE_LIST) | sed 's/^## /  /'

# ============================================================================
# Build Targets
# ============================================================================

## build: Build the Hall Monitor binary
build:
	@echo "$(CYAN)Building $(APP_NAME)...$(RESET)"
	@go build $(LDFLAGS) -o $(APP_NAME) cmd/server/main.go
	@echo "$(GREEN)✅ Build complete: ./$(APP_NAME)$(RESET)"

## build-clean: Clean build with quality checks
build-clean: check test build
	@echo "$(GREEN)✅ Clean build complete$(RESET)"

## build-linux: Build Linux binary (for Docker)
build-linux:
	@echo "$(CYAN)Building for Linux...$(RESET)"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(APP_NAME)-linux cmd/server/main.go
	@echo "$(GREEN)✅ Linux build complete$(RESET)"

## test: Run tests with coverage
test:
	@echo "$(CYAN)Running tests...$(RESET)"
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -n 1
	@echo "$(GREEN)✅ Tests passed$(RESET)"

## clean: Clean build artifacts
clean:
	@echo "$(CYAN)Cleaning...$(RESET)"
	@rm -f $(APP_NAME) $(APP_NAME)-* coverage.out
	@echo "$(GREEN)✅ Clean complete$(RESET)"

## deps: Download and tidy dependencies
deps:
	@echo "$(CYAN)Updating dependencies...$(RESET)"
	@go mod download && go mod tidy
	@echo "$(GREEN)✅ Dependencies updated$(RESET)"

## fmt: Format Go code
fmt:
	@echo "$(CYAN)Formatting code...$(RESET)"
	@go fmt ./...
	@echo "$(GREEN)✅ Code formatted$(RESET)"

## fmt-check: Check if code is formatted
fmt-check:
	@echo "$(CYAN)Checking code formatting...$(RESET)"
	@UNFORMATTED=$$(gofmt -l .); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "$(RED)❌ Code not formatted:$(RESET)"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi
	@echo "$(GREEN)✅ Code formatting OK$(RESET)"

## imports: Organize imports (requires goimports)
imports:
	@echo "$(CYAN)Organizing imports...$(RESET)"
	@command -v goimports >/dev/null 2>&1 || (echo "$(YELLOW)Installing goimports...$(RESET)" && go install golang.org/x/tools/cmd/goimports@latest)
	@goimports -w -local github.com/1broseidon/hallmonitor .
	@echo "$(GREEN)✅ Imports organized$(RESET)"

## vet: Run go vet
vet:
	@echo "$(CYAN)Running go vet...$(RESET)"
	@go vet ./...
	@echo "$(GREEN)✅ go vet passed$(RESET)"

## cyclo: Check cyclomatic complexity
cyclo:
	@echo "$(CYAN)Checking cyclomatic complexity...$(RESET)"
	@command -v gocyclo >/dev/null 2>&1 || (echo "$(YELLOW)Installing gocyclo...$(RESET)" && go install github.com/fzipp/gocyclo/cmd/gocyclo@latest)
	@gocyclo -over 15 . || echo "$(YELLOW)⚠️  Found functions with high complexity$(RESET)"
	@echo "$(GREEN)✅ Complexity check complete$(RESET)"

## staticcheck: Run staticcheck
staticcheck:
	@echo "$(CYAN)Running staticcheck...$(RESET)"
	@command -v staticcheck >/dev/null 2>&1 || (echo "$(YELLOW)Installing staticcheck...$(RESET)" && go install honnef.co/go/tools/cmd/staticcheck@latest)
	@staticcheck ./...
	@echo "$(GREEN)✅ staticcheck passed$(RESET)"

## lint: Run golangci-lint
lint:
	@echo "$(CYAN)Running golangci-lint...$(RESET)"
	@command -v golangci-lint >/dev/null 2>&1 || (echo "$(RED)golangci-lint not installed. Install from: https://golangci-lint.run/usage/install/$(RESET)" && exit 1)
	@golangci-lint run ./...
	@echo "$(GREEN)✅ Linting passed$(RESET)"

## check: Run all code quality checks
check: fmt-check vet cyclo staticcheck
	@echo "$(GREEN)✅ All code quality checks passed$(RESET)"

## fix: Auto-fix code issues
fix: fmt imports
	@echo "$(GREEN)✅ Code fixes applied$(RESET)"

# ============================================================================
# Docker Targets
# ============================================================================

## docker-build: Build Docker image (multi-arch)
docker-build:
	@echo "$(CYAN)Building Docker image...$(RESET)"
	@chmod +x scripts/docker-build.sh
	@./scripts/docker-build.sh
	@echo "$(GREEN)✅ Docker build complete$(RESET)"

## docker-push: Build and push Docker image
docker-push:
	@echo "$(CYAN)Building and pushing Docker image...$(RESET)"
	@chmod +x scripts/docker-build.sh
	@./scripts/docker-build.sh --push
	@echo "$(GREEN)✅ Docker push complete$(RESET)"

## docker-run: Run with Docker Compose (simple mode)
docker-run:
	@echo "$(CYAN)Starting Hall Monitor with Docker Compose...$(RESET)"
	@docker compose up -d
	@echo "$(GREEN)✅ Hall Monitor running at http://localhost:7878$(RESET)"


## docker-down: Stop Docker Compose stack
docker-down:
	@echo "$(CYAN)Stopping Docker stack...$(RESET)"
	@docker compose down
	@echo "$(GREEN)✅ Stopped$(RESET)"

## docker-logs: Show Docker logs
docker-logs:
	@docker compose logs -f hallmonitor

# ============================================================================
# Kubernetes Targets
# ============================================================================

## k8s-logs: Follow Kubernetes logs
k8s-logs:
	@echo "$(CYAN)Following logs...$(RESET)"
	@kubectl logs -f -l app.kubernetes.io/name=hallmonitor -n $(K8S_NAMESPACE)

## k8s-status: Check Kubernetes status
k8s-status:
	@echo "$(CYAN)Checking status...$(RESET)"
	@kubectl get all -l app.kubernetes.io/name=hallmonitor -n $(K8S_NAMESPACE)

## k8s-restart: Restart Kubernetes deployment
k8s-restart:
	@echo "$(CYAN)Restarting deployment...$(RESET)"
	@kubectl rollout restart deployment/hallmonitor -n $(K8S_NAMESPACE)
	@kubectl rollout status deployment/hallmonitor -n $(K8S_NAMESPACE)
	@echo "$(GREEN)✅ Restarted$(RESET)"

# ============================================================================
# Helm Targets
# ============================================================================

## helm-install: Install with Helm
helm-install:
	@echo "$(CYAN)Installing with Helm...$(RESET)"
	@helm install $(HELM_RELEASE) $(HELM_CHART) -n $(K8S_NAMESPACE) --create-namespace
	@echo "$(GREEN)✅ Helm installation complete$(RESET)"

## helm-upgrade: Upgrade Helm release
helm-upgrade:
	@echo "$(CYAN)Upgrading Helm release...$(RESET)"
	@helm upgrade $(HELM_RELEASE) $(HELM_CHART) -n $(K8S_NAMESPACE)
	@echo "$(GREEN)✅ Helm upgrade complete$(RESET)"

## helm-uninstall: Uninstall Helm release
helm-uninstall:
	@echo "$(CYAN)Uninstalling Helm release...$(RESET)"
	@helm uninstall $(HELM_RELEASE) -n $(K8S_NAMESPACE)
	@echo "$(GREEN)✅ Helm uninstall complete$(RESET)"

## helm-template: Show Helm template output
helm-template:
	@helm template $(HELM_RELEASE) $(HELM_CHART)

## helm-lint: Lint Helm chart
helm-lint:
	@helm lint $(HELM_CHART)

## helm-install-prod: Install with production values
helm-install-prod:
	@echo "$(CYAN)Installing with Helm (production)...$(RESET)"
	@helm install $(HELM_RELEASE) $(HELM_CHART) -n $(K8S_NAMESPACE) --create-namespace -f $(HELM_CHART)/values-production.yaml
	@echo "$(GREEN)✅ Production installation complete$(RESET)"

# ============================================================================
# Development Targets
# ============================================================================

## dev: Run in development mode
dev:
	@echo "$(CYAN)Starting in development mode...$(RESET)"
	@go run cmd/server/main.go --config config.yml

## dev-watch: Run with file watching (requires entr)
dev-watch:
	@echo "$(CYAN)Starting with file watching...$(RESET)"
	@find . -name "*.go" | entr -r go run cmd/server/main.go --config config.yml

# ============================================================================
# Utility Targets
# ============================================================================

## version: Show version information
version:
	@echo "Version:     $(VERSION)"
	@echo "Build Date:  $(BUILD_DATE)"
	@echo "Git Commit:  $(GIT_COMMIT)"

## status: Show current status (requires running instance)
status:
	@echo "$(CYAN)Checking Hall Monitor status...$(RESET)"
	@curl -sf http://localhost:7878/health && echo "$(GREEN)✅ Health: OK$(RESET)" || echo "$(RED)❌ Health: FAILED$(RESET)"
	@echo ""
	@curl -sf http://localhost:7878/api/v1/monitors | jq . 2>/dev/null || echo "$(YELLOW)API not responding$(RESET)"

## metrics: Show current metrics
metrics:
	@curl -s http://localhost:7878/metrics | grep hallmonitor || echo "$(YELLOW)Hall Monitor not running$(RESET)"

# ============================================================================
# All-in-One Targets
# ============================================================================

## all: Run all checks and build
all: clean deps check test build
	@echo "$(GREEN)✅ All checks passed$(RESET)"

## release: Create a complete release build
release: clean deps check test build docker-build
	@echo "$(GREEN)╔═══════════════════════════════════════════════════════════╗$(RESET)"
	@echo "$(GREEN)║        ✅ Release build complete!                         ║$(RESET)"
	@echo "$(GREEN)╚═══════════════════════════════════════════════════════════╝$(RESET)"
	@echo ""
	@echo "Version: $(VERSION)"
	@echo "Binary: ./$(APP_NAME)"
	@echo "Docker: $(DOCKER_IMAGE):$(VERSION)"
