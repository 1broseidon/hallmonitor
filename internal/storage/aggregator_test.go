package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

func TestAggregator_CalculateAggregate(t *testing.T) {
	// Create logger
	logger, err := logging.InitLogger(logging.Config{
		Level:  "error",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create aggregator
	agg := &Aggregator{
		logger: logger,
	}

	// Create test results
	now := time.Now()
	results := []*models.MonitorResult{
		{
			Monitor:   "test-monitor",
			Status:    models.StatusUp,
			Duration:  100 * time.Millisecond,
			Timestamp: now,
		},
		{
			Monitor:   "test-monitor",
			Status:    models.StatusUp,
			Duration:  200 * time.Millisecond,
			Timestamp: now.Add(1 * time.Minute),
		},
		{
			Monitor:   "test-monitor",
			Status:    models.StatusDown,
			Duration:  50 * time.Millisecond,
			Timestamp: now.Add(2 * time.Minute),
		},
		{
			Monitor:   "test-monitor",
			Status:    models.StatusUp,
			Duration:  150 * time.Millisecond,
			Timestamp: now.Add(3 * time.Minute),
		},
	}

	// Calculate aggregate
	aggregate := agg.calculateAggregate("test-monitor", "hour", now, now.Add(time.Hour), results)

	// Verify results
	if aggregate.TotalChecks != 4 {
		t.Errorf("Expected 4 total checks, got %d", aggregate.TotalChecks)
	}
	if aggregate.UpChecks != 3 {
		t.Errorf("Expected 3 up checks, got %d", aggregate.UpChecks)
	}
	if aggregate.DownChecks != 1 {
		t.Errorf("Expected 1 down check, got %d", aggregate.DownChecks)
	}

	expectedUptime := 75.0 // 3 out of 4
	if aggregate.UptimePercent != expectedUptime {
		t.Errorf("Expected %.2f%% uptime, got %.2f%%", expectedUptime, aggregate.UptimePercent)
	}

	// Check duration stats
	if aggregate.MinDuration != 50*time.Millisecond {
		t.Errorf("Expected min duration 50ms, got %v", aggregate.MinDuration)
	}
	if aggregate.MaxDuration != 200*time.Millisecond {
		t.Errorf("Expected max duration 200ms, got %v", aggregate.MaxDuration)
	}

	expectedAvg := (100 + 200 + 50 + 150) / 4
	if aggregate.AvgDuration != time.Duration(expectedAvg)*time.Millisecond {
		t.Errorf("Expected avg duration %dms, got %v", expectedAvg, aggregate.AvgDuration)
	}
}

func TestAggregator_StartStop(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "hallmonitor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

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
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create aggregator
	aggregator := NewAggregator(store, logger)

	// Start aggregator
	ctx := context.Background()
	err = aggregator.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start aggregator: %v", err)
	}

	// Verify it's running
	if !aggregator.running {
		t.Error("Aggregator should be running")
	}

	// Stop aggregator
	err = aggregator.Stop()
	if err != nil {
		t.Fatalf("Failed to stop aggregator: %v", err)
	}

	// Verify it's stopped
	if aggregator.running {
		t.Error("Aggregator should not be running")
	}
}

func TestAggregator_EmptyResults(t *testing.T) {
	// Create logger
	logger, err := logging.InitLogger(logging.Config{
		Level:  "error",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create aggregator
	agg := &Aggregator{
		logger: logger,
	}

	// Calculate aggregate with no results
	now := time.Now()
	results := []*models.MonitorResult{}

	aggregate := agg.calculateAggregate("test-monitor", "hour", now, now.Add(time.Hour), results)

	// Verify results
	if aggregate.TotalChecks != 0 {
		t.Errorf("Expected 0 total checks, got %d", aggregate.TotalChecks)
	}
	if aggregate.UptimePercent != 0.0 {
		t.Errorf("Expected 0%% uptime, got %.2f%%", aggregate.UptimePercent)
	}
}

func TestAggregator_AllUp(t *testing.T) {
	// Create logger
	logger, err := logging.InitLogger(logging.Config{
		Level:  "error",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create aggregator
	agg := &Aggregator{
		logger: logger,
	}

	// Create test results - all up
	now := time.Now()
	results := []*models.MonitorResult{
		{
			Monitor:   "test-monitor",
			Status:    models.StatusUp,
			Duration:  100 * time.Millisecond,
			Timestamp: now,
		},
		{
			Monitor:   "test-monitor",
			Status:    models.StatusUp,
			Duration:  110 * time.Millisecond,
			Timestamp: now.Add(1 * time.Minute),
		},
	}

	// Calculate aggregate
	aggregate := agg.calculateAggregate("test-monitor", "hour", now, now.Add(time.Hour), results)

	// Verify 100% uptime
	if aggregate.UptimePercent != 100.0 {
		t.Errorf("Expected 100%% uptime, got %.2f%%", aggregate.UptimePercent)
	}
	if aggregate.UpChecks != 2 {
		t.Errorf("Expected 2 up checks, got %d", aggregate.UpChecks)
	}
	if aggregate.DownChecks != 0 {
		t.Errorf("Expected 0 down checks, got %d", aggregate.DownChecks)
	}
}
