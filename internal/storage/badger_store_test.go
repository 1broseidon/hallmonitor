package storage

import (
	"os"
	"testing"
	"time"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

func createTestStore(t *testing.T) (*BadgerStore, string) {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "hallmonitor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create logger
	logger, err := logging.InitLogger(logging.Config{
		Level:  "error",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create store
	store, err := NewBadgerStore(tmpDir, 7, logger)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create store: %v", err)
	}

	return store, tmpDir
}

func TestBadgerStore_StoreAndRetrieveResult(t *testing.T) {
	store, tmpDir := createTestStore(t)
	defer func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}()

	// Create test result
	result := &models.MonitorResult{
		Monitor:   "test-monitor",
		Type:      models.MonitorTypeHTTP,
		Group:     "test-group",
		Status:    models.StatusUp,
		Duration:  100 * time.Millisecond,
		Timestamp: time.Now(),
	}

	// Store result
	err := store.StoreResult(result)
	if err != nil {
		t.Fatalf("Failed to store result: %v", err)
	}

	// Retrieve latest result
	retrieved, err := store.GetLatestResult("test-monitor")
	if err != nil {
		t.Fatalf("Failed to get latest result: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved result is nil")
	}

	// Verify result
	if retrieved.Monitor != result.Monitor {
		t.Errorf("Expected monitor %s, got %s", result.Monitor, retrieved.Monitor)
	}
	if retrieved.Status != result.Status {
		t.Errorf("Expected status %s, got %s", result.Status, retrieved.Status)
	}
}

func TestBadgerStore_GetResults(t *testing.T) {
	store, tmpDir := createTestStore(t)
	defer func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}()

	// Store multiple results
	now := time.Now()
	for i := 0; i < 10; i++ {
		result := &models.MonitorResult{
			Monitor:   "test-monitor",
			Type:      models.MonitorTypeHTTP,
			Group:     "test-group",
			Status:    models.StatusUp,
			Duration:  100 * time.Millisecond,
			Timestamp: now.Add(time.Duration(i) * time.Second),
		}
		if err := store.StoreResult(result); err != nil {
			t.Fatalf("Failed to store result %d: %v", i, err)
		}
	}

	// Retrieve results in time range
	start := now.Add(-1 * time.Second)
	end := now.Add(15 * time.Second)

	results, err := store.GetResults("test-monitor", start, end, 100)
	if err != nil {
		t.Fatalf("Failed to get results: %v", err)
	}

	if len(results) != 10 {
		t.Errorf("Expected 10 results, got %d", len(results))
	}
}

func TestBadgerStore_StoreAndRetrieveAggregate(t *testing.T) {
	store, tmpDir := createTestStore(t)
	defer func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}()

	// Create test aggregate
	now := time.Now().Truncate(time.Hour)
	agg := &models.AggregateResult{
		Monitor:       "test-monitor",
		PeriodStart:   now,
		PeriodEnd:     now.Add(time.Hour),
		PeriodType:    "hour",
		TotalChecks:   100,
		UpChecks:      98,
		DownChecks:    2,
		UptimePercent: 98.0,
		AvgDuration:   50 * time.Millisecond,
		MinDuration:   10 * time.Millisecond,
		MaxDuration:   200 * time.Millisecond,
	}

	// Store aggregate
	err := store.StoreAggregate(agg)
	if err != nil {
		t.Fatalf("Failed to store aggregate: %v", err)
	}

	// Retrieve aggregates
	start := now.Add(-1 * time.Hour)
	end := now.Add(2 * time.Hour)

	aggregates, err := store.GetAggregates("test-monitor", "hour", start, end)
	if err != nil {
		t.Fatalf("Failed to get aggregates: %v", err)
	}

	if len(aggregates) != 1 {
		t.Errorf("Expected 1 aggregate, got %d", len(aggregates))
	}

	if len(aggregates) > 0 {
		retrieved := aggregates[0]
		if retrieved.TotalChecks != agg.TotalChecks {
			t.Errorf("Expected %d total checks, got %d", agg.TotalChecks, retrieved.TotalChecks)
		}
		if retrieved.UptimePercent != agg.UptimePercent {
			t.Errorf("Expected %.2f uptime, got %.2f", agg.UptimePercent, retrieved.UptimePercent)
		}
	}
}

func TestBadgerStore_GetMonitorNames(t *testing.T) {
	store, tmpDir := createTestStore(t)
	defer func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}()

	// Store results for multiple monitors
	monitors := []string{"monitor-1", "monitor-2", "monitor-3"}
	now := time.Now()

	for _, monitorName := range monitors {
		result := &models.MonitorResult{
			Monitor:   monitorName,
			Type:      models.MonitorTypeHTTP,
			Group:     "test-group",
			Status:    models.StatusUp,
			Duration:  100 * time.Millisecond,
			Timestamp: now,
		}
		if err := store.StoreResult(result); err != nil {
			t.Fatalf("Failed to store result for %s: %v", monitorName, err)
		}
	}

	// Get monitor names
	names, err := store.GetMonitorNames()
	if err != nil {
		t.Fatalf("Failed to get monitor names: %v", err)
	}

	if len(names) != len(monitors) {
		t.Errorf("Expected %d monitor names, got %d", len(monitors), len(names))
	}
}

func TestBadgerStore_MetadataOperations(t *testing.T) {
	store, tmpDir := createTestStore(t)
	defer func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}()

	// Set metadata
	key := "test-key"
	value := []byte("test-value")

	err := store.SetMetadata(key, value)
	if err != nil {
		t.Fatalf("Failed to set metadata: %v", err)
	}

	// Get metadata
	retrieved, err := store.GetMetadata(key)
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}

	if string(retrieved) != string(value) {
		t.Errorf("Expected metadata %s, got %s", string(value), string(retrieved))
	}

	// Get non-existent metadata
	retrieved, err = store.GetMetadata("non-existent")
	if err != nil {
		t.Fatalf("Failed to get non-existent metadata: %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil for non-existent metadata, got %v", retrieved)
	}
}

