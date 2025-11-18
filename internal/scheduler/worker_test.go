package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// testLogger creates a logger for testing (discards output)
func testLogger(t *testing.T) *logging.Logger {
	t.Helper()
	logger, err := logging.InitLogger(logging.Config{
		Level:  "error",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("failed to create test logger: %v", err)
	}
	return logger
}

// mockMonitor implements the monitors.Monitor interface for testing
type mockMonitor struct {
	name     string
	group    string
	checkErr error
	delay    time.Duration
	status   models.MonitorStatus
}

func (m *mockMonitor) GetName() string {
	return m.name
}

func (m *mockMonitor) GetType() models.MonitorType {
	return models.MonitorTypeHTTP
}

func (m *mockMonitor) GetGroup() string {
	return m.group
}

func (m *mockMonitor) GetConfig() *models.Monitor {
	return &models.Monitor{
		Name:    m.name,
		Type:    models.MonitorTypeHTTP,
		Timeout: models.Duration(5 * time.Second),
	}
}

func (m *mockMonitor) Check(ctx context.Context) (*models.MonitorResult, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return &models.MonitorResult{
				Monitor:   m.name,
				Type:      models.MonitorTypeHTTP,
				Group:     m.group,
				Status:    models.StatusDown,
				Error:     "context canceled",
				Timestamp: time.Now(),
			}, ctx.Err()
		}
	}

	status := m.status
	if status == "" {
		status = models.StatusUp
	}

	return &models.MonitorResult{
		Monitor:   m.name,
		Type:      models.MonitorTypeHTTP,
		Group:     m.group,
		Status:    status,
		Timestamp: time.Now(),
	}, m.checkErr
}

func (m *mockMonitor) Validate() error {
	return nil
}

func (m *mockMonitor) IsEnabled() bool {
	return true
}

func TestWorkerPoolStartStop(t *testing.T) {
	logger := testLogger(t)
	wp := NewWorkerPool(3, logger, nil)

	ctx := context.Background()
	wp.Start(ctx)

	// Give workers time to start
	time.Sleep(10 * time.Millisecond)

	if wp.ActiveWorkers() > 3 {
		t.Fatalf("expected at most 3 active workers, got %d", wp.ActiveWorkers())
	}

	wp.Stop()

	// Verify all workers stopped
	if wp.ActiveWorkers() != 0 {
		t.Fatalf("expected 0 active workers after stop, got %d", wp.ActiveWorkers())
	}
}

func TestWorkerPoolSubmitJobs(t *testing.T) {
	logger := testLogger(t)
	wp := NewWorkerPool(2, logger, nil)
	rs := NewResultStore(10)

	ctx := context.Background()
	wp.Start(ctx)
	defer wp.Stop()

	monitor := &mockMonitor{
		name:   "test-monitor",
		group:  "test-group",
		status: models.StatusUp,
	}

	job := &MonitorJob{
		Monitor:     monitor,
		ResultStore: rs,
		ScheduledAt: time.Now(),
	}

	// Submit job
	if !wp.Submit(job) {
		t.Fatalf("failed to submit job")
	}

	// Wait for job to complete
	time.Sleep(50 * time.Millisecond)

	// Verify result was stored
	result := rs.GetLatestResult("test-monitor")
	if result == nil {
		t.Fatalf("expected result to be stored")
	}

	if result.Status != models.StatusUp {
		t.Fatalf("expected status up, got %s", result.Status)
	}

	if wp.ProcessedJobs() < 1 {
		t.Fatalf("expected at least 1 processed job, got %d", wp.ProcessedJobs())
	}
}

func TestWorkerPoolConcurrency(t *testing.T) {
	logger := testLogger(t)
	workerCount := 5
	wp := NewWorkerPool(workerCount, logger, nil)
	rs := NewResultStore(100)

	ctx := context.Background()
	wp.Start(ctx)
	defer wp.Stop()

	jobCount := 20
	var completed int32
	var submitted int

	for i := 0; i < jobCount; i++ {
		monitor := &mockMonitor{
			name:   "test-monitor-" + string(rune('A'+i)),
			group:  "test-group",
			delay:  10 * time.Millisecond,
			status: models.StatusUp,
		}

		job := &MonitorJob{
			Monitor:     monitor,
			ResultStore: rs,
			ScheduledAt: time.Now(),
		}

		if wp.Submit(job) {
			submitted++
		} else {
			// Queue might be full, wait a bit and retry
			time.Sleep(5 * time.Millisecond)
			if !wp.Submit(job) {
				t.Logf("warning: job %d could not be submitted even after retry", i)
			} else {
				submitted++
			}
		}
	}

	if submitted < jobCount/2 {
		t.Fatalf("too few jobs submitted: %d/%d", submitted, jobCount)
	}

	// Wait for all submitted jobs to complete
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatalf("timeout waiting for jobs to complete, processed: %d/%d", wp.ProcessedJobs(), submitted)
		case <-ticker.C:
			if wp.ProcessedJobs() >= int64(submitted) {
				atomic.StoreInt32(&completed, 1)
				goto done
			}
		}
	}

