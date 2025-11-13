# Hall Monitor Development Commands

## Build & Test
- `make build` - Build binary with version info
- `make test` - Run all tests
- `make test-race` - Run tests with race detector
- `make test-coverage` - Generate coverage report
- `make coverage` - Run enhanced coverage script

## Single Test
- `go test -v ./internal/monitors -run TestHTTPMonitor_Check` - Run specific test
- `go test -v ./internal/monitors -run TestHTTPMonitor` - Run test by prefix

## Code Quality
- `make check` - Run fmt-check, vet, cyclo, staticcheck
- `make lint` - Run golangci-lint (requires: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
- `make fmt` - Format code with go fmt

## Code Style
- **Imports**: Standard lib, external, internal packages (separate blocks)
- **Naming**: PascalCase for exported, camelCase for private, UPPERCASE for acronyms
- **Errors**: use `fmt.Errorf("%w", err)` with custom MonitorError type
- **Testing**: table-driven tests with `t.Run()`, setup helpers, interface-based mocking
- **Cyclomatic complexity**: max 15 (enforced by gocyclo)