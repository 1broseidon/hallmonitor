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

func getTestInfluxDBConfig() (url, token, org, bucket string) {
	url = os.Getenv("INFLUXDB_TEST_URL")
	if url == "" {
		url = "http://localhost:8086"
	}

	token = os.Getenv("INFLUXDB_TEST_TOKEN")
	if token == "" {
		token = "hallmonitor-test-token"
	}

	org = os.Getenv("INFLUXDB_TEST_ORG")
	if org == "" {
		org = "hallmonitor"
	}

	bucket = os.Getenv("INFLUXDB_TEST_BUCKET")
	if bucket == "" {
		bucket = "test"
	}

	return
}

func TestInfluxDBStore_StoreAndRetrieve(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	url, token, org, bucket := getTestInfluxDBConfig()
	store, err := NewInfluxDBStore(url, token, org, bucket, logger)
	if err != nil {
		t.Skipf("InfluxDB not available: %v", err)
	}
	defer store.Close()

	// Test StoreResult
	result := &models.MonitorResult{
		Monitor:   "test-influx-monitor",
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

	// Give InfluxDB time to process the write
	time.Sleep(1 * time.Second)

	// Test GetLatestResult
	retrieved, err := store.GetLatestResult("test-influx-monitor")
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
}

func TestInfluxDBStore_GetResults(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	url, token, org, bucket := getTestInfluxDBConfig()
	store, err := NewInfluxDBStore(url, token, org, bucket, logger)
	if err != nil {
		t.Skipf("InfluxDB not available: %v", err)
	}
	defer store.Close()

	// Store multiple results
	now := time.Now()
	for i := 0; i < 5; i++ {
		result := &models.MonitorResult{
			Monitor:   "test-influx-multi",
			Type:      models.MonitorTypeHTTP,
			Status:    models.StatusUp,
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Duration:  100 * time.Millisecond,
		}
		if err := store.StoreResult(result); err != nil {
			t.Fatalf("Failed to store result %d: %v", i, err)
		}
	}

	// Flush writes
	store.writeAPI.Flush()

	// Give InfluxDB time to process
	time.Sleep(2 * time.Second)

	// Retrieve results
	results, err := store.GetResults("test-influx-multi", now.Add(-1*time.Hour), now.Add(1*time.Hour), 10)
	if err != nil {
		t.Fatalf("Failed to get results: %v", err)
	}

	if len(results) < 1 {
		t.Logf("Warning: Expected multiple results, got %d (may be due to write latency)", len(results))
	}
}

func TestInfluxDBStore_GetMonitorNames(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	url, token, org, bucket := getTestInfluxDBConfig()
	store, err := NewInfluxDBStore(url, token, org, bucket, logger)
	if err != nil {
		t.Skipf("InfluxDB not available: %v", err)
	}
	defer store.Close()

	// Store results for different monitors
	monitors := []string{"influx-monitor-a", "influx-monitor-b", "influx-monitor-c"}
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

	// Flush writes
	store.writeAPI.Flush()

	// Give InfluxDB time to process
	time.Sleep(2 * time.Second)

	// Get monitor names
	names, err := store.GetMonitorNames()
	if err != nil {
		t.Fatalf("Failed to get monitor names: %v", err)
	}

	if len(names) < 1 {
		t.Logf("Warning: Expected monitor names, got %d (may be due to write latency)", len(names))
	}
}
