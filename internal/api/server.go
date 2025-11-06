package api

import (
	_ "embed"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/timeout"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/1broseidon/hallmonitor/internal/config"
	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/internal/metrics"
	"github.com/1broseidon/hallmonitor/internal/monitors"
	"github.com/1broseidon/hallmonitor/internal/scheduler"
)

//go:embed dashboard.html
var dashboardHTML string

//go:embed dashboard_ambient.html
var dashboardAmbientHTML string

// Server represents the API server
type Server struct {
	app            *fiber.App
	config         *config.Config
	logger         *logging.Logger
	metrics        *metrics.Metrics
	monitorManager *monitors.MonitorManager
	scheduler      *scheduler.Scheduler
	prometheusReg  prometheus.Registerer
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, logger *logging.Logger, prometheusReg prometheus.Registerer) *Server {
	// Create metrics instance
	metricsInstance := metrics.NewMetrics(prometheusReg)

	// Create monitor manager
	monitorManager := monitors.NewMonitorManager(logger, metricsInstance)

	// Create scheduler
	schedulerInstance := scheduler.NewScheduler(logger, metricsInstance, monitorManager)

	// Create Fiber app with configuration
	app := fiber.New(fiber.Config{
		AppName:               "Hall Monitor v1.0",
		DisableStartupMessage: false,
		ServerHeader:          "HallMonitor",
		ErrorHandler:          errorHandler(logger),
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           120 * time.Second,
		ReadBufferSize:        8192, // 8KB buffer for request headers (increased from 4KB default to handle proxy headers)
	})

	s := &Server{
		app:            app,
		config:         cfg,
		logger:         logger,
		metrics:        metricsInstance,
		monitorManager: monitorManager,
		scheduler:      schedulerInstance,
		prometheusReg:  prometheusReg,
	}

	// Setup middleware
	s.setupMiddleware()

	// Setup routes
	s.setupRoutes()

	return s
}

// setupMiddleware configures Fiber middleware
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// Request logger middleware
	s.app.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${method} ${path}\n",
		Output: nil, // Will use default (os.Stdout)
	}))

	// CORS middleware
	corsOrigins := "*"
	if len(s.config.Server.CORSOrigins) > 0 {
		corsOrigins = strings.Join(s.config.Server.CORSOrigins, ",")
	}
	s.app.Use(cors.New(cors.Config{
		AllowOrigins: corsOrigins,
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Global timeout middleware
	s.app.Use(timeout.NewWithContext(func(c *fiber.Ctx) error {
		return c.Next()
	}, 30*time.Second))
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health and metrics endpoints
	s.app.Get("/health", s.healthHandler)
	s.app.Get("/ready", s.readyHandler)
	s.app.Get("/metrics", s.metricsHandler)

	// Dashboard (if enabled)
	if s.config.Server.EnableDashboard {
		s.app.Get("/", s.dashboardHandler)
		s.app.Get("/dashboard", s.dashboardHandler)
	}

	// API v1 routes
	api := s.app.Group("/api/v1")

	// Monitor status endpoints
	api.Get("/monitors", s.getMonitorsHandler)
	api.Get("/monitors/:name", s.getMonitorHandler)
	api.Get("/groups", s.getGroupsHandler)
	api.Get("/groups/:name", s.getGroupHandler)

	// Configuration endpoints
	api.Post("/reload", s.reloadConfigHandler)
	api.Get("/config", s.getConfigHandler)

	// Grafana export endpoint
	if s.config.Server.EnableDashboard {
		api.Get("/grafana/dashboard", s.exportGrafanaDashboardHandler)
	}

	// Grafana JSON API endpoints (for datasource compatibility)
	api.Post("/query", s.grafanaQueryHandler)
	api.Post("/query/tags", s.grafanaTagsHandler)
	api.Get("/annotations", s.grafanaAnnotationsHandler)
}

// Start starts the server
func (s *Server) Start() error {
	address := s.config.Server.Host + ":" + s.config.Server.Port

	s.logger.WithComponent(logging.ComponentAPI).
		WithFields(map[string]interface{}{
			"address": address,
		}).
		Info("Starting HTTP server")

	return s.app.Listen(address)
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	s.logger.WithComponent(logging.ComponentAPI).Info("Stopping HTTP server")
	return s.app.Shutdown()
}

// GetMonitorManager returns the monitor manager
func (s *Server) GetMonitorManager() *monitors.MonitorManager {
	return s.monitorManager
}

// GetScheduler returns the scheduler
func (s *Server) GetScheduler() *scheduler.Scheduler {
	return s.scheduler
}

// errorHandler handles Fiber errors
func errorHandler(logger *logging.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError

		// Check if it's a Fiber error
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		// Log the error
		logger.WithComponent(logging.ComponentAPI).
			WithFields(map[string]interface{}{
				"method": c.Method(),
				"path":   c.Path(),
				"status": code,
			}).
			WithError(err).
			Error("HTTP request error")

		// Return error response
		return c.Status(code).JSON(fiber.Map{
			"error":   true,
			"message": err.Error(),
		})
	}
}
