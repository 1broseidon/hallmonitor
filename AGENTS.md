# Hall Monitor - AI Agent Development Guide

This document contains project-specific guidelines for AI-assisted development, including commands, code style, and testing standards.

---

## Build & Test Commands

### Build & Test
- `make build` - Build binary with version info
- `make test` - Run all tests
- `make test-race` - Run tests with race detector
- `make test-coverage` - Generate coverage report
- `make coverage` - Run enhanced coverage script

### Single Test
- `go test -v ./internal/monitors -run TestHTTPMonitor_Check` - Run specific test
- `go test -v ./internal/monitors -run TestHTTPMonitor` - Run test by prefix

### Code Quality
- `make check` - Run fmt-check, vet, cyclo, staticcheck
- `make lint` - Run golangci-lint (requires: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
- `make fmt` - Format code with go fmt

---

## Code Style Guidelines

- **Imports**: Standard lib, external, internal packages (separate blocks)
- **Naming**: PascalCase for exported, camelCase for private, UPPERCASE for acronyms
- **Errors**: use `fmt.Errorf("%w", err)` with custom MonitorError type
- **Testing**: table-driven tests with `t.Run()`, setup helpers, interface-based mocking
- **Cyclomatic complexity**: max 15 (enforced by gocyclo)

---

## Testing Standards

### Overview

This project follows Go testing best practices with an emphasis on:
- **Fast, isolated unit tests** (70% of test suite)
- **Targeted integration tests** (20% of test suite)
- **Minimal E2E tests** (10% of test suite)
- **Coverage as a guide, not a goal** (target: 70-75%, not 80%+)

### Testing Pyramid

```
        /\
       /E2E\      <- Few tests (slow, brittle, high value)
      /------\
     /Integration\ <- Some tests (medium speed, medium value)
    /------------\
   /  Unit Tests  \ <- Most tests (fast, stable, focused)
  /----------------\
```

---

## 1. Unit Testing Rules

### ✅ DO: Test Business Logic

**Priority:** HIGH

```go
// ✅ GOOD: Testing pure business logic
func TestCalculateUptimePercentage(t *testing.T) {
    results := []models.MonitorResult{
        {Status: models.StatusUp},
        {Status: models.StatusDown},
        {Status: models.StatusUp},
        {Status: models.StatusUp},
    }

    uptime := calculateUptimePercentage(results)

    if uptime != 75.0 {
        t.Errorf("expected 75.0, got %f", uptime)
    }
}
```

### ✅ DO: Use Test Helpers (Not Reflection)

**CRITICAL:** Never use `reflect` or `unsafe` to access private fields in tests.

```go
// ❌ WRONG: Using reflection
value := reflect.ValueOf(server.scheduler).Elem()
field := value.FieldByName("resultStore")
store := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()

// ✅ RIGHT: Use test helpers
testHelper := scheduler.NewTestHelper()
testHelper.InjectResult("monitor-name", result)
```

**Pattern:** Create `testing.go` files in packages that need test-only helpers:

```go
// internal/scheduler/testing.go
package scheduler

type TestHelper struct {
    scheduler *Scheduler
}

func (s *Scheduler) NewTestHelper() *TestHelper {
    return &TestHelper{scheduler: s}
}

func (th *TestHelper) InjectResult(name string, result *models.MonitorResult) {
    th.scheduler.resultStore.StoreResult(name, result)
}
```

### ✅ DO: Table-Driven Tests

```go
func TestMonitorStatusDetermination(t *testing.T) {
    tests := []struct {
        name           string
        statusCode     int
        responseTime   time.Duration
        expectedStatus models.MonitorStatus
    }{
        {
            name:           "200 OK with fast response",
            statusCode:     200,
            responseTime:   50 * time.Millisecond,
            expectedStatus: models.StatusUp,
        },
        {
            name:           "200 OK with slow response",
            statusCode:     200,
            responseTime:   5 * time.Second,
            expectedStatus: models.StatusDegraded,
        },
        {
            name:           "500 Internal Server Error",
            statusCode:     500,
            responseTime:   100 * time.Millisecond,
            expectedStatus: models.StatusDown,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            status := determineStatus(tt.statusCode, tt.responseTime)
            if status != tt.expectedStatus {
                t.Errorf("expected %s, got %s", tt.expectedStatus, status)
            }
        })
    }
}
```

### ❌ DON'T: Test Simple Getters/Setters

```go
// ❌ DON'T TEST THIS
func (m *Monitor) GetName() string {
    return m.name
}

// ❌ DON'T WRITE THIS TEST
func TestGetName(t *testing.T) {
    monitor := &Monitor{name: "test"}
    if monitor.GetName() != "test" {
        t.Error("getter failed")
    }
}
```

### ❌ DON'T: Test Third-Party Libraries

```go
// ❌ DON'T TEST THIS
func TestZerologActuallyLogs(t *testing.T) {
    // Testing that zerolog works is not our responsibility
}

// ❌ DON'T TEST THIS
func TestFiberRoutingWorks(t *testing.T) {
    // Testing that Fiber framework routes correctly is not our responsibility
}
```

---

## 2. Integration Testing Rules

### When to Write Integration Tests

- Testing **multiple components** working together
- Testing **database interactions**
- Testing **external service interactions** (with mocks/stubs)

### Build Tags for Separation

```go
//go:build integration

package api_test

func TestFullAPIWorkflow_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Test involves server + scheduler + storage
}
```

**Run separately:**
```bash
# Unit tests only (fast)
go test ./...

# Include integration tests
go test -tags=integration ./...

# Skip slow tests
go test -short ./...
```

### File Organization

```
internal/monitors/
├── dns.go                    # Production code
├── dns_test.go              # Unit tests (fast, mocked)
├── dns_integration_test.go  # Integration tests (//go:build integration)
└── testdata/                # Test fixtures
    └── dns_responses.json
```

---

## 3. External Dependencies

### ❌ NEVER: Make Real External Calls in Unit Tests

```go
// ❌ BAD: Calls real DNS server
func TestDNSMonitor(t *testing.T) {
    monitor := &DNSMonitor{
        resolver: net.DefaultResolver, // Real DNS!
        target:   "8.8.8.8",          // Real Google DNS!
    }

    result, _ := monitor.Check(context.Background())
    // This test will fail if internet is down
}
```

### ✅ DO: Mock External Dependencies

**Step 1:** Define interfaces for external dependencies

```go
// internal/monitors/interfaces.go
type DNSResolver interface {
    LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
    LookupCNAME(ctx context.Context, host string) (string, error)
    LookupMX(ctx context.Context, name string) ([]*net.MX, error)
}

type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}
```

**Step 2:** Use interfaces in production code

```go
type DNSMonitor struct {
    *BaseMonitor
    resolver DNSResolver // Interface, not concrete type
}

func NewDNSMonitor(...) *DNSMonitor {
    return &DNSMonitor{
        resolver: &NetResolver{Resolver: net.DefaultResolver}, // Wrap in adapter
    }
}
```

**Step 3:** Mock in tests

```go
type MockDNSResolver struct {
    LookupIPAddrFunc func(ctx context.Context, host string) ([]net.IPAddr, error)
}

func (m *MockDNSResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
    return m.LookupIPAddrFunc(ctx, host)
}

func TestDNSMonitor_Success(t *testing.T) {
    mockResolver := &MockDNSResolver{
        LookupIPAddrFunc: func(ctx context.Context, host string) ([]net.IPAddr, error) {
            return []net.IPAddr{{IP: net.ParseIP("1.2.3.4")}}, nil
        },
    }

    monitor := &DNSMonitor{resolver: mockResolver}
    result, err := monitor.Check(context.Background())

    // Fast, deterministic, no network required
}
```

**Real network calls belong in integration tests only:**

```go
//go:build integration

func TestDNSMonitor_RealNetwork_Integration(t *testing.T) {
    monitor := &DNSMonitor{
        resolver: &NetResolver{Resolver: net.DefaultResolver},
    }
    // Only runs when explicitly requested
}
```

---

## 4. Coverage Guidelines

### Philosophy

**Coverage is a tool to find untested critical paths, NOT a goal.**

### Target Coverage by Package

```yaml
# Realistic targets
internal/monitors:   75-80%  # Critical business logic
internal/scheduler:  70-75%  # Core functionality
internal/api:        60-70%  # HTTP handlers (harder to test)
internal/storage:    65-75%  # Data layer
internal/logging:    50-60%  # Wrapper around zerolog
internal/metrics:    80-95%  # Wrapper around prometheus
cmd/server:          0-10%   # Main packages (test with E2E)

# Overall project: 70-75%
```

### What Coverage Doesn't Mean

- ❌ 80% coverage ≠ 80% bug-free
- ❌ 100% coverage ≠ perfect tests
- ✅ 70% coverage with good tests > 95% coverage with bad tests

### Use Coverage to Find Gaps

```bash
# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Look for:
# - Uncovered error paths
# - Uncovered edge cases
# - Uncovered critical business logic
```

---

## 5. Test Organization

### File Naming

```
✅ monitor.go        → monitor_test.go         (unit tests)
✅ monitor.go        → monitor_integration_test.go (integration)
✅ (package)         → testing.go              (test helpers)
✅ (package)         → testdata/               (fixtures)
✅ (package)         → mocks/                  (mock implementations)
```

### Package Naming

```go
// ✅ PREFER: Same package (access to internals when needed)
package monitors

func TestValidateConfig(t *testing.T) {
    // Can access private helpers for unit testing
}

// ✅ ACCEPTABLE: Black-box testing
package monitors_test

func TestPublicAPI(t *testing.T) {
    // Only tests public API
}
```

### Test Helper Pattern

```go
// testhelpers.go or testing.go
package monitors

func setupTestMonitor(t *testing.T) *HTTPMonitor {
    t.Helper()

    logger, _ := logging.InitLogger(logging.Config{
        Level:  "error",
        Format: "json",
    })

    return &HTTPMonitor{
        BaseMonitor: NewBaseMonitor(&models.Monitor{
            Name:     "test",
            Type:     "http",
            Interval: 30 * time.Second,
        }, "test-group", logger, nil),
    }
}
```

---

## 6. Testing Anti-Patterns to Avoid

### ❌ Coverage-Driven Development

```go
// ❌ BAD: Writing tests just to hit coverage number
func TestEveryGetter(t *testing.T) {
    m := &Monitor{name: "test"}
    _ = m.GetName()     // No assertions
    _ = m.GetType()     // No value
    _ = m.GetGroup()    // Just hitting coverage
}
```

### ❌ Testing Implementation Details

```go
// ❌ BAD: Testing internal data structures
func TestInternalCache(t *testing.T) {
    monitor := NewMonitor()
    // Don't test that cache is a map
    if len(monitor.internalCache) != 0 {
        t.Error("cache should be empty")
    }
}

// ✅ GOOD: Testing behavior
func TestMonitorCachesResults(t *testing.T) {
    monitor := NewMonitor()

    // First call - slow
    result1, _ := monitor.Check()

    // Second call - should use cache
    result2, _ := monitor.Check()

    // Test the behavior, not the implementation
    if result1.Duration < result2.Duration {
        t.Error("expected second call to be faster (cached)")
    }
}
```

### ❌ Flaky Tests

```go
// ❌ BAD: Race conditions
func TestConcurrentAccess(t *testing.T) {
    counter := 0
    go func() { counter++ }()
    go func() { counter++ }()
    time.Sleep(100 * time.Millisecond) // Flaky!
    if counter != 2 {
        t.Error("expected 2")
    }
}

// ✅ GOOD: Proper synchronization
func TestConcurrentAccess(t *testing.T) {
    counter := 0
    var wg sync.WaitGroup
    var mu sync.Mutex

    for i := 0; i < 2; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            mu.Lock()
            counter++
            mu.Unlock()
        }()
    }

    wg.Wait()
    if counter != 2 {
        t.Error("expected 2")
    }
}
```

### ❌ God Tests (Too Much in One Test)

```go
// ❌ BAD: Testing everything in one test
func TestMonitorEverything(t *testing.T) {
    // Creates monitor
    // Validates config
    // Runs check
    // Stores result
    // Calculates uptime
    // Sends alert
    // ... 200 lines later
}

// ✅ GOOD: Focused tests
func TestMonitorCreation(t *testing.T) { /* ... */ }
func TestMonitorValidation(t *testing.T) { /* ... */ }
func TestMonitorCheck(t *testing.T) { /* ... */ }
```

---

## 7. Makefile Commands

Create test commands in your Makefile:

```makefile
.PHONY: test test-unit test-integration test-coverage test-race

# Fast unit tests
test-unit:
	@echo "Running unit tests..."
	@go test -short -race ./...

# Integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -tags=integration -race ./...

# All tests
test:
	@echo "Running all tests..."
	@go test -race ./...

# Coverage report
test-coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total
	@echo "Open coverage.html to view detailed coverage"

# Race detection (run critical packages many times)
test-race:
	@echo "Running race detection..."
	@go test -race -count=100 ./internal/scheduler/...
	@go test -race -count=100 ./internal/storage/...
```

---

## 8. Quick Reference

### When to Write Tests

| Scenario | Test Type | Priority |
|----------|-----------|----------|
| Business logic (calculations, rules) | Unit | HIGH |
| Error handling | Unit | HIGH |
| Edge cases (nil, empty, max) | Unit | HIGH |
| Config validation | Unit | MEDIUM |
| API handlers (simple) | Integration | MEDIUM |
| Database interactions | Integration | MEDIUM |
| Full workflows | E2E | LOW |
| Getters/setters | None | SKIP |
| Framework functionality | None | SKIP |

### Test Speed Goals

- Unit test: < 10ms
- Integration test: < 100ms
- E2E test: < 5s
- Full suite: < 30s

### Red Flags in Tests

🚩 Using `reflect` or `unsafe`
🚩 `time.Sleep()` for synchronization
🚩 Tests that fail intermittently
🚩 Tests that depend on external services
🚩 Tests that take >1 second
🚩 Tests with no assertions
🚩 100+ line test functions

---

## 9. Resources

- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Testing with Interfaces](https://www.youtube.com/watch?v=8hQG7QlcLBk)
- [Test Doubles (Mocks, Stubs, Fakes)](https://martinfowler.com/bliki/TestDouble.html)

---

**Last Updated:** 2025-11-16
**Project Coverage Target:** 70-75%
**Current Coverage:** 70.7% ✅
