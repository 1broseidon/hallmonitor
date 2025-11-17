//go:build integration
// +build integration

package storage

import (
	"os"
	"testing"
	"time"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

func getTestPostgresConnection() string {
	// Use environment variable or default
	connString := os.Getenv("POSTGRES_TEST_URL")
	if connString == "" {
		return "host=localhost port=5432 user=hallmonitor password=hallmonitor dbname=hallmonitor_test sslmode=disable"
	}
	return connString
}

func TestPostgresStore_StoreAndRetrieve(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	store, err := NewPostgresStore(getTestPostgresConnection(), logger)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer store.Close()

	// Test StoreResult
	result := &models.MonitorResult{
		Monitor:      "test-postgres-monitor",
		Type:         models.MonitorTypeHTTP,
		Status:       models.StatusUp,
		Timestamp:    time.Now(),
		ResponseTime: 150 * time.Millisecond,
		StatusCode:   200,
	}

	err = store.StoreResult(result)
	if err != nil {
		t.Fatalf("Failed to store result: %v", err)
	}

	// Test GetLatestResult
	retrieved, err := store.GetLatestResult("test-postgres-monitor")
	if err != nil {
		t.Fatalf("Failed to retrieve result: %v", err)
	}

	if retrieved.Monitor != result.Monitor {
		t.Errorf("Expected monitor %s, got %s", result.Monitor, retrieved.Monitor)
	}

	if retrieved.Status != result.Status {
		t.Errorf("Expected status %s, got %s", result.Status, retrieved.Status)
	}

	// Cleanup
	_, _ = store.pool.Exec(store.ctx, "DELETE FROM monitor_results WHERE monitor = $1", "test-postgres-monitor")
}

func TestPostgresStore_GetResults(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	store, err := NewPostgresStore(getTestPostgresConnection(), logger)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer store.Close()

	// Store multiple results
	monitorName := "test-postgres-multiple"
	now := time.Now()

	for i := 0; i < 5; i++ {
		result := &models.MonitorResult{
			Monitor:      monitorName,
			Type:         models.MonitorTypeHTTP,
			Status:       models.StatusUp,
			Timestamp:    now.Add(time.Duration(i) * time.Minute),
			ResponseTime: time.Duration(100+i*10) * time.Millisecond,
			StatusCode:   200,
		}
		if err := store.StoreResult(result); err != nil {
			t.Fatalf("Failed to store result %d: %v", i, err)
		}
	}

	// Retrieve results
	results, err := store.GetResults(monitorName, now.Add(-1*time.Hour), now.Add(1*time.Hour), 10)
	if err != nil {
		t.Fatalf("Failed to retrieve results: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	// Cleanup
	_, _ = store.pool.Exec(store.ctx, "DELETE FROM monitor_results WHERE monitor = $1", monitorName)
}

func TestPostgresStore_Aggregates(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	store, err := NewPostgresStore(getTestPostgresConnection(), logger)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer store.Close()

	// Store an aggregate
	aggregate := &models.AggregateResult{
		Monitor:         "test-postgres-aggregate",
		PeriodType:      "hourly",
		PeriodStart:     time.Now().Truncate(time.Hour),
		PeriodEnd:       time.Now().Truncate(time.Hour).Add(time.Hour),
		TotalChecks:     100,
		UpChecks:        95,
		DownChecks:      5,
		AvgResponseTime: 150 * time.Millisecond,
		MinResponseTime: 50 * time.Millisecond,
		MaxResponseTime: 300 * time.Millisecond,
	}

	if err := store.StoreAggregate(aggregate); err != nil {
		t.Fatalf("Failed to store aggregate: %v", err)
	}

	// Retrieve aggregates
	aggregates, err := store.GetAggregates(
		"test-postgres-aggregate",
		"hourly",
		time.Now().Add(-24*time.Hour),
		time.Now().Add(24*time.Hour),
	)
	if err != nil {
		t.Fatalf("Failed to retrieve aggregates: %v", err)
	}

	if len(aggregates) == 0 {
		t.Fatal("Expected at least one aggregate")
	}

	agg := aggregates[0]
	if agg.TotalChecks != 100 {
		t.Errorf("Expected 100 total checks, got %d", agg.TotalChecks)
	}

	// Cleanup
	_, _ = store.pool.Exec(store.ctx, "DELETE FROM monitor_aggregates WHERE monitor = $1", "test-postgres-aggregate")
}

func TestPostgresStore_MonitorNames(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	store, err := NewPostgresStore(getTestPostgresConnection(), logger)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer store.Close()

	// Store results for multiple monitors
	monitors := []string{"monitor-a", "monitor-b", "monitor-c"}
	for _, name := range monitors {
		result := &models.MonitorResult{
			Monitor:      name,
			Type:         models.MonitorTypeHTTP,
			Status:       models.StatusUp,
			Timestamp:    time.Now(),
			ResponseTime: 100 * time.Millisecond,
			StatusCode:   200,
		}
		if err := store.StoreResult(result); err != nil {
			t.Fatalf("Failed to store result for %s: %v", name, err)
		}
	}

	// Get monitor names
	names, err := store.GetMonitorNames()
	if err != nil {
		t.Fatalf("Failed to get monitor names: %v", err)
	}

	// Check that our monitors are in the list
	foundCount := 0
	for _, name := range names {
		for _, expected := range monitors {
			if name == expected {
				foundCount++
			}
		}
	}

	if foundCount < len(monitors) {
		t.Errorf("Expected to find at least %d monitors, found %d", len(monitors), foundCount)
	}

	// Cleanup
	for _, name := range monitors {
		_, _ = store.pool.Exec(store.ctx, "DELETE FROM monitor_results WHERE monitor = $1", name)
	}
}

func TestPostgresStore_Capabilities(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	store, err := NewPostgresStore(getTestPostgresConnection(), logger)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer store.Close()

	caps := store.Capabilities()

	if !caps.SupportsAggregation {
		t.Error("Expected SupportsAggregation to be true")
	}

	if !caps.SupportsRetention {
		t.Error("Expected SupportsRetention to be true")
	}

	if !caps.SupportsRawResults {
		t.Error("Expected SupportsRawResults to be true")
	}

	if caps.ReadOnly {
		t.Error("Expected ReadOnly to be false")
	}
}

func TestPostgresStore_HealthCheck(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	store, err := NewPostgresStore(getTestPostgresConnection(), logger)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer store.Close()

	if err := store.HealthCheck(); err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}
