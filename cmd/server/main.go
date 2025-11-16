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

	// Initialize storage if enabled
	var server *api.Server
	if cfg.Storage.Enabled {
		logger.Info("Initializing persistent storage")

		// Create BadgerDB store
		badgerStore, err := storage.NewBadgerStore(cfg.Storage.Path, cfg.Storage.RetentionDays, logger)
		if err != nil {
			logger.WithError(err).Fatal("Failed to initialize storage")
		}

		// Create aggregator if enabled
		var aggregator *storage.Aggregator
		if cfg.Storage.EnableAggregation {
			aggregator = storage.NewAggregator(badgerStore, logger)
		}

		// Create server with storage
		server = api.NewServerWithStorage(cfg, logger, registry, badgerStore, aggregator, badgerStore)

		logger.WithFields(map[string]interface{}{
			"path":          cfg.Storage.Path,
			"retentionDays": cfg.Storage.RetentionDays,
			"aggregation":   cfg.Storage.EnableAggregation,
		}).Info("Persistent storage enabled")
	} else {
		// Create server without storage
		server = api.NewServer(cfg, logger, registry)
		logger.Info("Running without persistent storage (data will be lost on restart)")
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
