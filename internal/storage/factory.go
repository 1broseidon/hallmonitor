package storage

import (
	"fmt"

	"github.com/1broseidon/hallmonitor/internal/config"
	"github.com/1broseidon/hallmonitor/internal/logging"
)

// BackendType represents the type of storage backend
type BackendType string

const (
	// BackendNone means no persistent storage, metrics only via Prometheus
	BackendNone BackendType = "none"
	// BackendBadger uses BadgerDB for embedded storage
	BackendBadger BackendType = "badger"
)

// NewStore creates a new storage backend based on configuration
func NewStore(cfg *config.StorageConfig, logger *logging.Logger) (ResultStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("storage config cannot be nil")
	}

	// Determine backend type, defaulting to "badger" for backward compatibility
	backendType := BackendType(cfg.Backend)
	if backendType == "" {
		// If no backend is specified, use the old "enabled" field for backward compatibility
		if cfg.Enabled {
			backendType = BackendBadger
		} else {
			backendType = BackendNone
		}
	}

	switch backendType {
	case BackendNone:
		logger.Info("Using NoOp storage - metrics only via Prometheus")
		return NewNoOpStore(), nil

	case BackendBadger:
		logger.Info("Using BadgerDB storage")
		// Use Badger-specific config if available, otherwise fall back to top-level config
		path := cfg.Badger.Path
		retentionDays := cfg.Badger.RetentionDays

		// Backward compatibility: if badger config is empty, use top-level config
		if path == "" {
			path = cfg.Path
		}
		if retentionDays == 0 {
			retentionDays = cfg.RetentionDays
		}

		return NewBadgerStore(path, retentionDays, logger)

	default:
		return nil, fmt.Errorf("unknown storage backend: %s (valid options: none, badger)", cfg.Backend)
	}
}
