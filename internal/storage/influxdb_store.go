package storage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/query"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// InfluxDBStore manages persistent storage of monitor results using InfluxDB
type InfluxDBStore struct {
	client     influxdb2.Client
	writeAPI   api.WriteAPI
	queryAPI   api.QueryAPI
	bucket     string
	org        string
	logger     *logging.Logger
	stopErr    chan struct{}
	errStopped chan struct{}
}

// NewInfluxDBStore creates an InfluxDB-backed storage
func NewInfluxDBStore(url, token, org, bucket string, logger *logging.Logger) (*InfluxDBStore, error) {
	client := influxdb2.NewClient(url, token)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("influxdb health check failed: %w", err)
	}

	if health.Status != "pass" {
		client.Close()
		return nil, fmt.Errorf("influxdb not healthy: %s", health.Status)
	}

	store := &InfluxDBStore{
		client:     client,
		writeAPI:   client.WriteAPI(org, bucket),
		queryAPI:   client.QueryAPI(org),
		bucket:     bucket,
		org:        org,
		logger:     logger,
		stopErr:    make(chan struct{}),
		errStopped: make(chan struct{}),
	}

	// Start error listener
	go store.listenForWriteErrors()

	logger.WithComponent("storage").
		WithFields(map[string]interface{}{
			"backend": "influxdb",
			"url":     url,
			"org":     org,
			"bucket":  bucket,
		}).
		Info("InfluxDB storage initialized successfully")

	return store, nil
}

// listenForWriteErrors handles async write errors
func (is *InfluxDBStore) listenForWriteErrors() {
	defer close(is.errStopped)

	for {
		select {
		case err := <-is.writeAPI.Errors():
			is.logger.WithComponent("storage").
				WithError(err).
				Error("InfluxDB write error")
		case <-is.stopErr:
			return
		}
	}
}

// StoreResult stores a monitor result
func (is *InfluxDBStore) StoreResult(result *models.MonitorResult) error {
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}

	// Create point with tags and fields
	p := influxdb2.NewPointWithMeasurement("monitor_result").
		AddTag("monitor", result.Monitor).
		AddTag("type", string(result.Type)).
		AddTag("status", string(result.Status)).
		AddField("response_time_ms", result.Duration.Milliseconds()).
		SetTime(result.Timestamp)

	// Add optional fields
	if result.HTTPResult != nil && result.HTTPResult.StatusCode > 0 {
		p.AddField("status_code", result.HTTPResult.StatusCode)
	}
	if result.Error != "" {
		p.AddField("error_message", result.Error)
	}

	// Add metadata as tags if it's a map (InfluxDB works better with tags for filtering)
	if result.Metadata != nil {
		if metaMap, ok := result.Metadata.(map[string]interface{}); ok {
			for k, v := range metaMap {
				// Convert metadata to string tags
				p.AddTag(fmt.Sprintf("meta_%s", k), fmt.Sprintf("%v", v))
			}
		}
	}

	// Write point (async)
	is.writeAPI.WritePoint(p)

	return nil
}

// GetLatestResult retrieves the most recent result for a monitor
func (is *InfluxDBStore) GetLatestResult(monitor string) (*models.MonitorResult, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: -24h)
		|> filter(fn: (r) => r._measurement == "monitor_result")
		|> filter(fn: (r) => r.monitor == "%s")
		|> sort(columns: ["_time"], desc: true)
		|> limit(n: 1)
		|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
	`, is.bucket, escapeFluxString(monitor))

	result, err := is.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest result: %w", err)
	}
	defer result.Close()

	if !result.Next() {
		if result.Err() != nil {
			return nil, fmt.Errorf("query error: %w", result.Err())
		}
		return nil, nil // No results found
	}

	record := result.Record()
	return is.recordToMonitorResult(record), nil
}

// GetResults retrieves results for a monitor within a time range
func (is *InfluxDBStore) GetResults(monitor string, start, end time.Time, limit int) ([]*models.MonitorResult, error) {
	if limit <= 0 {
		limit = 1000 // default limit
	}

	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "monitor_result")
		|> filter(fn: (r) => r.monitor == "%s")
		|> sort(columns: ["_time"], desc: true)
		|> limit(n: %d)
		|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
	`, is.bucket, start.Format(time.RFC3339), end.Format(time.RFC3339), escapeFluxString(monitor), limit)

	queryResult, err := is.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query results: %w", err)
	}
	defer queryResult.Close()

	var results []*models.MonitorResult
	for queryResult.Next() {
		record := queryResult.Record()
		results = append(results, is.recordToMonitorResult(record))
	}

	if queryResult.Err() != nil {
		return nil, fmt.Errorf("query error: %w", queryResult.Err())
	}

	return results, nil
}

