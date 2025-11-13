package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// Aggregator generates hourly and daily aggregate statistics from raw monitor results
type Aggregator struct {
	store   *BadgerStore
	logger  *logging.Logger
	stopCh  chan struct{}
	wg      sync.WaitGroup
	running bool
	mu      sync.RWMutex
}

// NewAggregator creates a new aggregator instance
func NewAggregator(store *BadgerStore, logger *logging.Logger) *Aggregator {
	return &Aggregator{
		store:  store,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// Start begins the aggregation background process
func (a *Aggregator) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return nil
	}

	a.logger.WithComponent("aggregator").Info("Starting aggregation service")

	a.wg.Add(1)
	go a.aggregationLoop(ctx)

	a.running = true
	return nil
}

// Stop stops the aggregation service
func (a *Aggregator) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.logger.WithComponent("aggregator").Info("Stopping aggregation service")
	close(a.stopCh)
	a.wg.Wait()

	a.running = false
	return nil
}

// aggregationLoop runs the periodic aggregation
func (a *Aggregator) aggregationLoop(ctx context.Context) {
	defer a.wg.Done()

	// Run immediately on startup to catch up on any missing aggregations
	a.runAggregation()

	// Then run every hour
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.logger.WithComponent("aggregator").Info("Aggregation stopped by context")
			return
		case <-a.stopCh:
			a.logger.WithComponent("aggregator").Info("Aggregation stopped by signal")
			return
		case <-ticker.C:
			a.runAggregation()
		}
	}
}

// runAggregation performs the aggregation for all monitors
func (a *Aggregator) runAggregation() {
	a.logger.WithComponent("aggregator").Info("Running aggregation")

	// Get last aggregation time
	lastRun := a.getLastAggregationTime()
	now := time.Now()

	// Get all monitor names
	monitors, err := a.store.GetMonitorNames()
	if err != nil {
		a.logger.WithComponent("aggregator").
			WithError(err).
			Error("Failed to get monitor names")
		return
	}

	// Aggregate each monitor
	for _, monitor := range monitors {
		if err := a.aggregateMonitor(monitor, lastRun, now); err != nil {
			a.logger.WithComponent("aggregator").
				WithError(err).
				WithFields(map[string]interface{}{"monitor": monitor}).
				Warn("Failed to aggregate monitor")
		}
	}

	// Update last aggregation time
	a.setLastAggregationTime(now)

	a.logger.WithComponent("aggregator").
		WithFields(map[string]interface{}{
			"monitors": len(monitors),
		}).
		Info("Aggregation completed")
}

// aggregateMonitor aggregates data for a single monitor
func (a *Aggregator) aggregateMonitor(monitor string, lastRun, now time.Time) error {
	// Aggregate hourly data
	if err := a.aggregateHourly(monitor, lastRun, now); err != nil {
		return fmt.Errorf("hourly aggregation failed: %w", err)
	}

	// Aggregate daily data
	if err := a.aggregateDaily(monitor, lastRun, now); err != nil {
		return fmt.Errorf("daily aggregation failed: %w", err)
	}

	return nil
}

// aggregateHourly generates hourly aggregates for a monitor
func (a *Aggregator) aggregateHourly(monitor string, lastRun, now time.Time) error {
	// Start from the hour after last run (or 24 hours ago if no last run)
	if lastRun.IsZero() {
		lastRun = now.Add(-24 * time.Hour)
	}

	// Truncate to the start of the hour
	currentHour := lastRun.Truncate(time.Hour)
	endHour := now.Truncate(time.Hour)

	for currentHour.Before(endHour) {
		nextHour := currentHour.Add(time.Hour)

		// Get results for this hour
		results, err := a.store.GetResultsByPeriod(monitor, currentHour, nextHour)
		if err != nil {
			return fmt.Errorf("failed to get results for hour %s: %w", currentHour, err)
		}

		// Skip if no results
		if len(results) == 0 {
			currentHour = nextHour
			continue
		}

		// Calculate aggregate
		agg := a.calculateAggregate(monitor, "hour", currentHour, nextHour, results)

		// Store aggregate
		if err := a.store.StoreAggregate(agg); err != nil {
			return fmt.Errorf("failed to store hourly aggregate: %w", err)
		}

		currentHour = nextHour
	}

	return nil
}

