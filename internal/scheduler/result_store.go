package scheduler

import (
	"sort"
	"sync"
	"time"

	"github.com/1broseidon/hallmonitor/pkg/models"
)

// ResultStore manages monitor results in memory with circular buffer behavior
type ResultStore struct {
	maxResults int
	results    map[string]*MonitorResults
	mu         sync.RWMutex
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

// StoreResult stores a monitor result
func (rs *ResultStore) StoreResult(monitorName string, result *models.MonitorResult) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

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
	results := rs.GetResults(monitorName, 1)
	if len(results) > 0 {
		return results[0]
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

// ResultStoreStats represents statistics about the result store
type ResultStoreStats struct {
	Monitors     int        `json:"monitors"`
	MaxResults   int        `json:"max_results"`
	TotalResults int        `json:"total_results"`
	OldestResult *time.Time `json:"oldest_result,omitempty"`
	NewestResult *time.Time `json:"newest_result,omitempty"`
}