// GetAggregates retrieves aggregates for a monitor within a time range
func (is *InfluxDBStore) GetAggregates(monitor, periodType string, start, end time.Time) ([]*models.AggregateResult, error) {
	if periodType != "hour" && periodType != "day" {
		return nil, fmt.Errorf("invalid period type: %s", periodType)
	}

	// Determine window duration for Flux
	var window string
	if periodType == "hour" {
		window = "1h"
	} else {
		window = "1d"
	}

	// Use Flux to compute aggregations on the fly
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "monitor_result")
		|> filter(fn: (r) => r.monitor == "%s")
		|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
		|> window(every: %s)
		|> group(columns: ["_start", "_stop", "monitor"])
		|> reduce(
			identity: {
				total: 0,
				up: 0,
				down: 0,
				sum_rt: 0,
				min_rt: 9999999,
				max_rt: 0
			},
			fn: (r, accumulator) => ({
				total: accumulator.total + 1,
				up: accumulator.up + (if r.status == "up" then 1 else 0),
				down: accumulator.down + (if r.status == "down" then 1 else 0),
				sum_rt: accumulator.sum_rt + r.response_time_ms,
				min_rt: if r.response_time_ms < accumulator.min_rt then r.response_time_ms else accumulator.min_rt,
				max_rt: if r.response_time_ms > accumulator.max_rt then r.response_time_ms else accumulator.max_rt,
				monitor: r.monitor
			})
		)
	`, is.bucket, start.Format(time.RFC3339), end.Format(time.RFC3339), escapeFluxString(monitor), window)

	queryResult, err := is.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregates: %w", err)
	}
	defer queryResult.Close()

	var aggregates []*models.AggregateResult
	for queryResult.Next() {
		record := queryResult.Record()

		total := getInt64FromRecord(record, "total")
		up := getInt64FromRecord(record, "up")
		down := getInt64FromRecord(record, "down")
		sumRT := getInt64FromRecord(record, "sum_rt")
		minRT := getInt64FromRecord(record, "min_rt")
		maxRT := getInt64FromRecord(record, "max_rt")

		var avgRT int64
		if total > 0 {
			avgRT = sumRT / total
		}

		agg := &models.AggregateResult{
			Monitor:     monitor,
			PeriodType:  periodType,
			PeriodStart: record.Start(),
			PeriodEnd:   record.Stop(),
			TotalChecks: int(total),
			UpChecks:    int(up),
			DownChecks:  int(down),
			AvgDuration: time.Duration(avgRT) * time.Millisecond,
			MinDuration: time.Duration(minRT) * time.Millisecond,
			MaxDuration: time.Duration(maxRT) * time.Millisecond,
		}

		aggregates = append(aggregates, agg)
	}

	if queryResult.Err() != nil {
		return nil, fmt.Errorf("query error: %w", queryResult.Err())
	}

	// Sort by period start descending
	sort.Slice(aggregates, func(i, j int) bool {
		return aggregates[i].PeriodStart.After(aggregates[j].PeriodStart)
	})

	return aggregates, nil
}

// StoreAggregate stores an aggregate result
// Note: InfluxDB computes aggregates on-the-fly, so this is a no-op
// However, we implement it for interface compliance
func (is *InfluxDBStore) StoreAggregate(agg *models.AggregateResult) error {
	// InfluxDB computes aggregates dynamically from raw data
	// Storing pre-computed aggregates is not necessary and not recommended
	// This is a no-op for interface compliance
	return nil
}

// GetMonitorNames returns all monitor names that have stored results
func (is *InfluxDBStore) GetMonitorNames() ([]string, error) {
	query := fmt.Sprintf(`
		import "influxdata/influxdb/schema"

		schema.tagValues(
			bucket: "%s",
			tag: "monitor",
			predicate: (r) => r._measurement == "monitor_result",
			start: -30d
		)
	`, is.bucket)

	queryResult, err := is.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query monitor names: %w", err)
	}
	defer queryResult.Close()

	monitorMap := make(map[string]bool)
	for queryResult.Next() {
		record := queryResult.Record()
		if value, ok := record.Value().(string); ok {
			monitorMap[value] = true
		}
	}

	if queryResult.Err() != nil {
		return nil, fmt.Errorf("query error: %w", queryResult.Err())
	}

	monitors := make([]string, 0, len(monitorMap))
	for monitor := range monitorMap {
		monitors = append(monitors, monitor)
	}

	sort.Strings(monitors)
	return monitors, nil
}

// Close gracefully closes the InfluxDB client
func (is *InfluxDBStore) Close() error {
	// Signal error listener to stop
	close(is.stopErr)

	// Wait for error listener to finish (with timeout)
	select {
	case <-is.errStopped:
		// Error listener stopped gracefully
	case <-time.After(2 * time.Second):
		is.logger.WithComponent("storage").Warn("Error listener did not stop in time")
	}

	// Flush any pending writes
	is.writeAPI.Flush()

	// Close client
	is.client.Close()

	is.logger.WithComponent("storage").Info("InfluxDB client closed")
	return nil
}

// Capabilities returns the capabilities of the InfluxDB storage backend
func (is *InfluxDBStore) Capabilities() BackendCapabilities {
	return BackendCapabilities{
		SupportsAggregation: true, // Can use Flux aggregation functions
		SupportsRetention:   true, // Built-in retention policies
		SupportsRawResults:  true,
		ReadOnly:            false,
	}
}

// recordToMonitorResult converts a Flux query record to a MonitorResult
func (is *InfluxDBStore) recordToMonitorResult(record *query.FluxRecord) *models.MonitorResult {
	result := &models.MonitorResult{
		Timestamp: record.Time(),
	}

	// Extract tags
	if monitor, ok := record.ValueByKey("monitor").(string); ok {
		result.Monitor = monitor
	}
	if monitorType, ok := record.ValueByKey("type").(string); ok {
		result.Type = models.MonitorType(monitorType)
	}
	if status, ok := record.ValueByKey("status").(string); ok {
		result.Status = models.MonitorStatus(status)
	}

	// Extract fields
	if rt, ok := record.ValueByKey("response_time_ms").(int64); ok {
		result.Duration = time.Duration(rt) * time.Millisecond
	}
	if sc, ok := record.ValueByKey("status_code").(int64); ok {
		// Store HTTP-specific data if status code is present
		result.HTTPResult = &models.HTTPResult{
			StatusCode:   int(sc),
			ResponseTime: result.Duration,
		}
	}
	if em, ok := record.ValueByKey("error_message").(string); ok {
		result.Error = em
	}

	// Extract metadata from meta_ tags
	metadata := make(map[string]interface{})
	for key, value := range record.Values() {
		if strings.HasPrefix(key, "meta_") {
			metaKey := strings.TrimPrefix(key, "meta_")
			metadata[metaKey] = value
		}
	}
	if len(metadata) > 0 {
		result.Metadata = metadata
	}

	return result
}

// escapeFluxString escapes special characters in Flux query strings
func escapeFluxString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// getInt64FromRecord safely extracts an int64 value from a record
func getInt64FromRecord(record *query.FluxRecord, key string) int64 {
	if val, ok := record.ValueByKey(key).(int64); ok {
		return val
	}
	if val, ok := record.ValueByKey(key).(int); ok {
		return int64(val)
	}
	return 0
}
