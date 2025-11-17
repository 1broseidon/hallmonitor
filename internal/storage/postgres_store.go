package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// PostgresStore manages persistent storage of monitor results using PostgreSQL
type PostgresStore struct {
	pool           *pgxpool.Pool
	logger         *logging.Logger
	ctx            context.Context
	retentionDays  int
	stopCleanup    chan struct{}
	cleanupStopped chan struct{}
}

// NewPostgresStore creates a PostgreSQL-backed storage
func NewPostgresStore(connString string, retentionDays int, logger *logging.Logger) (*PostgresStore, error) {
	ctx := context.Background()

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Connection pool settings
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if retentionDays <= 0 {
		retentionDays = 30 // default to 30 days
	}

	ps := &PostgresStore{
		pool:           pool,
		logger:         logger,
		ctx:            ctx,
		retentionDays:  retentionDays,
		stopCleanup:    make(chan struct{}),
		cleanupStopped: make(chan struct{}),
	}

	// Initialize schema
	if err := ps.initSchema(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Start retention cleanup
	go ps.runRetentionCleanup()

	logger.WithComponent("storage").
		WithFields(map[string]interface{}{
			"backend":       "postgres",
			"retentionDays": retentionDays,
		}).
		Info("PostgreSQL storage initialized successfully")

	return ps, nil
}

func (ps *PostgresStore) initSchema() error {
	schema := `
	-- Monitor results table
	CREATE TABLE IF NOT EXISTS monitor_results (
		id BIGSERIAL PRIMARY KEY,
		monitor VARCHAR(255) NOT NULL,
		type VARCHAR(50) NOT NULL,
		status VARCHAR(20) NOT NULL,
		timestamp TIMESTAMPTZ NOT NULL,
		response_time_ms BIGINT,
		status_code INTEGER,
		error_message TEXT,
		metadata JSONB,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Indexes for common queries
	CREATE INDEX IF NOT EXISTS idx_monitor_results_monitor ON monitor_results(monitor);
	CREATE INDEX IF NOT EXISTS idx_monitor_results_timestamp ON monitor_results(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_monitor_results_monitor_timestamp ON monitor_results(monitor, timestamp DESC);

	-- Aggregates table
	CREATE TABLE IF NOT EXISTS monitor_aggregates (
		id BIGSERIAL PRIMARY KEY,
		monitor VARCHAR(255) NOT NULL,
		period_type VARCHAR(20) NOT NULL,
		period_start TIMESTAMPTZ NOT NULL,
		period_end TIMESTAMPTZ NOT NULL,
		total_checks INTEGER NOT NULL,
		up_checks INTEGER NOT NULL,
		down_checks INTEGER NOT NULL,
		avg_response_time_ms BIGINT,
		min_response_time_ms BIGINT,
		max_response_time_ms BIGINT,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		UNIQUE(monitor, period_type, period_start)
	);

	CREATE INDEX IF NOT EXISTS idx_aggregates_monitor_period ON monitor_aggregates(monitor, period_type, period_start DESC);

	-- Metadata table for storing operational metadata
	CREATE TABLE IF NOT EXISTS storage_metadata (
		key VARCHAR(255) PRIMARY KEY,
		value BYTEA NOT NULL,
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);
	`

	_, err := ps.pool.Exec(ps.ctx, schema)
	return err
}

// StoreResult stores a monitor result
func (ps *PostgresStore) StoreResult(result *models.MonitorResult) error {
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}

	query := `
		INSERT INTO monitor_results (monitor, type, status, timestamp, response_time_ms, status_code, error_message, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	// Convert metadata to JSON
	var metadataJSON []byte
	var err error
	if result.Metadata != nil {
		metadataJSON, err = json.Marshal(result.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Extract status code from HTTP result if available
	var statusCode *int
	if result.HTTPResult != nil {
		statusCode = &result.HTTPResult.StatusCode
	}

	_, err = ps.pool.Exec(ps.ctx, query,
		result.Monitor,
		result.Type,
		result.Status,
		result.Timestamp,
		result.Duration.Milliseconds(),
		statusCode,
		result.Error,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to store result: %w", err)
	}

	return nil
}

// GetLatestResult retrieves the most recent result for a monitor
func (ps *PostgresStore) GetLatestResult(monitor string) (*models.MonitorResult, error) {
	query := `
		SELECT monitor, type, status, timestamp, response_time_ms, status_code, error_message, metadata
		FROM monitor_results
		WHERE monitor = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var result models.MonitorResult
	var responseTimeMs int64
	var statusCode sql.NullInt64
	var errorMessage sql.NullString
	var metadataJSON []byte

	err := ps.pool.QueryRow(ps.ctx, query, monitor).Scan(
		&result.Monitor,
		&result.Type,
		&result.Status,
		&result.Timestamp,
		&responseTimeMs,
		&statusCode,
		&errorMessage,
		&metadataJSON,
	)

	if err == sql.ErrNoRows || err != nil && err.Error() == "no rows in result set" {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest result: %w", err)
	}

	result.Duration = time.Duration(responseTimeMs) * time.Millisecond
	if statusCode.Valid {
		// Store HTTP-specific data if status code is present
		result.HTTPResult = &models.HTTPResult{
			StatusCode:   int(statusCode.Int64),
			ResponseTime: result.Duration,
		}
	}
	if errorMessage.Valid {
		result.Error = errorMessage.String
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &result.Metadata); err != nil {
			ps.logger.WithComponent("storage").
				WithError(err).
				Warn("Failed to unmarshal metadata")
		}
	}

	return &result, nil
}

