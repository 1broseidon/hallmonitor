package storage

import (
	"testing"
	"time"

	"github.com/1broseidon/hallmonitor/pkg/models"
)

func TestNoOpStore_StoreResult(t *testing.T) {
	store := NewNoOpStore()

	result := &models.MonitorResult{
		Monitor:   "test-monitor",
		Type:      models.MonitorTypeHTTP,
		Status:    models.StatusUp,
		Timestamp: time.Now(),
	}

	err := store.StoreResult(result)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestNoOpStore_GetLatestResult(t *testing.T) {
	store := NewNoOpStore()

	result, err := store.GetLatestResult("test-monitor")
	if err != ErrNotSupported {
		t.Errorf("Expected ErrNotSupported, got %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

func TestNoOpStore_GetResults(t *testing.T) {
	store := NewNoOpStore()

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()

	results, err := store.GetResults("test-monitor", start, end, 100)
	if err != ErrNotSupported {
		t.Errorf("Expected ErrNotSupported, got %v", err)
	}
	if results != nil {
		t.Errorf("Expected nil results, got %v", results)
	}
}

func TestNoOpStore_GetAggregates(t *testing.T) {
	store := NewNoOpStore()

	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	aggregates, err := store.GetAggregates("test-monitor", "hour", start, end)
	if err != ErrNotSupported {
		t.Errorf("Expected ErrNotSupported, got %v", err)
	}
	if aggregates != nil {
		t.Errorf("Expected nil aggregates, got %v", aggregates)
	}
}

func TestNoOpStore_StoreAggregate(t *testing.T) {
	store := NewNoOpStore()

	agg := &models.AggregateResult{
		Monitor:     "test-monitor",
		PeriodStart: time.Now().Add(-1 * time.Hour),
		PeriodEnd:   time.Now(),
		PeriodType:  "hour",
		TotalChecks: 10,
		UpChecks:    8,
	}

	err := store.StoreAggregate(agg)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestNoOpStore_GetMonitorNames(t *testing.T) {
	store := NewNoOpStore()

	names, err := store.GetMonitorNames()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(names) != 0 {
		t.Errorf("Expected empty list, got %v", names)
	}
}

func TestNoOpStore_Close(t *testing.T) {
	store := NewNoOpStore()

	err := store.Close()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestNoOpStore_Capabilities(t *testing.T) {
	store := NewNoOpStore()

	caps := store.Capabilities()

	if caps.SupportsAggregation {
		t.Error("Expected SupportsAggregation to be false")
	}
	if caps.SupportsRetention {
		t.Error("Expected SupportsRetention to be false")
	}
	if caps.SupportsRawResults {
		t.Error("Expected SupportsRawResults to be false")
	}
	if !caps.ReadOnly {
		t.Error("Expected ReadOnly to be true")
	}
}
