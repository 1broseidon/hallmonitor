package scheduler

import (
	"testing"
	"time"
)

func TestBackoffManagerBackoffSequence(t *testing.T) {
	bm := NewBackoffManager()
	bm.baseDelay = 100 * time.Millisecond
	bm.maxDelay = time.Second
	bm.maxRetries = 10

	name := "monitor-1"

	expected := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		time.Second,
		time.Second,
	}

	for i, want := range expected {
		bm.RecordFailure(name)
		got := bm.GetBackoff(name)
		if got != want {
			t.Fatalf("failure %d: expected backoff %s, got %s", i+1, want, got)
		}
	}
}

func TestBackoffManagerResetOnSuccess(t *testing.T) {
	bm := NewBackoffManager()
	bm.RecordFailure("monitor")

	bm.RecordSuccess("monitor")

	if backoff := bm.GetBackoff("monitor"); backoff != 0 {
		t.Fatalf("expected backoff to reset after success, got %s", backoff)
	}

	if stats := bm.GetStats(); stats.BackedOffMonitors != 0 || stats.TotalMonitored != 0 {
		t.Fatalf("expected stats to reset after success, got %+v", stats)
	}
}

func TestBackoffManagerResetThreshold(t *testing.T) {
	bm := NewBackoffManager()
	bm.RecordFailure("stale-monitor")

	// Simulate last failure beyond reset threshold
	bm.mu.Lock()
	bm.lastFail["stale-monitor"] = time.Now().Add(-bm.resetThreshold - time.Minute)
	bm.mu.Unlock()

	if backoff := bm.GetBackoff("stale-monitor"); backoff != 0 {
		t.Fatalf("expected backoff to be zero after reset threshold, got %s", backoff)
	}

	bm.CleanupStale()

	if backoff := bm.GetBackoff("stale-monitor"); backoff != 0 {
		t.Fatalf("expected stale monitor to be removed after cleanup, got backoff %s", backoff)
	}
}

func TestBackoffManagerShouldCheck(t *testing.T) {
	bm := NewBackoffManager()
	bm.baseDelay = 200 * time.Millisecond
	bm.maxDelay = time.Second

	name := "monitor"
	bm.RecordFailure(name)
	bm.RecordFailure(name)

	backoff := bm.GetBackoff(name)

	if backoff == 0 {
		t.Fatalf("expected non-zero backoff after failures")
	}

	if bm.ShouldCheck(name, time.Now()) {
		t.Fatalf("expected ShouldCheck to be false when last check happened now")
	}

	if !bm.ShouldCheck(name, time.Now().Add(-backoff-time.Millisecond)) {
		t.Fatalf("expected ShouldCheck to be true when backoff has elapsed")
	}
}
