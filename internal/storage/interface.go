package storage

import (
	"errors"
	"time"

	"github.com/1broseidon/hallmonitor/pkg/models"
)

// ResultStore is the interface all storage backends must implement
type ResultStore interface {
	// Core operations
	StoreResult(result *models.MonitorResult) error
	GetLatestResult(monitor string) (*models.MonitorResult, error)
	GetResults(monitor string, start, end time.Time, limit int) ([]*models.MonitorResult, error)

	// Aggregation (optional - return ErrNotSupported if backend doesn't support)
	GetAggregates(monitor, periodType string, start, end time.Time) ([]*models.AggregateResult, error)
	StoreAggregate(agg *models.AggregateResult) error

	// Metadata
	GetMonitorNames() ([]string, error)

	// Lifecycle
	Close() error

	// Capabilities reporting
	Capabilities() BackendCapabilities
}

// BackendCapabilities describes what features a storage backend supports
type BackendCapabilities struct {
	SupportsAggregation bool
	SupportsRetention   bool
	SupportsRawResults  bool
	ReadOnly            bool
}

// ErrNotSupported is returned when a backend doesn't support an operation
var ErrNotSupported = errors.New("operation not supported by this backend")
