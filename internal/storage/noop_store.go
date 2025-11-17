package storage

import (
	"time"

	"github.com/1broseidon/hallmonitor/pkg/models"
)

// NoOpStore is a storage backend that doesn't persist any data.
// It's designed for users who only want Prometheus metrics without historical data.
type NoOpStore struct{}

// NewNoOpStore creates a new no-op storage backend
func NewNoOpStore() *NoOpStore {
	return &NoOpStore{}
}

// StoreResult does nothing (metrics already exported via Prometheus)
func (n *NoOpStore) StoreResult(result *models.MonitorResult) error {
	return nil
}

// GetLatestResult returns ErrNotSupported
func (n *NoOpStore) GetLatestResult(monitor string) (*models.MonitorResult, error) {
	return nil, ErrNotSupported
}

// GetResults returns ErrNotSupported
func (n *NoOpStore) GetResults(monitor string, start, end time.Time, limit int) ([]*models.MonitorResult, error) {
	return nil, ErrNotSupported
}

// GetAggregates returns ErrNotSupported
func (n *NoOpStore) GetAggregates(monitor, periodType string, start, end time.Time) ([]*models.AggregateResult, error) {
	return nil, ErrNotSupported
}

// StoreAggregate does nothing
func (n *NoOpStore) StoreAggregate(agg *models.AggregateResult) error {
	return nil
}

// GetMonitorNames returns an empty list
func (n *NoOpStore) GetMonitorNames() ([]string, error) {
	return []string{}, nil
}

// Close does nothing
func (n *NoOpStore) Close() error {
	return nil
}

// Capabilities returns the capabilities of the no-op storage backend
func (n *NoOpStore) Capabilities() BackendCapabilities {
	return BackendCapabilities{
		SupportsAggregation: false,
		SupportsRetention:   false,
		SupportsRawResults:  false,
		ReadOnly:            true,
	}
}
