// Hall Monitor server provides continuous health monitoring for HTTP, DNS, TCP,
// and ICMP targets with a web dashboard, metrics, and persistent storage.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/1broseidon/hallmonitor/internal/api"
	"github.com/1broseidon/hallmonitor/internal/config"
	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/internal/storage"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize logger
	logger, err := logging.InitLogger(logging.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
		Fields: cfg.Logging.Fields,
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Create Prometheus registry
	registry := prometheus.NewRegistry()

	// Initialize storage backend
	var server *api.Server
	store, err := storage.NewStore(&cfg.Storage, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize storage backend")
	}

	// Check storage capabilities
	caps := store.Capabilities()

	// Create aggregator if backend supports aggregation and it's enabled
	var aggregator *storage.Aggregator
	enableAggregation := cfg.Storage.EnableAggregation || cfg.Storage.Badger.EnableAggregation
	if caps.SupportsAggregation && enableAggregation {
		if badgerStore, ok := store.(*storage.BadgerStore); ok {
			aggregator = storage.NewAggregator(badgerStore, logger)
			logger.Info("Storage aggregation enabled")
		}
	}

	// Create server with storage
	if caps.SupportsRawResults {
		server = api.NewServerWithStorage(cfg, *configPath, logger, registry, store, aggregator, store)
		logger.WithFields(map[string]interface{}{
			"backend":             cfg.Storage.Backend,
			"supportsRawResults":  caps.SupportsRawResults,
			"supportsAggregation": caps.SupportsAggregation,
		}).Info("Persistent storage enabled")
	} else {
		// Create server without persistent storage (NoOp backend)
		server = api.NewServer(cfg, *configPath, logger, registry)
		logger.Info("Running in metrics-only mode (no persistent storage)")
	}

	// Load monitors from configuration
	if err := server.GetMonitorManager().LoadMonitors(cfg.Monitoring.Groups); err != nil {
		logger.WithError(err).Fatal("Failed to load monitors")
	}

	// Start the monitoring scheduler
	scheduler := server.GetScheduler()
	if err := scheduler.Start(context.Background()); err != nil {
		logger.WithError(err).Fatal("Failed to start scheduler")
	}

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	logger.Info("Hall Monitor started successfully")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Hall Monitor...")

	// Stop the scheduler first
	if err := scheduler.Stop(); err != nil {
		logger.WithError(err).Error("Failed to stop scheduler gracefully")
	}

	// Gracefully shutdown the server
	if err := server.Stop(); err != nil {
		logger.WithError(err).Error("Failed to shutdown server gracefully")
	}

	logger.Info("Hall Monitor stopped")
}
