package scheduler

import (
	"sync"
	"time"
)

// BackoffManager manages exponential backoff for failing monitors
type BackoffManager struct {
	failures map[string]int       // Monitor name -> failure count
	lastFail map[string]time.Time // Monitor name -> last failure time
	mu       sync.RWMutex

	// Configuration
	maxRetries     int           // Maximum consecutive failures before max backoff
	baseDelay      time.Duration // Base delay (e.g., 1s)
	maxDelay       time.Duration // Maximum delay (e.g., 5m)
	resetThreshold time.Duration // Time after which to reset failure count
}

// NewBackoffManager creates a new backoff manager
func NewBackoffManager() *BackoffManager {
	return &BackoffManager{
		failures:       make(map[string]int),
		lastFail:       make(map[string]time.Time),
		maxRetries:     5,
		baseDelay:      time.Second,
		maxDelay:       5 * time.Minute,
		resetThreshold: 10 * time.Minute,
	}
}

// RecordSuccess records a successful monitor check
func (bm *BackoffManager) RecordSuccess(monitorName string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Reset failure count on success
	delete(bm.failures, monitorName)
	delete(bm.lastFail, monitorName)
}

// RecordFailure records a failed monitor check
func (bm *BackoffManager) RecordFailure(monitorName string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.failures[monitorName]++
	bm.lastFail[monitorName] = time.Now()
}

// GetBackoff returns the backoff duration for a monitor
func (bm *BackoffManager) GetBackoff(monitorName string) time.Duration {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	failures, exists := bm.failures[monitorName]
	if !exists {
		return 0 // No backoff needed
	}

	// Check if we should reset based on threshold
	if lastFail, ok := bm.lastFail[monitorName]; ok {
		if time.Since(lastFail) > bm.resetThreshold {
			// Reset will happen on next call, but for now return 0
			return 0
		}
	}

	// Calculate exponential backoff: baseDelay * 2^failures
	backoff := bm.baseDelay
	for i := 0; i < failures-1 && i < bm.maxRetries; i++ {
		backoff *= 2
		if backoff > bm.maxDelay {
			backoff = bm.maxDelay
			break
		}
	}

	return backoff
}

// ShouldCheck determines if a monitor should be checked based on backoff
func (bm *BackoffManager) ShouldCheck(monitorName string, lastCheckTime time.Time) bool {
	backoff := bm.GetBackoff(monitorName)
	if backoff == 0 {
		return true
	}

	return time.Since(lastCheckTime) >= backoff
}

// GetStats returns backoff statistics
func (bm *BackoffManager) GetStats() BackoffStats {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	backedOffMonitors := 0
	maxFailures := 0

	for _, failures := range bm.failures {
		if failures > 0 {
			backedOffMonitors++
		}
		if failures > maxFailures {
			maxFailures = failures
		}
	}

	return BackoffStats{
		BackedOffMonitors: backedOffMonitors,
		TotalMonitored:    len(bm.failures),
		MaxFailures:       maxFailures,
	}
}

// Reset resets backoff for a specific monitor
func (bm *BackoffManager) Reset(monitorName string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	delete(bm.failures, monitorName)
	delete(bm.lastFail, monitorName)
}

// ResetAll resets all backoff state
func (bm *BackoffManager) ResetAll() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.failures = make(map[string]int)
	bm.lastFail = make(map[string]time.Time)
}

// CleanupStale removes old entries
func (bm *BackoffManager) CleanupStale() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	now := time.Now()
	for monitor, lastFail := range bm.lastFail {
		if now.Sub(lastFail) > bm.resetThreshold {
			delete(bm.failures, monitor)
			delete(bm.lastFail, monitor)
		}
	}
}

// BackoffStats represents backoff statistics
type BackoffStats struct {
	BackedOffMonitors int `json:"backed_off_monitors"`
	TotalMonitored    int `json:"total_monitored"`
	MaxFailures       int `json:"max_failures"`
}
