package scheduler

import (
	"testing"
	"time"

	"github.com/1broseidon/hallmonitor/pkg/models"
)

func newResult(monitor string, status models.MonitorStatus, ts time.Time) *models.MonitorResult {
	return &models.MonitorResult{
		Monitor:   monitor,
		Type:      models.MonitorTypeHTTP,
		Group:     "group",
		Status:    status,
		Timestamp: ts,
	}
}

func TestResultStoreStoreAndRetrieve(t *testing.T) {
	rs := NewResultStore(3)

	now := time.Now()
	rs.StoreResult("alpha", newResult("alpha", models.StatusUp, now.Add(-3*time.Second)))
	rs.StoreResult("alpha", newResult("alpha", models.StatusDown, now.Add(-2*time.Second)))
	rs.StoreResult("alpha", newResult("alpha", models.StatusUp, now.Add(-1*time.Second)))

	results := rs.GetResults("alpha", 0)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].Status != models.StatusUp {
		t.Fatalf("expected newest result to be StatusUp, got %s", results[0].Status)
	}

	latest := rs.GetLatestResult("alpha")
	if latest == nil || latest.Status != models.StatusUp {
		t.Fatalf("expected latest result status up, got %+v", latest)
	}

	// Verify circular buffer eviction when exceeding capacity
	rs.StoreResult("alpha", newResult("alpha", models.StatusDown, now))
	results = rs.GetResults("alpha", 0)
	if len(results) != 3 {
		t.Fatalf("expected results to remain capped at 3, got %d", len(results))
	}

	expectedStatuses := []models.MonitorStatus{models.StatusDown, models.StatusUp, models.StatusDown}
	for i, want := range expectedStatuses {
		if results[i] == nil {
			t.Fatalf("result %d is nil", i)
		}
		if results[i].Status != want {
			t.Fatalf("expected status %s at index %d, got %s", want, i, results[i].Status)
		}
	}
}

func TestResultStoreCleanupOldResults(t *testing.T) {
	rs := NewResultStore(5)

	now := time.Now()
	rs.StoreResult("alpha", newResult("alpha", models.StatusUp, now.Add(-3*time.Hour)))
	rs.StoreResult("alpha", newResult("alpha", models.StatusDown, now.Add(-30*time.Minute)))
	rs.StoreResult("beta", newResult("beta", models.StatusUp, now.Add(-2*time.Hour)))

	cleaned := rs.CleanupOldResults(time.Hour)
	if cleaned != 2 {
		t.Fatalf("expected 2 old results cleaned, got %d", cleaned)
	}

	if latest := rs.GetLatestResult("alpha"); latest == nil || latest.Status != models.StatusDown {
		t.Fatalf("expected recent alpha result to remain after cleanup")
	}

	if latest := rs.GetLatestResult("beta"); latest != nil {
		t.Fatalf("expected beta monitor to be removed after cleaning all results")
	}
}

func TestResultStoreGetUptime(t *testing.T) {
	rs := NewResultStore(10)
	now := time.Now()

	// Results within the last hour
	rs.StoreResult("alpha", newResult("alpha", models.StatusUp, now.Add(-50*time.Minute)))
	rs.StoreResult("alpha", newResult("alpha", models.StatusDown, now.Add(-40*time.Minute)))
	rs.StoreResult("alpha", newResult("alpha", models.StatusUp, now.Add(-30*time.Minute)))

	// Old result outside the period should be ignored
	rs.StoreResult("alpha", newResult("alpha", models.StatusDown, now.Add(-2*time.Hour)))

	uptime := rs.GetUptime("alpha", time.Hour)
	if uptime <= 0 || uptime >= 100 {
		t.Fatalf("expected uptime between 0 and 100, got %.2f", uptime)
	}

	// Up results are 2 out of 3 within the period -> 66.66%
	expected := (2.0 / 3.0) * 100
	if diff := uptime - expected; diff > 0.01 || diff < -0.01 {
		t.Fatalf("expected uptime %.2f, got %.2f", expected, uptime)
	}
}

func TestResultStoreGetHistoricalResultsInMemory(t *testing.T) {
	rs := NewResultStore(10)
	base := time.Now().Add(-10 * time.Minute)

	for i := 0; i < 6; i++ {
		ts := base.Add(time.Duration(i) * time.Minute)
		rs.StoreResult("alpha", newResult("alpha", models.StatusUp, ts))
	}

	start := base.Add(2 * time.Minute)
	end := base.Add(5 * time.Minute)

	results, err := rs.GetHistoricalResults("alpha", start, end, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 results in range, got %d", len(results))
	}

	for i := 1; i < len(results); i++ {
		if !results[i-1].Timestamp.Before(results[i].Timestamp) && !results[i-1].Timestamp.Equal(results[i].Timestamp) {
			t.Fatalf("results not sorted chronologically")
		}
	}

	limited, err := rs.GetHistoricalResults("alpha", start, end, 2)
	if err != nil {
		t.Fatalf("expected no error limiting results, got %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("expected 2 limited results, got %d", len(limited))
	}
}
