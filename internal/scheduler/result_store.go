package scheduler

import (
	"sort"
	"sync"
	"time"

	"github.com/1broseidon/hallmonitor/pkg/models"
)

// PersistentStore interface for persistent storage backend
type PersistentStore interface {
	StoreResult(result *models.MonitorResult) error
	GetLatestResult(monitor string) (*models.MonitorResult, error)
	GetResults(monitor string, start, end time.Time, limit int) ([]*models.MonitorResult, error)
}

// ResultStore manages monitor results in memory with optional persistent backend
type ResultStore struct {
	maxResults      int
	results         map[string]*MonitorResults
	mu              sync.RWMutex
	persistentStore PersistentStore // Optional BadgerDB backend
}

// MonitorResults holds results for a specific monitor
type MonitorResults struct {
	Results []*models.MonitorResult
	Index   int // Current write index (circular buffer)
	Count   int // Total results stored (up to maxResults)
}

// NewResultStore creates a new result store
func NewResultStore(maxResults int) *ResultStore {
	if maxResults <= 0 {
		maxResults = 100 // Default size
	}

	return &ResultStore{
		maxResults: maxResults,
		results:    make(map[string]*MonitorResults),
	}
}

// NewResultStoreWithPersistence creates a new result store with persistent backend
func NewResultStoreWithPersistence(maxResults int, persistentStore PersistentStore) *ResultStore {
	if maxResults <= 0 {
		maxResults = 100 // Default size
	}

	return &ResultStore{
		maxResults:      maxResults,
		results:         make(map[string]*MonitorResults),
		persistentStore: persistentStore,
	}
}

// StoreResult stores a monitor result in memory and optionally to persistent storage
func (rs *ResultStore) StoreResult(monitorName string, result *models.MonitorResult) {
	// Store in memory
	rs.mu.Lock()
	
	// Get or create monitor results
	monitorResults, exists := rs.results[monitorName]
	if !exists {
		monitorResults = &MonitorResults{
			Results: make([]*models.MonitorResult, rs.maxResults),
			Index:   0,
			Count:   0,
		}
		rs.results[monitorName] = monitorResults
	}

	// Store result in circular buffer
	monitorResults.Results[monitorResults.Index] = result
	monitorResults.Index = (monitorResults.Index + 1) % rs.maxResults

	// Update count (capped at maxResults)
	if monitorResults.Count < rs.maxResults {
		monitorResults.Count++
	}
	
	rs.mu.Unlock()

	// Store to persistent storage (if available) - do this outside the lock to avoid blocking
	if rs.persistentStore != nil {
		// Fire and forget - we don't want to slow down the monitoring
		go func() {
			if err := rs.persistentStore.StoreResult(result); err != nil {
				// Log error but don't fail the operation
				// The logger would need to be passed in, but for now we silently ignore
				// This could be improved by adding a logger to the ResultStore
			}
		}()
	}
}

// GetResults returns the most recent results for a monitor
func (rs *ResultStore) GetResults(monitorName string, limit int) []*models.MonitorResult {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	monitorResults, exists := rs.results[monitorName]
	if !exists || monitorResults.Count == 0 {
		return []*models.MonitorResult{}
	}

	// Determine how many results to return
	count := monitorResults.Count
	if limit > 0 && limit < count {
		count = limit
	}

	// Collect results in reverse chronological order (newest first)
	results := make([]*models.MonitorResult, 0, count)

	for i := 0; i < count; i++ {
		// Calculate index going backwards from the most recent
		idx := (monitorResults.Index - 1 - i + rs.maxResults) % rs.maxResults
		if monitorResults.Results[idx] != nil {
			results = append(results, monitorResults.Results[idx])
		}
	}

	return results
}

// GetLatestResult returns the most recent result for a monitor
func (rs *ResultStore) GetLatestResult(monitorName string) *models.MonitorResult {
	// Try memory first
	results := rs.GetResults(monitorName, 1)
	if len(results) > 0 {
		return results[0]
	}

	// Fallback to persistent storage if available
	if rs.persistentStore != nil {
		result, err := rs.persistentStore.GetLatestResult(monitorName)
		if err == nil && result != nil {
			return result
		}
	}

	return nil
}

