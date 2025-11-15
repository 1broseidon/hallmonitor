package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// BadgerStore manages persistent storage of monitor results using BadgerDB
type BadgerStore struct {
	db            *badger.DB
	logger        *logging.Logger
	retentionDays int
}

const (
	resultKeyPrefix    = "result"
	latestKeyPrefix    = "latest"
	aggregateKeyPrefix = "agg"
	metaKeyPrefix      = "meta"
	timestampKeyWidth  = 20
)

func formatTimestampKey(ts int64) string {
	return fmt.Sprintf("%0*d", timestampKeyWidth, ts)
}

// NewBadgerStore creates a new BadgerDB-backed storage
func NewBadgerStore(path string, retentionDays int, logger *logging.Logger) (*BadgerStore, error) {
	if retentionDays <= 0 {
		retentionDays = 30 // default to 30 days
	}

	opts := badger.DefaultOptions(path)
	opts.Logger = &badgerLogger{logger: logger}

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db: %w", err)
	}

	store := &BadgerStore{
		db:            db,
		logger:        logger,
		retentionDays: retentionDays,
	}

	// Start garbage collection
	go store.runGC()

	logger.WithComponent("storage").
		WithFields(map[string]interface{}{
			"path":          path,
			"retentionDays": retentionDays,
		}).
		Info("BadgerDB storage initialized")

	return store, nil
}

// StoreResult stores a monitor result with TTL
func (bs *BadgerStore) StoreResult(result *models.MonitorResult) error {
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}

	// Generate key: result:{monitor_name}:{unix_nano_timestamp}
	key := fmt.Sprintf("%s:%s:%s", resultKeyPrefix, result.Monitor, formatTimestampKey(result.Timestamp.UnixNano()))

	// Marshal result to JSON
	value, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	// Calculate TTL
	ttl := time.Duration(bs.retentionDays) * 24 * time.Hour

	// Store with TTL
	err = bs.db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), value).WithTTL(ttl)
		return txn.SetEntry(entry)
	})

	if err != nil {
		return fmt.Errorf("failed to store result: %w", err)
	}

	// Also update the latest result cache
	latestKey := fmt.Sprintf("%s:%s", latestKeyPrefix, result.Monitor)
	err = bs.db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(latestKey), value).WithTTL(ttl)
		return txn.SetEntry(entry)
	})

	if err != nil {
		bs.logger.WithComponent("storage").
			WithError(err).
			Warn("Failed to update latest result cache")
	}

	return nil
}

// GetLatestResult retrieves the most recent result for a monitor
func (bs *BadgerStore) GetLatestResult(monitor string) (*models.MonitorResult, error) {
	latestKey := fmt.Sprintf("%s:%s", latestKeyPrefix, monitor)

	var result *models.MonitorResult
	err := bs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(latestKey))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result = &models.MonitorResult{}
			return json.Unmarshal(val, result)
		})
	})

	if err == badger.ErrKeyNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest result: %w", err)
	}

	return result, nil
}

// GetResults retrieves results for a monitor within a time range
func (bs *BadgerStore) GetResults(monitor string, start, end time.Time, limit int) ([]*models.MonitorResult, error) {
	if limit <= 0 {
		limit = 1000 // default limit
	}

	prefix := []byte(fmt.Sprintf("%s:%s:", resultKeyPrefix, monitor))
	startKey := []byte(fmt.Sprintf("%s:%s:%s", resultKeyPrefix, monitor, formatTimestampKey(start.UnixNano())))
	endKey := []byte(fmt.Sprintf("%s:%s:%s", resultKeyPrefix, monitor, formatTimestampKey(end.UnixNano())))

	var results []*models.MonitorResult

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix
		it := txn.NewIterator(opts)
		defer it.Close()

		// Seek to start position
		for it.Seek(startKey); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.Key()

			// Check if we've exceeded the end key
			if bytes.Compare(key, endKey) > 0 {
				break
			}

			// Check if we've hit the limit
			if len(results) >= limit {
				break
			}

			err := item.Value(func(val []byte) error {
				var result models.MonitorResult
				if err := json.Unmarshal(val, &result); err != nil {
					return err
				}
				results = append(results, &result)
				return nil
			})

			if err != nil {
				bs.logger.WithComponent("storage").
					WithError(err).
					Warn("Failed to unmarshal result")
				continue
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get results: %w", err)
	}

	return results, nil
}

// GetResultsByPeriod retrieves all results for a monitor within a time range (for aggregation)
func (bs *BadgerStore) GetResultsByPeriod(monitor string, start, end time.Time) ([]*models.MonitorResult, error) {
	// Use a large limit for aggregation purposes
	return bs.GetResults(monitor, start, end, 100000)
}

