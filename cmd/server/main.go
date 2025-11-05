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

	// Create and start the server
	server := api.NewServer(cfg, logger, registry)

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