// GetAllLatestResults returns the latest result for each monitor
func (rs *ResultStore) GetAllLatestResults() map[string]*models.MonitorResult {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	results := make(map[string]*models.MonitorResult)

	for monitorName := range rs.results {
		if latest := rs.GetLatestResult(monitorName); latest != nil {
			results[monitorName] = latest
		}
	}

	return results
}

// GetMonitorNames returns all monitor names that have results
func (rs *ResultStore) GetMonitorNames() []string {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	names := make([]string, 0, len(rs.results))
	for name := range rs.results {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// GetStats returns statistics about stored results
func (rs *ResultStore) GetStats() ResultStoreStats {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	stats := ResultStoreStats{
		Monitors:     len(rs.results),
		MaxResults:   rs.maxResults,
		TotalResults: 0,
	}

	var oldestTime, newestTime time.Time
	first := true

	for _, monitorResults := range rs.results {
		stats.TotalResults += monitorResults.Count

		// Find oldest and newest timestamps
		for i := 0; i < monitorResults.Count; i++ {
			if monitorResults.Results[i] != nil {
				timestamp := monitorResults.Results[i].Timestamp
				if first {
					oldestTime = timestamp
					newestTime = timestamp
					first = false
				} else {
					if timestamp.Before(oldestTime) {
						oldestTime = timestamp
					}
					if timestamp.After(newestTime) {
						newestTime = timestamp
					}
				}
			}
		}
	}

	if !first {
		stats.OldestResult = &oldestTime
		stats.NewestResult = &newestTime
	}

	return stats
}

// CleanupOldResults removes results older than the specified duration
func (rs *ResultStore) CleanupOldResults(maxAge time.Duration) int {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	totalCleaned := 0

	for monitorName, monitorResults := range rs.results {
		cleaned := 0

		// Create new slice for results that are not too old
		newResults := make([]*models.MonitorResult, rs.maxResults)
		newIndex := 0
		newCount := 0

		// Copy results that are newer than cutoff
		for i := 0; i < monitorResults.Count; i++ {
			idx := (monitorResults.Index - 1 - i + rs.maxResults) % rs.maxResults
			result := monitorResults.Results[idx]

			if result != nil && result.Timestamp.After(cutoff) {
				newResults[newIndex] = result
				newIndex = (newIndex + 1) % rs.maxResults
				newCount++
			} else {
				cleaned++
			}
		}

		// Update monitor results if any were cleaned
		if cleaned > 0 {
			monitorResults.Results = newResults
			monitorResults.Index = newIndex
			monitorResults.Count = newCount
			totalCleaned += cleaned

			// Remove monitor entirely if no results remain
			if newCount == 0 {
				delete(rs.results, monitorName)
			}
		}
	}

	return totalCleaned
}

// GetUptime calculates uptime percentage for a monitor over a given period
func (rs *ResultStore) GetUptime(monitorName string, period time.Duration) float64 {
	results := rs.GetResults(monitorName, -1) // Get all results

	if len(results) == 0 {
		return 0.0
	}

	cutoff := time.Now().Add(-period)
	var total, up int

	for _, result := range results {
		if result.Timestamp.After(cutoff) {
			total++
			if result.Status == models.StatusUp {
				up++
			}
		}
	}

	if total == 0 {
		return 0.0
	}

	return float64(up) / float64(total) * 100.0
}

// GetHistoricalResults retrieves results from persistent storage for a time range
func (rs *ResultStore) GetHistoricalResults(monitorName string, start, end time.Time, limit int) ([]*models.MonitorResult, error) {
	if rs.persistentStore == nil {
		// No persistent storage, return empty results
		return []*models.MonitorResult{}, nil
	}

	return rs.persistentStore.GetResults(monitorName, start, end, limit)
}

// ResultStoreStats represents statistics about the result store
type ResultStoreStats struct {
	Monitors     int        `json:"monitors"`
	MaxResults   int        `json:"max_results"`
	TotalResults int        `json:"total_results"`
	OldestResult *time.Time `json:"oldest_result,omitempty"`
	NewestResult *time.Time `json:"newest_result,omitempty"`
}