// aggregateDaily generates daily aggregates for a monitor
func (a *Aggregator) aggregateDaily(monitor string, lastRun, now time.Time) error {
	// Start from the day after last run (or 7 days ago if no last run)
	if lastRun.IsZero() {
		lastRun = now.Add(-7 * 24 * time.Hour)
	}

	// Truncate to the start of the day
	currentDay := lastRun.Truncate(24 * time.Hour)
	endDay := now.Truncate(24 * time.Hour)

	for currentDay.Before(endDay) {
		nextDay := currentDay.Add(24 * time.Hour)

		// Get results for this day
		results, err := a.store.GetResultsByPeriod(monitor, currentDay, nextDay)
		if err != nil {
			return fmt.Errorf("failed to get results for day %s: %w", currentDay, err)
		}

		// Skip if no results
		if len(results) == 0 {
			currentDay = nextDay
			continue
		}

		// Calculate aggregate
		agg := a.calculateAggregate(monitor, "day", currentDay, nextDay, results)

		// Store aggregate
		if err := a.store.StoreAggregate(agg); err != nil {
			return fmt.Errorf("failed to store daily aggregate: %w", err)
		}

		currentDay = nextDay
	}

	return nil
}

// calculateAggregate computes aggregate statistics from a set of results
func (a *Aggregator) calculateAggregate(monitor, periodType string, start, end time.Time, results []*models.MonitorResult) *models.AggregateResult {
	agg := &models.AggregateResult{
		Monitor:     monitor,
		PeriodStart: start,
		PeriodEnd:   end,
		PeriodType:  periodType,
		TotalChecks: len(results),
	}

	if len(results) == 0 {
		return agg
	}

	// Initialize min/max
	agg.MinDuration = results[0].Duration
	agg.MaxDuration = results[0].Duration

	var totalDuration time.Duration

	for _, result := range results {
		// Count up/down
		if result.Status == models.StatusUp {
			agg.UpChecks++
		} else if result.Status == models.StatusDown {
			agg.DownChecks++
		}

		// Track duration stats
		totalDuration += result.Duration
		if result.Duration < agg.MinDuration {
			agg.MinDuration = result.Duration
		}
		if result.Duration > agg.MaxDuration {
			agg.MaxDuration = result.Duration
		}
	}

	// Calculate averages
	if agg.TotalChecks > 0 {
		agg.AvgDuration = totalDuration / time.Duration(agg.TotalChecks)
		agg.UptimePercent = float64(agg.UpChecks) / float64(agg.TotalChecks) * 100.0
	}

	return agg
}

// getLastAggregationTime retrieves the timestamp of the last successful aggregation
func (a *Aggregator) getLastAggregationTime() time.Time {
	data, err := a.store.GetMetadata("aggregator:last_run")
	if err != nil || data == nil {
		return time.Time{} // Return zero time if not found
	}

	var timestamp time.Time
	if err := json.Unmarshal(data, &timestamp); err != nil {
		a.logger.WithComponent("aggregator").
			WithError(err).
			Warn("Failed to parse last aggregation time")
		return time.Time{}
	}

	return timestamp
}

// setLastAggregationTime stores the timestamp of the last successful aggregation
func (a *Aggregator) setLastAggregationTime(t time.Time) {
	data, err := json.Marshal(t)
	if err != nil {
		a.logger.WithComponent("aggregator").
			WithError(err).
			Error("Failed to marshal aggregation time")
		return
	}

	if err := a.store.SetMetadata("aggregator:last_run", data); err != nil {
		a.logger.WithComponent("aggregator").
			WithError(err).
			Error("Failed to store last aggregation time")
	}
}

// GetAggregatesByPeriod returns aggregated data for a monitor within a time period
func (a *Aggregator) GetAggregatesByPeriod(monitor string, start, end time.Time, periodType string) ([]*models.AggregateResult, error) {
	return a.store.GetAggregatesByPeriod(monitor, start, end, periodType)
}

// GetAggregatedMetrics returns metrics for dashboard charts
func (a *Aggregator) GetAggregatedMetrics(monitor string, start, end time.Time, granularity string) ([]AggregatorDataPoint, error) {
	// Determine the period type based on granularity
	var periodType string
	switch granularity {
	case "1m", "5m", "15m":
		periodType = "hour" // Use hourly aggregates for minute granularity
	case "1h":
		periodType = "hour"
	case "1d":
		periodType = "day"
	default:
		periodType = "hour"
	}

	aggregates, err := a.GetAggregatesByPeriod(monitor, start, end, periodType)
	if err != nil {
		return nil, err
	}

	var dataPoints []AggregatorDataPoint
	for _, agg := range aggregates {
		dataPoints = append(dataPoints, AggregatorDataPoint{
			Timestamp: agg.PeriodStart,
			Value:     float64(agg.AvgDuration.Milliseconds()),
		})
	}

	return dataPoints, nil
}

// AggregatorDataPoint represents a data point for aggregation
type AggregatorDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}
