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
	// Use environment variable or default test connection
	connString := os.Getenv("POSTGRES_TEST_URL")
	if connString == "" {
		return "host=localhost port=5432 user=hallmonitor password=hallmonitor dbname=hallmonitor_test sslmode=disable"
	}
	return connString
}

func TestPostgresStore_StoreAndRetrieve(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	store, err := NewPostgresStore(getTestPostgresConnection(), 30, logger)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer store.Close()

	// Test StoreResult
	result := &models.MonitorResult{
		Monitor:   "test-postgres-monitor",
		Type:      models.MonitorTypeHTTP,
		Status:    models.StatusUp,
		Timestamp: time.Now(),
		Duration:  150 * time.Millisecond,
		HTTPResult: &models.HTTPResult{
			StatusCode:   200,
			ResponseTime: 150 * time.Millisecond,
		},
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

	if retrieved == nil {
		t.Fatal("Expected to retrieve a result, got nil")
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

	store, err := NewPostgresStore(getTestPostgresConnection(), 30, logger)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer store.Close()

	// Store multiple results
	now := time.Now()
	for i := 0; i < 5; i++ {
		result := &models.MonitorResult{
			Monitor:   "test-multi-monitor",
			Type:      models.MonitorTypeHTTP,
			Status:    models.StatusUp,
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Duration:  100 * time.Millisecond,
		}
		if err := store.StoreResult(result); err != nil {
			t.Fatalf("Failed to store result %d: %v", i, err)
		}
	}

	// Retrieve results
	results, err := store.GetResults("test-multi-monitor", now.Add(-1*time.Hour), now.Add(1*time.Hour), 10)
	if err != nil {
		t.Fatalf("Failed to get results: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	// Cleanup
	_, _ = store.pool.Exec(store.ctx, "DELETE FROM monitor_results WHERE monitor = $1", "test-multi-monitor")
}

func TestPostgresStore_Aggregates(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	store, err := NewPostgresStore(getTestPostgresConnection(), 30, logger)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer store.Close()

	// Store aggregate
	now := time.Now()
	agg := &models.AggregateResult{
		Monitor:     "test-agg-monitor",
		PeriodType:  "hour",
		PeriodStart: now.Truncate(time.Hour),
		PeriodEnd:   now.Truncate(time.Hour).Add(time.Hour),
		TotalChecks: 10,
		UpChecks:    8,
		DownChecks:  2,
		AvgDuration: 100 * time.Millisecond,
		MinDuration: 50 * time.Millisecond,
		MaxDuration: 200 * time.Millisecond,
	}

	err = store.StoreAggregate(agg)
	if err != nil {
		t.Fatalf("Failed to store aggregate: %v", err)
	}

	// Retrieve aggregates
	aggregates, err := store.GetAggregates("test-agg-monitor", "hour", now.Add(-24*time.Hour), now.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Failed to get aggregates: %v", err)
	}

	if len(aggregates) < 1 {
		t.Errorf("Expected at least 1 aggregate, got %d", len(aggregates))
	}

	// Cleanup
	_, _ = store.pool.Exec(store.ctx, "DELETE FROM monitor_aggregates WHERE monitor = $1", "test-agg-monitor")
}

func TestPostgresStore_GetMonitorNames(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	store, err := NewPostgresStore(getTestPostgresConnection(), 30, logger)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer store.Close()

	// Store results for different monitors
	monitors := []string{"monitor-a", "monitor-b", "monitor-c"}
	for _, monitor := range monitors {
		result := &models.MonitorResult{
			Monitor:   monitor,
			Type:      models.MonitorTypeHTTP,
			Status:    models.StatusUp,
			Timestamp: time.Now(),
			Duration:  100 * time.Millisecond,
		}
		if err := store.StoreResult(result); err != nil {
			t.Fatalf("Failed to store result for %s: %v", monitor, err)
		}
	}

	// Get monitor names
	names, err := store.GetMonitorNames()
	if err != nil {
		t.Fatalf("Failed to get monitor names: %v", err)
	}

	if len(names) < 3 {
		t.Errorf("Expected at least 3 monitors, got %d", len(names))
	}

	// Cleanup
	for _, monitor := range monitors {
		_, _ = store.pool.Exec(store.ctx, "DELETE FROM monitor_results WHERE monitor = $1", monitor)
	}
}
