package storage

import (
	"testing"

	"github.com/1broseidon/hallmonitor/internal/config"
	"github.com/1broseidon/hallmonitor/internal/logging"
)

func TestNewStore_NilConfig(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{
		Level:  "error",
		Format: "json",
		Output: "stdout",
	})

	_, err := NewStore(nil, logger)
	if err == nil {
		t.Error("Expected error for nil config")
	}
	if err.Error() != "storage config cannot be nil" {
		t.Errorf("Expected 'storage config cannot be nil', got %v", err)
	}
}

func TestNewStore_NilLogger(t *testing.T) {
	cfg := &config.StorageConfig{
		Backend: "badger",
	}

	_, err := NewStore(cfg, nil)
	if err == nil {
		t.Error("Expected error for nil logger")
	}
	if err.Error() != "logger cannot be nil" {
		t.Errorf("Expected 'logger cannot be nil', got %v", err)
	}
}

func TestNewStore_BackwardCompatibility(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{
		Level:  "error",
		Format: "json",
		Output: "stdout",
	})

	tests := []struct {
		name           string
		config         config.StorageConfig
		wantBackendMsg string // Expected log message fragment
	}{
		{
			name: "old format enabled=true",
			config: config.StorageConfig{
				Enabled:       true,
				Path:          "./data/test.db",
				RetentionDays: 7,
			},
			wantBackendMsg: "BadgerDB",
		},
		{
			name: "old format enabled=false",
			config: config.StorageConfig{
				Enabled: false,
			},
			wantBackendMsg: "NoOp",
		},
		{
			name: "new format badger",
			config: config.StorageConfig{
				Backend: "badger",
				Badger: config.BadgerConfig{
					Enabled:       true,
					Path:          "./data/test.db",
					RetentionDays: 30,
				},
			},
			wantBackendMsg: "BadgerDB",
		},
		{
			name: "new format none",
			config: config.StorageConfig{
				Backend: "none",
			},
			wantBackendMsg: "NoOp",
		},
		{
			name: "badger config fallback to top-level",
			config: config.StorageConfig{
				Backend:       "badger",
				Path:          "./data/fallback.db",
				RetentionDays: 15,
				Badger: config.BadgerConfig{
					Enabled: true,
					// Path and RetentionDays intentionally empty to test fallback
				},
			},
			wantBackendMsg: "BadgerDB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewStore(&tt.config, logger)
			if err != nil {
				t.Fatalf("NewStore() error = %v", err)
			}
			if store == nil {
				t.Fatal("Expected non-nil store")
			}

			// Verify backend type by checking capabilities
			caps := store.Capabilities()
			if tt.wantBackendMsg == "BadgerDB" {
				if !caps.SupportsRawResults {
					t.Error("Expected BadgerDB backend to support raw results")
				}
			} else if tt.wantBackendMsg == "NoOp" {
				if caps.SupportsRawResults {
					t.Error("Expected NoOp backend to not support raw results")
				}
			}

			// Cleanup
			store.Close()
		})
	}
}

func TestNewStore_BadgerBackend(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{
		Level:  "error",
		Format: "json",
		Output: "stdout",
	})

	cfg := &config.StorageConfig{
		Backend: "badger",
		Badger: config.BadgerConfig{
			Enabled:       true,
			Path:          t.TempDir() + "/test.db",
			RetentionDays: 7,
		},
	}

	store, err := NewStore(cfg, logger)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	// Verify it's a BadgerStore by checking capabilities
	caps := store.Capabilities()
	if !caps.SupportsAggregation {
		t.Error("BadgerStore should support aggregation")
	}
	if !caps.SupportsRawResults {
		t.Error("BadgerStore should support raw results")
	}
	if !caps.SupportsRetention {
		t.Error("BadgerStore should support retention")
	}
	if caps.ReadOnly {
		t.Error("BadgerStore should not be read-only")
	}
}

func TestNewStore_NoneBackend(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{
		Level:  "error",
		Format: "json",
		Output: "stdout",
	})

	cfg := &config.StorageConfig{
		Backend: "none",
	}

	store, err := NewStore(cfg, logger)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	// Verify it's a NoOpStore by checking capabilities
	caps := store.Capabilities()
	if caps.SupportsAggregation {
		t.Error("NoOpStore should not support aggregation")
	}
	if caps.SupportsRawResults {
		t.Error("NoOpStore should not support raw results")
	}
	if caps.SupportsRetention {
		t.Error("NoOpStore should not support retention")
	}
	if !caps.ReadOnly {
		t.Error("NoOpStore should be read-only")
	}
}

func TestNewStore_UnknownBackend(t *testing.T) {
	logger, _ := logging.InitLogger(logging.Config{
		Level:  "error",
		Format: "json",
		Output: "stdout",
	})

	cfg := &config.StorageConfig{
		Backend: "unknown-backend",
	}

	_, err := NewStore(cfg, logger)
	if err == nil {
		t.Error("Expected error for unknown backend")
	}
	if err.Error() != "unknown storage backend: unknown-backend (valid options: none, badger, postgres, influxdb)" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestBackendType_Constants(t *testing.T) {
	if BackendNone != "none" {
		t.Errorf("BackendNone = %v, want 'none'", BackendNone)
	}
	if BackendBadger != "badger" {
		t.Errorf("BackendBadger = %v, want 'badger'", BackendBadger)
	}
}
