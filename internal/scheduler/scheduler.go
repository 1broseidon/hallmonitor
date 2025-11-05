package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/internal/metrics"
	"github.com/1broseidon/hallmonitor/internal/monitors"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// Scheduler manages the execution of monitor checks
type Scheduler struct {
	logger         *logging.Logger
	metrics        *metrics.Metrics
	monitorManager *monitors.MonitorManager
	resultStore    *ResultStore
	workers        *WorkerPool
	backoff        *BackoffManager
	stopChan       chan struct{}
	wg             sync.WaitGroup
	running        bool
	mu             sync.RWMutex
}

// NewScheduler creates a new scheduler instance
func NewScheduler(logger *logging.Logger, metrics *metrics.Metrics, monitorManager *monitors.MonitorManager) *Scheduler {
	return &Scheduler{
		logger:         logger,
		metrics:        metrics,
		monitorManager: monitorManager,
		resultStore:    NewResultStore(1000),               // Keep last 1000 results per monitor
		workers:        NewWorkerPool(10, logger, metrics), // 10 concurrent workers
		backoff:        NewBackoffManager(),
		stopChan:       make(chan struct{}),
		running:        false,
	}
}

// Start begins the monitoring schedule
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	s.logger.WithComponent(logging.ComponentScheduler).Info("Starting scheduler")

	// Start worker pool
	s.workers.Start(ctx)

	// Start scheduling goroutine
	s.wg.Add(1)
	go s.schedulingLoop(ctx)

	s.running = true
	return nil
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.WithComponent(logging.ComponentScheduler).Info("Stopping scheduler")

	// Signal stop
	close(s.stopChan)

	// Stop worker pool
	s.workers.Stop()

	// Wait for scheduling loop to finish
	s.wg.Wait()

	s.running = false
	return nil
}

// IsRunning returns whether the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetResults returns recent results for a monitor
func (s *Scheduler) GetResults(monitorName string, limit int) []*models.MonitorResult {
	return s.resultStore.GetResults(monitorName, limit)
}

// GetLatestResult returns the most recent result for a monitor
func (s *Scheduler) GetLatestResult(monitorName string) *models.MonitorResult {
	results := s.resultStore.GetResults(monitorName, 1)
	if len(results) > 0 {
		return results[0]
	}
	return nil
}

// GetAllLatestResults returns the latest result for each monitor
func (s *Scheduler) GetAllLatestResults() map[string]*models.MonitorResult {
	results := make(map[string]*models.MonitorResult)

	for _, monitor := range s.monitorManager.GetMonitors() {
		if latest := s.GetLatestResult(monitor.GetName()); latest != nil {
			results[monitor.GetName()] = latest
		}
	}

	return results
}

// schedulingLoop is the main scheduling loop
func (s *Scheduler) schedulingLoop(ctx context.Context) {
	defer s.wg.Done()

	// Create a ticker for scheduling checks
	ticker := time.NewTicker(1 * time.Second) // Check every second for due monitors
	defer ticker.Stop()

	// Track next execution time for each monitor
	nextExecution := make(map[string]time.Time)

	// Initialize next execution times
	for _, monitor := range s.monitorManager.GetMonitors() {
		if monitor.IsEnabled() {
			// Add jitter to prevent thundering herd
			jitter := time.Duration(rand.Intn(5)) * time.Second
			nextExecution[monitor.GetName()] = time.Now().Add(jitter)
		}
	}

	s.logger.WithComponent(logging.ComponentScheduler).
		WithFields(map[string]interface{}{
			"monitors": len(nextExecution),
		}).
		Info("Scheduler loop started")

	for {
		select {
		case <-ctx.Done():
			s.logger.WithComponent(logging.ComponentScheduler).Info("Scheduler stopped by context")
			return
		case <-s.stopChan:
			s.logger.WithComponent(logging.ComponentScheduler).Info("Scheduler stopped by signal")
			return
		case now := <-ticker.C:
			s.checkAndScheduleMonitors(ctx, now, nextExecution)
		}
	}
}

// checkAndScheduleMonitors checks which monitors are due and schedules them
func (s *Scheduler) checkAndScheduleMonitors(ctx context.Context, now time.Time, nextExecution map[string]time.Time) {
	for _, monitor := range s.monitorManager.GetMonitors() {
		if !monitor.IsEnabled() {
			continue
		}

		monitorName := monitor.GetName()

		// Check if this monitor is due for execution
		if nextTime, exists := nextExecution[monitorName]; exists && now.After(nextTime) {
			// Schedule the monitor check
			job := &MonitorJob{
				Monitor:     monitor,
				ResultStore: s.resultStore,
				Backoff:     s.backoff,
				ScheduledAt: now,
			}

			// Submit job to worker pool
			select {
			case <-ctx.Done():
				return
			default:
				if s.workers.Submit(job) {
					// Update next execution time
					interval := monitor.GetConfig().Interval
					if interval == 0 {
						interval = 30 * time.Second // fallback
					}

					// Add small jitter (±10% of interval) to prevent synchronization
					jitter := time.Duration(rand.Intn(int(interval.Nanoseconds()/5))) - interval/10
					nextExecution[monitorName] = now.Add(interval).Add(jitter)

					s.logger.WithComponent(logging.ComponentScheduler).
						WithFields(map[string]interface{}{
							"monitor":    monitorName,
							"next_check": nextExecution[monitorName],
							"interval":   interval,
						}).
						Debug("Monitor scheduled")
				} else {
					// Worker pool is full, skip this execution
					s.logger.WithComponent(logging.ComponentScheduler).
						WithFields(map[string]interface{}{
							"monitor": monitorName,
						}).
						Warn("Worker pool full, skipping monitor check")
				}
			}
		}
	}
}

// GetStats returns scheduler statistics
func (s *Scheduler) GetStats() SchedulerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := SchedulerStats{
		Running:       s.running,
		TotalMonitors: len(s.monitorManager.GetMonitors()),
		ActiveWorkers: s.workers.ActiveWorkers(),
		PendingJobs:   s.workers.PendingJobs(),
		ProcessedJobs: s.workers.ProcessedJobs(),
	}

	// Count enabled monitors
	for _, monitor := range s.monitorManager.GetMonitors() {
		if monitor.IsEnabled() {
			stats.EnabledMonitors++
		}
	}

	return stats
}

// SchedulerStats represents scheduler statistics
type SchedulerStats struct {
	Running         bool  `json:"running"`
	TotalMonitors   int   `json:"total_monitors"`
	EnabledMonitors int   `json:"enabled_monitors"`
	ActiveWorkers   int   `json:"active_workers"`
	PendingJobs     int   `json:"pending_jobs"`
	ProcessedJobs   int64 `json:"processed_jobs"`
}

// Simple random number generator for jitter
var rand = &simpleRand{seed: uint64(time.Now().UnixNano())}

type simpleRand struct {
	seed uint64
}

func (r *simpleRand) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	r.seed = r.seed*1664525 + 1013904223
	return int(r.seed % uint64(n))
}