// GetResults retrieves results for a monitor within a time range
func (ps *PostgresStore) GetResults(monitor string, start, end time.Time, limit int) ([]*models.MonitorResult, error) {
	if limit <= 0 {
		limit = 1000 // default limit
	}

	query := `
		SELECT monitor, type, status, timestamp, response_time_ms, status_code, error_message, metadata
		FROM monitor_results
		WHERE monitor = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp DESC
		LIMIT $4
	`

	rows, err := ps.pool.Query(ps.ctx, query, monitor, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query results: %w", err)
	}
	defer rows.Close()

	var results []*models.MonitorResult
	for rows.Next() {
		var result models.MonitorResult
		var responseTimeMs int64
		var statusCode sql.NullInt64
		var errorMessage sql.NullString
		var metadataJSON []byte

		err := rows.Scan(
			&result.Monitor,
			&result.Type,
			&result.Status,
			&result.Timestamp,
			&responseTimeMs,
			&statusCode,
			&errorMessage,
			&metadataJSON,
		)
		if err != nil {
			ps.logger.WithComponent("storage").
				WithError(err).
				Warn("Failed to scan result row")
			continue
		}

		result.Duration = time.Duration(responseTimeMs) * time.Millisecond
		if statusCode.Valid {
			// Store HTTP-specific data if status code is present
			result.HTTPResult = &models.HTTPResult{
				StatusCode:   int(statusCode.Int64),
				ResponseTime: result.Duration,
			}
		}
		if errorMessage.Valid {
			result.Error = errorMessage.String
		}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &result.Metadata); err != nil {
				ps.logger.WithComponent("storage").
					WithError(err).
					Warn("Failed to unmarshal metadata")
			}
		}

		results = append(results, &result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

// GetAggregates retrieves aggregates for a monitor within a time range
func (ps *PostgresStore) GetAggregates(monitor, periodType string, start, end time.Time) ([]*models.AggregateResult, error) {
	if periodType != "hour" && periodType != "day" {
		return nil, fmt.Errorf("invalid period type: %s", periodType)
	}

	query := `
		SELECT monitor, period_type, period_start, period_end, total_checks, up_checks, down_checks,
		       avg_response_time_ms, min_response_time_ms, max_response_time_ms
		FROM monitor_aggregates
		WHERE monitor = $1 AND period_type = $2 AND period_start BETWEEN $3 AND $4
		ORDER BY period_start DESC
	`

	rows, err := ps.pool.Query(ps.ctx, query, monitor, periodType, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregates: %w", err)
	}
	defer rows.Close()

	var aggregates []*models.AggregateResult
	for rows.Next() {
		var agg models.AggregateResult
		var avgMs, minMs, maxMs sql.NullInt64

		err := rows.Scan(
			&agg.Monitor,
			&agg.PeriodType,
			&agg.PeriodStart,
			&agg.PeriodEnd,
			&agg.TotalChecks,
			&agg.UpChecks,
			&agg.DownChecks,
			&avgMs,
			&minMs,
			&maxMs,
		)
		if err != nil {
			ps.logger.WithComponent("storage").
				WithError(err).
				Warn("Failed to scan aggregate row")
			continue
		}

		if avgMs.Valid {
			agg.AvgDuration = time.Duration(avgMs.Int64) * time.Millisecond
		}
		if minMs.Valid {
			agg.MinDuration = time.Duration(minMs.Int64) * time.Millisecond
		}
		if maxMs.Valid {
			agg.MaxDuration = time.Duration(maxMs.Int64) * time.Millisecond
		}

		aggregates = append(aggregates, &agg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating aggregates: %w", err)
	}

	return aggregates, nil
}

// StoreAggregate stores an aggregate result
func (ps *PostgresStore) StoreAggregate(agg *models.AggregateResult) error {
	if agg == nil {
		return fmt.Errorf("aggregate cannot be nil")
	}

	query := `
		INSERT INTO monitor_aggregates (
			monitor, period_type, period_start, period_end, total_checks, up_checks, down_checks,
			avg_response_time_ms, min_response_time_ms, max_response_time_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (monitor, period_type, period_start)
		DO UPDATE SET
			period_end = EXCLUDED.period_end,
			total_checks = EXCLUDED.total_checks,
			up_checks = EXCLUDED.up_checks,
			down_checks = EXCLUDED.down_checks,
			avg_response_time_ms = EXCLUDED.avg_response_time_ms,
			min_response_time_ms = EXCLUDED.min_response_time_ms,
			max_response_time_ms = EXCLUDED.max_response_time_ms
	`

	_, err := ps.pool.Exec(ps.ctx, query,
		agg.Monitor,
		agg.PeriodType,
		agg.PeriodStart,
		agg.PeriodEnd,
		agg.TotalChecks,
		agg.UpChecks,
		agg.DownChecks,
		agg.AvgDuration.Milliseconds(),
		agg.MinDuration.Milliseconds(),
		agg.MaxDuration.Milliseconds(),
	)

	if err != nil {
		return fmt.Errorf("failed to store aggregate: %w", err)
	}

	return nil
}

// GetMonitorNames returns all monitor names that have stored results
func (ps *PostgresStore) GetMonitorNames() ([]string, error) {
	query := `SELECT DISTINCT monitor FROM monitor_results ORDER BY monitor`

	rows, err := ps.pool.Query(ps.ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query monitor names: %w", err)
	}
	defer rows.Close()

	var monitors []string
	for rows.Next() {
		var monitor string
		if err := rows.Scan(&monitor); err != nil {
			ps.logger.WithComponent("storage").
				WithError(err).
				Warn("Failed to scan monitor name")
			continue
		}
		monitors = append(monitors, monitor)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating monitor names: %w", err)
	}

	return monitors, nil
}

// Close gracefully closes the database connection pool
func (ps *PostgresStore) Close() error {
	// Signal cleanup goroutine to stop
	close(ps.stopCleanup)

	// Wait for cleanup to finish (with timeout)
	select {
	case <-ps.cleanupStopped:
		// Cleanup stopped gracefully
	case <-time.After(5 * time.Second):
		ps.logger.WithComponent("storage").Warn("Cleanup goroutine did not stop in time")
	}

	ps.pool.Close()
	ps.logger.WithComponent("storage").Info("PostgreSQL connection pool closed")
	return nil
}

// Capabilities returns the capabilities of the PostgreSQL storage backend
func (ps *PostgresStore) Capabilities() BackendCapabilities {
	return BackendCapabilities{
		SupportsAggregation: true,
		SupportsRetention:   true,
		SupportsRawResults:  true,
		ReadOnly:            false,
	}
}

// SetMetadata stores metadata (e.g., last aggregation time)
func (ps *PostgresStore) SetMetadata(key string, value []byte) error {
	query := `
		INSERT INTO storage_metadata (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key)
		DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
	`

	_, err := ps.pool.Exec(ps.ctx, query, key, value)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
}

// GetMetadata retrieves metadata
func (ps *PostgresStore) GetMetadata(key string) ([]byte, error) {
	query := `SELECT value FROM storage_metadata WHERE key = $1`

	var value []byte
	err := ps.pool.QueryRow(ps.ctx, query, key).Scan(&value)
	if err == sql.ErrNoRows || err != nil && err.Error() == "no rows in result set" {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	return value, nil
}

// runRetentionCleanup runs periodic cleanup of old data
func (ps *PostgresStore) runRetentionCleanup() {
	defer close(ps.cleanupStopped)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run immediately on startup
	ps.cleanOldData()

	for {
		select {
		case <-ticker.C:
			ps.cleanOldData()
		case <-ps.stopCleanup:
			return
		}
	}
}

func (ps *PostgresStore) cleanOldData() {
	cutoff := time.Now().AddDate(0, 0, -ps.retentionDays)

	query := `DELETE FROM monitor_results WHERE timestamp < $1`
	result, err := ps.pool.Exec(ps.ctx, query, cutoff)
	if err != nil {
		ps.logger.WithComponent("storage").
			WithError(err).
			Error("Failed to clean old data")
		return
	}

	rows := result.RowsAffected()
	if rows > 0 {
		ps.logger.WithComponent("storage").
			WithFields(map[string]interface{}{
				"rows_deleted": rows,
				"cutoff_date":  cutoff,
			}).
			Info("Cleaned old monitor results")
	}
}
