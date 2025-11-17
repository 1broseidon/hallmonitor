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

// PostgresStore implements ResultStore using PostgreSQL/TimescaleDB
type PostgresStore struct {
	pool   *pgxpool.Pool
	logger *logging.Logger
	ctx    context.Context
}

// NewPostgresStore creates a PostgreSQL-backed storage
func NewPostgresStore(connString string, logger *logging.Logger) (*PostgresStore, error) {
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

	ps := &PostgresStore{
		pool:   pool,
		logger: logger,
		ctx:    ctx,
	}

	// Initialize schema
	if err := ps.initSchema(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	logger.Info("PostgreSQL storage initialized successfully")
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

	-- Optional: Enable TimescaleDB hypertable if available
	DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
			PERFORM create_hypertable('monitor_results', 'timestamp', if_not_exists => TRUE);
			PERFORM create_hypertable('monitor_aggregates', 'period_start', if_not_exists => TRUE);
		END IF;
	EXCEPTION
		WHEN OTHERS THEN
			-- TimescaleDB not available or error creating hypertables, continue without it
			NULL;
	END $$;
	`

	_, err := ps.pool.Exec(ps.ctx, schema)
	return err
}

// StoreResult stores a monitor result
func (ps *PostgresStore) StoreResult(result *models.MonitorResult) error {
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

	_, err = ps.pool.Exec(ps.ctx, query,
		result.Monitor,
		result.Type,
		result.Status,
		result.Timestamp,
		result.ResponseTime.Milliseconds(),
		result.StatusCode,
		result.ErrorMessage,
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
	var statusCode sql.NullInt32
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

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("no results found for monitor: %s", monitor)
		}
		return nil, fmt.Errorf("failed to get latest result: %w", err)
	}

	result.ResponseTime = time.Duration(responseTimeMs) * time.Millisecond
	if statusCode.Valid {
		result.StatusCode = int(statusCode.Int32)
	}
	if errorMessage.Valid {
		result.ErrorMessage = errorMessage.String
	}

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			ps.logger.WithError(err).Warn("Failed to unmarshal metadata")
		} else {
			result.Metadata = metadata
		}
	}

	return &result, nil
}

// GetResults retrieves results for a monitor within a time range
func (ps *PostgresStore) GetResults(monitor string, start, end time.Time, limit int) ([]*models.MonitorResult, error) {
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
		var statusCode sql.NullInt32
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
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		result.ResponseTime = time.Duration(responseTimeMs) * time.Millisecond
		if statusCode.Valid {
			result.StatusCode = int(statusCode.Int32)
		}
		if errorMessage.Valid {
			result.ErrorMessage = errorMessage.String
		}

		// Unmarshal metadata
		if len(metadataJSON) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
				ps.logger.WithError(err).Warn("Failed to unmarshal metadata")
			} else {
				result.Metadata = metadata
			}
		}

		results = append(results, &result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

// GetAggregates retrieves aggregated results for a monitor
func (ps *PostgresStore) GetAggregates(monitor, periodType string, start, end time.Time) ([]*models.AggregateResult, error) {
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
		var avgMs, minMs, maxMs int64

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
			return nil, fmt.Errorf("failed to scan aggregate: %w", err)
		}

		agg.AvgResponseTime = time.Duration(avgMs) * time.Millisecond
		agg.MinResponseTime = time.Duration(minMs) * time.Millisecond
		agg.MaxResponseTime = time.Duration(maxMs) * time.Millisecond
		aggregates = append(aggregates, &agg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating aggregates: %w", err)
	}

	return aggregates, nil
}

// StoreAggregate stores an aggregate result
func (ps *PostgresStore) StoreAggregate(agg *models.AggregateResult) error {
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
		agg.AvgResponseTime.Milliseconds(),
		agg.MinResponseTime.Milliseconds(),
		agg.MaxResponseTime.Milliseconds(),
	)

	if err != nil {
		return fmt.Errorf("failed to store aggregate: %w", err)
	}

	return nil
}

// GetMonitorNames retrieves all unique monitor names
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
			return nil, fmt.Errorf("failed to scan monitor name: %w", err)
		}
		monitors = append(monitors, monitor)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating monitor names: %w", err)
	}

	return monitors, nil
}

// Close closes the database connection pool
func (ps *PostgresStore) Close() error {
	ps.pool.Close()
	ps.logger.Info("PostgreSQL connection pool closed")
	return nil
}

// Capabilities reports the capabilities of the PostgreSQL backend
func (ps *PostgresStore) Capabilities() BackendCapabilities {
	return BackendCapabilities{
		SupportsAggregation: true,
		SupportsRetention:   true,
		SupportsRawResults:  true,
		ReadOnly:            false,
	}
}

// StartRetentionCleanup starts a background goroutine to clean up old data
func (ps *PostgresStore) StartRetentionCleanup(retentionDays int) {
	if retentionDays <= 0 {
		ps.logger.Info("Retention cleanup disabled (retentionDays <= 0)")
		return
	}

	ps.logger.WithField("retentionDays", retentionDays).Info("Starting retention cleanup task")

	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		// Run cleanup immediately on start
		ps.cleanOldData(retentionDays)

		// Then run daily
		for range ticker.C {
			ps.cleanOldData(retentionDays)
		}
	}()
}

func (ps *PostgresStore) cleanOldData(retentionDays int) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	// Clean monitor results
	query := `DELETE FROM monitor_results WHERE timestamp < $1`
	result, err := ps.pool.Exec(ps.ctx, query, cutoff)
	if err != nil {
		ps.logger.WithError(err).Error("Failed to clean old monitor results")
		return
	}

	rows := result.RowsAffected()
	if rows > 0 {
		ps.logger.WithFields(map[string]interface{}{
			"rows_deleted": rows,
			"cutoff_date":  cutoff.Format(time.RFC3339),
			"table":        "monitor_results",
		}).Info("Cleaned old data")
	}

	// Clean aggregates
	query = `DELETE FROM monitor_aggregates WHERE period_start < $1`
	result, err = ps.pool.Exec(ps.ctx, query, cutoff)
	if err != nil {
		ps.logger.WithError(err).Error("Failed to clean old aggregates")
		return
	}

	rows = result.RowsAffected()
	if rows > 0 {
		ps.logger.WithFields(map[string]interface{}{
			"rows_deleted": rows,
			"cutoff_date":  cutoff.Format(time.RFC3339),
			"table":        "monitor_aggregates",
		}).Info("Cleaned old data")
	}
}

// HealthCheck verifies the database connection is healthy
func (ps *PostgresStore) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return ps.pool.Ping(ctx)
}
