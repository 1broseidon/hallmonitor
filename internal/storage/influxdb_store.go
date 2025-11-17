package storage

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// InfluxDBStore implements ResultStore using InfluxDB
type InfluxDBStore struct {
	client   influxdb2.Client
	writeAPI api.WriteAPI
	queryAPI api.QueryAPI
	bucket   string
	org      string
	logger   *logging.Logger
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
		client:   client,
		writeAPI: client.WriteAPI(org, bucket),
		queryAPI: client.QueryAPI(org),
		bucket:   bucket,
		org:      org,
		logger:   logger,
	}

	// Listen for write errors
	go func() {
		for err := range store.writeAPI.Errors() {
			logger.WithError(err).Error("InfluxDB write error")
		}
	}()

	logger.Info("InfluxDB storage initialized successfully")
	return store, nil
}

// StoreResult stores a monitor result
func (is *InfluxDBStore) StoreResult(result *models.MonitorResult) error {
	// Create point
	p := influxdb2.NewPointWithMeasurement("monitor_result").
		AddTag("monitor", result.Monitor).
		AddTag("type", string(result.Type)).
		AddTag("status", string(result.Status)).
		AddField("response_time_ms", result.ResponseTime.Milliseconds()).
		AddField("status_code", result.StatusCode).
		AddField("error_message", result.ErrorMessage).
		SetTime(result.Timestamp)

	// Add metadata as fields if present
	if result.Metadata != nil {
		for k, v := range result.Metadata {
			p.AddField(fmt.Sprintf("metadata_%s", k), v)
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
		|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
		|> sort(columns: ["_time"], desc: true)
		|> limit(n: 1)
	`, is.bucket, monitor)

	result, err := is.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest result: %w", err)
	}
	defer result.Close()

	if !result.Next() {
		return nil, fmt.Errorf("no results found for monitor: %s", monitor)
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("query error: %w", result.Err())
	}

	record := result.Record()

	monitorResult := &models.MonitorResult{
		Monitor:   getStringValue(record, "monitor"),
		Type:      models.MonitorType(getStringValue(record, "type")),
		Status:    models.MonitorStatus(getStringValue(record, "status")),
		Timestamp: record.Time(),
	}

	if rt := getInt64Value(record, "response_time_ms"); rt > 0 {
		monitorResult.ResponseTime = time.Duration(rt) * time.Millisecond
	}

	if sc := getInt64Value(record, "status_code"); sc > 0 {
		monitorResult.StatusCode = int(sc)
	}

	if em := getStringValue(record, "error_message"); em != "" {
		monitorResult.ErrorMessage = em
	}

	// Extract metadata fields
	metadata := make(map[string]interface{})
	for k, v := range record.Values() {
		if len(k) > 9 && k[:9] == "metadata_" {
			metadata[k[9:]] = v
		}
	}
	if len(metadata) > 0 {
		monitorResult.Metadata = metadata
	}

	return monitorResult, nil
}

// GetResults retrieves results for a monitor within a time range
func (is *InfluxDBStore) GetResults(monitor string, start, end time.Time, limit int) ([]*models.MonitorResult, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "monitor_result")
		|> filter(fn: (r) => r.monitor == "%s")
		|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
		|> sort(columns: ["_time"], desc: true)
		|> limit(n: %d)
	`, is.bucket, start.Format(time.RFC3339), end.Format(time.RFC3339), monitor, limit)

	result, err := is.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query results: %w", err)
	}
	defer result.Close()

	var results []*models.MonitorResult
	for result.Next() {
		record := result.Record()

		mr := &models.MonitorResult{
			Monitor:   getStringValue(record, "monitor"),
			Type:      models.MonitorType(getStringValue(record, "type")),
			Status:    models.MonitorStatus(getStringValue(record, "status")),
			Timestamp: record.Time(),
		}

		if rt := getInt64Value(record, "response_time_ms"); rt > 0 {
			mr.ResponseTime = time.Duration(rt) * time.Millisecond
		}

		if sc := getInt64Value(record, "status_code"); sc > 0 {
			mr.StatusCode = int(sc)
		}

		if em := getStringValue(record, "error_message"); em != "" {
			mr.ErrorMessage = em
		}

		// Extract metadata fields
		metadata := make(map[string]interface{})
		for k, v := range record.Values() {
			if len(k) > 9 && k[:9] == "metadata_" {
				metadata[k[9:]] = v
			}
		}
		if len(metadata) > 0 {
			mr.Metadata = metadata
		}

		results = append(results, mr)
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("query error: %w", result.Err())
	}

	return results, nil
}