// StoreAggregate stores an aggregate result
func (bs *BadgerStore) StoreAggregate(agg *models.AggregateResult) error {
	if agg == nil {
		return fmt.Errorf("aggregate cannot be nil")
	}

	// Generate key: agg:{type}:{monitor_name}:{period_timestamp}
	var key string
	var ttl time.Duration

	if agg.PeriodType == "hour" {
		key = fmt.Sprintf("%s:hour:%s:%s", aggregateKeyPrefix, agg.Monitor, formatTimestampKey(agg.PeriodStart.Unix()))
		// Hourly aggregates kept for 2x retention period
		ttl = time.Duration(bs.retentionDays*2) * 24 * time.Hour
	} else if agg.PeriodType == "day" {
		key = fmt.Sprintf("%s:day:%s:%s", aggregateKeyPrefix, agg.Monitor, formatTimestampKey(agg.PeriodStart.Unix()))
		// Daily aggregates kept for 1 year
		ttl = 365 * 24 * time.Hour
	} else {
		return fmt.Errorf("invalid period type: %s", agg.PeriodType)
	}

	// Marshal aggregate to JSON
	value, err := json.Marshal(agg)
	if err != nil {
		return fmt.Errorf("failed to marshal aggregate: %w", err)
	}

	// Store with TTL
	err = bs.db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), value).WithTTL(ttl)
		return txn.SetEntry(entry)
	})

	if err != nil {
		return fmt.Errorf("failed to store aggregate: %w", err)
	}

	return nil
}

// GetAggregates retrieves aggregates for a monitor within a time range
func (bs *BadgerStore) GetAggregates(monitor string, periodType string, start, end time.Time) ([]*models.AggregateResult, error) {
	if periodType != "hour" && periodType != "day" {
		return nil, fmt.Errorf("invalid period type: %s", periodType)
	}

	prefix := []byte(fmt.Sprintf("%s:%s:%s:", aggregateKeyPrefix, periodType, monitor))
	startKey := []byte(fmt.Sprintf("%s:%s:%s:%s", aggregateKeyPrefix, periodType, monitor, formatTimestampKey(start.Unix())))
	endKey := []byte(fmt.Sprintf("%s:%s:%s:%s", aggregateKeyPrefix, periodType, monitor, formatTimestampKey(end.Unix())))

	var aggregates []*models.AggregateResult

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(startKey); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.Key()

			// Check if we've exceeded the end key
			if bytes.Compare(key, endKey) > 0 {
				break
			}

			err := item.Value(func(val []byte) error {
				var agg models.AggregateResult
				if err := json.Unmarshal(val, &agg); err != nil {
					return err
				}
				aggregates = append(aggregates, &agg)
				return nil
			})

			if err != nil {
				bs.logger.WithComponent("storage").
					WithError(err).
					Warn("Failed to unmarshal aggregate")
				continue
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get aggregates: %w", err)
	}

	return aggregates, nil
}

// GetAggregatesByPeriod is an alias for GetAggregates for consistency
func (bs *BadgerStore) GetAggregatesByPeriod(monitor string, start, end time.Time, periodType string) ([]*models.AggregateResult, error) {
	return bs.GetAggregates(monitor, periodType, start, end)
}

// GetMonitorNames returns all monitor names that have stored results
func (bs *BadgerStore) GetMonitorNames() ([]string, error) {
	monitorNames := make(map[string]bool)

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // We only need keys
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(resultKeyPrefix + ":")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.Key()

			rest := key[len(prefix):]
			colonIdx := bytes.LastIndexByte(rest, ':')
			if colonIdx <= 0 {
				continue
			}

			monitorName := string(rest[:colonIdx])
			if monitorName != "" {
				monitorNames[monitorName] = true
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get monitor names: %w", err)
	}

	names := make([]string, 0, len(monitorNames))
	for name := range monitorNames {
		names = append(names, name)
	}

	return names, nil
}

// SetMetadata stores metadata (e.g., last aggregation time)
func (bs *BadgerStore) SetMetadata(key string, value []byte) error {
	metaKey := fmt.Sprintf("%s:%s", metaKeyPrefix, key)

	return bs.db.Update(func(txn *badger.Txn) error {
		// Metadata doesn't expire
		return txn.Set([]byte(metaKey), value)
	})
}

// GetMetadata retrieves metadata
func (bs *BadgerStore) GetMetadata(key string) ([]byte, error) {
	metaKey := fmt.Sprintf("%s:%s", metaKeyPrefix, key)

	var value []byte
	err := bs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(metaKey))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			value = append([]byte{}, val...)
			return nil
		})
	})

	if err == badger.ErrKeyNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	return value, nil
}

// Close gracefully closes the database
func (bs *BadgerStore) Close() error {
	bs.logger.WithComponent("storage").Info("Closing BadgerDB")
	return bs.db.Close()
}

// runGC runs garbage collection periodically
func (bs *BadgerStore) runGC() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		err := bs.db.RunValueLogGC(0.5)
		if err != nil && err != badger.ErrNoRewrite {
			bs.logger.WithComponent("storage").
				WithError(err).
				Debug("Garbage collection completed with notice")
		}
	}
}

// badgerLogger adapts our logger to BadgerDB's logger interface
type badgerLogger struct {
	logger *logging.Logger
}

func (bl *badgerLogger) Errorf(format string, args ...interface{}) {
	bl.logger.WithComponent("badger").Errorf(format, args...)
}

func (bl *badgerLogger) Warningf(format string, args ...interface{}) {
	bl.logger.WithComponent("badger").Warnf(format, args...)
}

func (bl *badgerLogger) Infof(format string, args ...interface{}) {
	bl.logger.WithComponent("badger").Infof(format, args...)
}

func (bl *badgerLogger) Debugf(format string, args ...interface{}) {
	bl.logger.WithComponent("badger").Debugf(format, args...)
}
