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
		Monitor:      "test-influx-monitor",
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

	// Flush writes
	store.writeAPI.Flush()

	// Wait a bit for data to be available
	time.Sleep(2 * time.Second)

	// Test GetLatestResult
	retrieved, err := store.GetLatestResult("test-influx-monitor")
	if err != nil {
		t.Fatalf("Failed to retrieve result: %v", err)
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
	monitorName := "test-influx-multiple"
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

	// Flush writes
	store.writeAPI.Flush()

	// Wait for data to be available
	time.Sleep(2 * time.Second)

	// Retrieve results
	results, err := store.GetResults(monitorName, now.Add(-1*time.Hour), now.Add(1*time.Hour), 10)
	if err != nil {
		t.Fatalf("Failed to retrieve results: %v", err)
	}

	if len(results) < 1 {
		t.Errorf("Expected at least 1 result, got %d", len(results))
	}
}

func TestInfluxDBStore_MonitorNames(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	url, token, org, bucket := getTestInfluxDBConfig()

	store, err := NewInfluxDBStore(url, token, org, bucket, logger)
	if err != nil {
		t.Skipf("InfluxDB not available: %v", err)
	}
	defer store.Close()

	// Store results for a test monitor
	testMonitor := "test-influx-names"
	result := &models.MonitorResult{
		Monitor:      testMonitor,
		Type:         models.MonitorTypeHTTP,
		Status:       models.StatusUp,
		Timestamp:    time.Now(),
		ResponseTime: 100 * time.Millisecond,
		StatusCode:   200,
	}
	if err := store.StoreResult(result); err != nil {
		t.Fatalf("Failed to store result: %v", err)
	}

	// Flush writes
	store.writeAPI.Flush()

	// Wait for data to be available
	time.Sleep(2 * time.Second)

	// Get monitor names
	names, err := store.GetMonitorNames()
	if err != nil {
		t.Fatalf("Failed to get monitor names: %v", err)
	}

	// Check that our monitor is in the list
	found := false
	for _, name := range names {
		if name == testMonitor {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find monitor %s in list", testMonitor)
	}
}

func TestInfluxDBStore_Capabilities(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{Level: "error", Format: "json", Output: "stdout"})

	url, token, org, bucket := getTestInfluxDBConfig()

	store, err := NewInfluxDBStore(url, token, org, bucket, logger)
	if err != nil {
		t.Skipf("InfluxDB not available: %v", err)
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