// GetAggregates retrieves aggregated results for a monitor
func (is *InfluxDBStore) GetAggregates(monitor, periodType string, start, end time.Time) ([]*models.AggregateResult, error) {
	// Map period type to InfluxDB window duration
	windowDuration := "1h"
	switch periodType {
	case "hourly":
		windowDuration = "1h"
	case "daily":
		windowDuration = "1d"
	case "weekly":
		windowDuration = "7d"
	case "monthly":
		windowDuration = "30d"
	}

	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "monitor_result")
		|> filter(fn: (r) => r.monitor == "%s")
		|> filter(fn: (r) => r._field == "response_time_ms")
		|> window(every: %s)
		|> reduce(
			identity: {total: 0, up: 0, down: 0, sum_rt: 0, min_rt: 0, max_rt: 0},
			fn: (r, accumulator) => ({
				total: accumulator.total + 1,
				up: if r.status == "up" then accumulator.up + 1 else accumulator.up,
				down: if r.status == "down" then accumulator.down + 1 else accumulator.down,
				sum_rt: accumulator.sum_rt + r._value,
				min_rt: if accumulator.min_rt == 0 or r._value < accumulator.min_rt then r._value else accumulator.min_rt,
				max_rt: if r._value > accumulator.max_rt then r._value else accumulator.max_rt
			})
		)
	`, is.bucket, start.Format(time.RFC3339), end.Format(time.RFC3339), monitor, windowDuration)

	result, err := is.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregates: %w", err)
	}
	defer result.Close()

	var aggregates []*models.AggregateResult
	for result.Next() {
		record := result.Record()

		total := getInt64Value(record, "total")
		if total == 0 {
			continue
		}

		agg := &models.AggregateResult{
			Monitor:     monitor,
			PeriodType:  periodType,
			PeriodStart: record.Time(),
			PeriodEnd:   record.Time().Add(parseDuration(windowDuration)),
			TotalChecks: int(total),
			UpChecks:    int(getInt64Value(record, "up")),
			DownChecks:  int(getInt64Value(record, "down")),
		}

		if sumRT := getInt64Value(record, "sum_rt"); sumRT > 0 && total > 0 {
			agg.AvgResponseTime = time.Duration(sumRT/total) * time.Millisecond
		}

		if minRT := getInt64Value(record, "min_rt"); minRT > 0 {
			agg.MinResponseTime = time.Duration(minRT) * time.Millisecond
		}

		if maxRT := getInt64Value(record, "max_rt"); maxRT > 0 {
			agg.MaxResponseTime = time.Duration(maxRT) * time.Millisecond
		}

		aggregates = append(aggregates, agg)
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("query error: %w", result.Err())
	}

	return aggregates, nil
}

// StoreAggregate stores an aggregate result (InfluxDB can compute aggregates on-the-fly)
// This implementation stores pre-computed aggregates in a separate measurement
func (is *InfluxDBStore) StoreAggregate(agg *models.AggregateResult) error {
	p := influxdb2.NewPointWithMeasurement("monitor_aggregate").
		AddTag("monitor", agg.Monitor).
		AddTag("period_type", agg.PeriodType).
		AddField("total_checks", agg.TotalChecks).
		AddField("up_checks", agg.UpChecks).
		AddField("down_checks", agg.DownChecks).
		AddField("avg_response_time_ms", agg.AvgResponseTime.Milliseconds()).
		AddField("min_response_time_ms", agg.MinResponseTime.Milliseconds()).
		AddField("max_response_time_ms", agg.MaxResponseTime.Milliseconds()).
		AddField("period_end", agg.PeriodEnd.Unix()).
		SetTime(agg.PeriodStart)

	is.writeAPI.WritePoint(p)
	return nil
}

// GetMonitorNames retrieves all unique monitor names
func (is *InfluxDBStore) GetMonitorNames() ([]string, error) {
	query := fmt.Sprintf(`
		import "influxdata/influxdb/schema"
		schema.tagValues(bucket: "%s", tag: "monitor")
	`, is.bucket)

	result, err := is.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query monitor names: %w", err)
	}
	defer result.Close()

	var monitors []string
	for result.Next() {
		record := result.Record()
		if val, ok := record.Value().(string); ok {
			monitors = append(monitors, val)
		}
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("query error: %w", result.Err())
	}

	return monitors, nil
}

// Close closes the InfluxDB client
func (is *InfluxDBStore) Close() error {
	is.writeAPI.Flush()
	is.client.Close()
	is.logger.Info("InfluxDB client closed")
	return nil
}

// Capabilities reports the capabilities of the InfluxDB backend
func (is *InfluxDBStore) Capabilities() BackendCapabilities {
	return BackendCapabilities{
		SupportsAggregation: true,
		SupportsRetention:   true, // InfluxDB has built-in retention policies
		SupportsRawResults:  true,
		ReadOnly:            false,
	}
}

// Helper functions to safely extract values from InfluxDB records
func getStringValue(record *api.FluxRecord, key string) string {
	if val, ok := record.ValueByKey(key).(string); ok {
		return val
	}
	return ""
}

func getInt64Value(record *api.FluxRecord, key string) int64 {
	val := record.ValueByKey(key)
	switch v := val.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		// Handle InfluxDB duration format (e.g., "1d", "7d", "30d")
		if len(s) > 1 && s[len(s)-1] == 'd' {
			days := 1
			fmt.Sscanf(s[:len(s)-1], "%d", &days)
			return time.Duration(days) * 24 * time.Hour
		}
		return 0
	}
	return d
}