done:
	if atomic.LoadInt32(&completed) != 1 {
		t.Fatalf("not all jobs completed")
	}

	// Verify results were stored (at least half of them)
	storedCount := 0
	for i := 0; i < jobCount; i++ {
		monitorName := "test-monitor-" + string(rune('A'+i))
		result := rs.GetLatestResult(monitorName)
		if result != nil {
			storedCount++
		}
	}

	if storedCount < submitted/2 {
		t.Fatalf("too few results stored: %d (expected at least %d)", storedCount, submitted/2)
	}
}

func TestWorkerPoolContextCancellation(t *testing.T) {
	logger := testLogger(t)
	wp := NewWorkerPool(2, logger, nil)

	ctx, cancel := context.WithCancel(context.Background())
	wp.Start(ctx)

	// Submit a job with long delay
	monitor := &mockMonitor{
		name:   "test-monitor",
		group:  "test-group",
		delay:  500 * time.Millisecond,
		status: models.StatusUp,
	}

	job := &MonitorJob{
		Monitor:     monitor,
		ResultStore: NewResultStore(10),
		ScheduledAt: time.Now(),
	}

	wp.Submit(job)

	// Cancel context immediately
	cancel()

	// Workers should stop gracefully
	wp.Stop()

	if wp.ActiveWorkers() != 0 {
		t.Fatalf("expected 0 active workers after context cancel, got %d", wp.ActiveWorkers())
	}
}

func TestWorkerPoolQueueFull(t *testing.T) {
	logger := testLogger(t)
	// Create small pool with small queue
	wp := NewWorkerPool(1, logger, nil)
	rs := NewResultStore(10)

	ctx := context.Background()
	wp.Start(ctx)
	defer wp.Stop()

	// Fill the queue with slow jobs
	queueSize := cap(wp.jobQueue)
	for i := 0; i < queueSize+10; i++ {
		monitor := &mockMonitor{
			name:   "slow-monitor",
			group:  "test-group",
			delay:  100 * time.Millisecond,
			status: models.StatusUp,
		}

		job := &MonitorJob{
			Monitor:     monitor,
			ResultStore: rs,
			ScheduledAt: time.Now(),
		}

		submitted := wp.Submit(job)
		if i >= queueSize && submitted {
			t.Logf("job %d submitted despite full queue", i)
		}
	}

	// At least some jobs should have been rejected
	pendingJobs := wp.PendingJobs()
	if pendingJobs <= 0 {
		t.Fatalf("expected some pending jobs")
	}
}

func TestWorkerPoolBackoffIntegration(t *testing.T) {
	logger := testLogger(t)
	wp := NewWorkerPool(2, logger, nil)
	rs := NewResultStore(10)
	bm := NewBackoffManager()

	ctx := context.Background()
	wp.Start(ctx)
	defer wp.Stop()

	// Submit failing monitor
	monitor := &mockMonitor{
		name:   "failing-monitor",
		group:  "test-group",
		status: models.StatusDown,
	}

	for i := 0; i < 3; i++ {
		job := &MonitorJob{
			Monitor:     monitor,
			ResultStore: rs,
			Backoff:     bm,
			ScheduledAt: time.Now(),
		}

		wp.Submit(job)
		time.Sleep(20 * time.Millisecond)
	}

	// Verify backoff was recorded
	backoff := bm.GetBackoff("failing-monitor")
	if backoff == 0 {
		t.Fatalf("expected backoff to be set after failures")
	}

	stats := bm.GetStats()
	if stats.BackedOffMonitors == 0 {
		t.Fatalf("expected at least one backed off monitor")
	}
}
