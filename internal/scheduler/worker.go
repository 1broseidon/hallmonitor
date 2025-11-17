package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/internal/metrics"
	"github.com/1broseidon/hallmonitor/internal/monitors"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// WorkerPool manages a pool of workers for executing monitor checks
type WorkerPool struct {
	size          int
	jobQueue      chan *MonitorJob
	workers       []*Worker
	logger        *logging.Logger
	metrics       *metrics.Metrics
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	processedJobs int64
	activeWorkers int32
}

// MonitorJob represents a monitor check job
type MonitorJob struct {
	Monitor     monitors.Monitor
	ResultStore *ResultStore
	Backoff     *BackoffManager
	ScheduledAt time.Time
}

// Worker represents a single worker goroutine
type Worker struct {
	id      int
	pool    *WorkerPool
	logger  *logging.Logger
	metrics *metrics.Metrics
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(size int, logger *logging.Logger, metrics *metrics.Metrics) *WorkerPool {
	if size <= 0 {
		size = 5 // Default size
	}

	return &WorkerPool{
		size:     size,
		jobQueue: make(chan *MonitorJob, size*2), // Buffer for 2x worker count
		workers:  make([]*Worker, size),
		logger:   logger,
		metrics:  metrics,
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start(ctx context.Context) {
	wp.ctx, wp.cancel = context.WithCancel(ctx)

	wp.logger.WithComponent(logging.ComponentScheduler).
		WithFields(map[string]interface{}{
			"worker_count": wp.size,
			"queue_size":   cap(wp.jobQueue),
		}).
		Info("Starting worker pool")

	// Start workers
	for i := 0; i < wp.size; i++ {
		worker := &Worker{
			id:      i,
			pool:    wp,
			logger:  wp.logger,
			metrics: wp.metrics,
		}
		wp.workers[i] = worker

		wp.wg.Add(1)
		go worker.start(wp.ctx)
	}
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	wp.logger.WithComponent(logging.ComponentScheduler).Info("Stopping worker pool")

	// Cancel context and close job queue
	wp.cancel()
	close(wp.jobQueue)

	// Wait for all workers to finish
	wp.wg.Wait()

	wp.logger.WithComponent(logging.ComponentScheduler).
		WithFields(map[string]interface{}{
			"processed_jobs": atomic.LoadInt64(&wp.processedJobs),
		}).
		Info("Worker pool stopped")
}

// Submit submits a job to the worker pool
func (wp *WorkerPool) Submit(job *MonitorJob) bool {
	select {
	case wp.jobQueue <- job:
		return true
	default:
		// Queue is full
		return false
	}
}

// ActiveWorkers returns the number of currently active workers
func (wp *WorkerPool) ActiveWorkers() int {
	return int(atomic.LoadInt32(&wp.activeWorkers))
}

// PendingJobs returns the number of pending jobs in the queue
func (wp *WorkerPool) PendingJobs() int {
	return len(wp.jobQueue)
}

// ProcessedJobs returns the total number of processed jobs
func (wp *WorkerPool) ProcessedJobs() int64 {
	return atomic.LoadInt64(&wp.processedJobs)
}

// start starts a worker goroutine
func (w *Worker) start(ctx context.Context) {
	defer w.pool.wg.Done()

	w.logger.WithComponent(logging.ComponentScheduler).
		WithFields(map[string]interface{}{
			"worker_id": w.id,
		}).
		Debug("Worker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.WithComponent(logging.ComponentScheduler).
				WithFields(map[string]interface{}{
					"worker_id": w.id,
				}).
				Debug("Worker stopped by context")
			return

		case job, ok := <-w.pool.jobQueue:
			if !ok {
				w.logger.WithComponent(logging.ComponentScheduler).
					WithFields(map[string]interface{}{
						"worker_id": w.id,
					}).
					Debug("Worker stopped - job queue closed")
				return
			}

			w.processJob(ctx, job)
		}
	}
}

// processJob processes a single monitor job
func (w *Worker) processJob(ctx context.Context, job *MonitorJob) {
	// Add panic recovery to prevent worker crashes
	defer func() {
		if r := recover(); r != nil {
			w.logger.WithComponent(logging.ComponentScheduler).
				WithFields(map[string]interface{}{
					"worker_id": w.id,
					"monitor":   job.Monitor.GetName(),
					"panic":     r,
				}).
				Error("Worker panic recovered")

			// Record failure in backoff manager
			if job.Backoff != nil {
				job.Backoff.RecordFailure(job.Monitor.GetName())
			}
		}
	}()

	// Track active worker count
	atomic.AddInt32(&w.pool.activeWorkers, 1)
	defer atomic.AddInt32(&w.pool.activeWorkers, -1)

	// Track processed jobs
	defer atomic.AddInt64(&w.pool.processedJobs, 1)

	// Track running monitors in metrics
	if w.metrics != nil {
		w.metrics.IncrementRunningMonitors()
		defer w.metrics.DecrementRunningMonitors()
	}

	monitor := job.Monitor
	monitorName := monitor.GetName()

	// Create timeout context for the monitor check
	timeout := monitor.GetConfig().Timeout.ToDuration()
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Log start of check
	w.logger.WithComponent(logging.ComponentScheduler).
		WithMonitor(monitorName, string(monitor.GetType()), monitor.GetGroup()).
		WithEvent(logging.EventCheckStarted).
		WithFields(map[string]interface{}{
			"worker_id":    w.id,
			"scheduled_at": job.ScheduledAt,
			"timeout":      timeout,
		}).
		Debug("Starting monitor check")

	startTime := time.Now()

	// Execute the monitor check
	result, err := monitor.Check(checkCtx)

	duration := time.Since(startTime)

	if err != nil {
		// Log error
		w.logger.WithComponent(logging.ComponentScheduler).
			WithMonitor(monitorName, string(monitor.GetType()), monitor.GetGroup()).
			WithEvent(logging.EventCheckFailed).
			WithError(err).
			WithFields(map[string]interface{}{
				"worker_id": w.id,
				"duration":  duration,
			}).
			Error("Monitor check failed")

		// Create error result if monitor didn't return one
		if result == nil {
			result = &models.MonitorResult{
				Monitor:   monitorName,
				Type:      monitor.GetType(),
				Group:     monitor.GetGroup(),
				Status:    models.StatusDown,
				Duration:  duration,
				Error:     err.Error(),
				Timestamp: time.Now(),
			}
		}
	} else {
		// Log successful check
		w.logger.WithComponent(logging.ComponentScheduler).
			WithMonitor(monitorName, string(monitor.GetType()), monitor.GetGroup()).
			WithEvent(logging.EventCheckCompleted).
			WithFields(map[string]interface{}{
				"worker_id": w.id,
				"duration":  duration,
				"status":    string(result.Status),
			}).
			Debug("Monitor check completed")
	}

	// Store the result
	if result != nil {
		job.ResultStore.StoreResult(monitorName, result)

		// Update backoff based on result status
		if job.Backoff != nil {
			if result.Status == models.StatusUp {
				job.Backoff.RecordSuccess(monitorName)
			} else {
				job.Backoff.RecordFailure(monitorName)
			}
		}

		// Log structured monitor result
		w.logger.MonitorCheck(
			monitorName,
			string(monitor.GetType()),
			monitor.GetGroup(),
			string(result.Status),
			duration,
			err,
		)
	}
}
