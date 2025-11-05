package api

import (
	"bytes"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// healthHandler handles health check requests
func (s *Server) healthHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "healthy",
		"service": "hallmonitor",
		"version": "1.0.0",
	})
}

// readyHandler handles readiness probe requests
func (s *Server) readyHandler(c *fiber.Ctx) error {
	// Check if the service is ready to accept traffic
	// For now, just return ready if the server is running
	return c.JSON(fiber.Map{
		"status": "ready",
		"checks": fiber.Map{
			"config":   "ok",
			"monitors": "ok",
		},
	})
}

// metricsHandler handles Prometheus metrics endpoint
func (s *Server) metricsHandler(c *fiber.Ctx) error {
	// Set content type for Prometheus metrics
	c.Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// Create a buffer to capture the metrics
	var buf bytes.Buffer

	// Create a fake HTTP request and response writer
	req, _ := http.NewRequest("GET", "/metrics", nil)
	rw := &responseWriter{Buffer: &buf, header: make(http.Header)}

	// Get the Prometheus handler for our custom registry and call it
	gatherer, ok := s.prometheusReg.(prometheus.Gatherer)
	if !ok {
		return c.Status(500).SendString("Error: registry does not implement Gatherer interface")
	}
	handler := promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})
	handler.ServeHTTP(rw, req)

	// Return the captured metrics
	return c.SendString(buf.String())
}

// responseWriter is a simple implementation of http.ResponseWriter for capturing metrics
type responseWriter struct {
	*bytes.Buffer
	header http.Header
}

func (rw *responseWriter) Header() http.Header {
	return rw.header
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	// Do nothing, we don't need to track status codes for metrics
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	return rw.Buffer.Write(data)
}

// getMonitorsHandler returns all monitor statuses
func (s *Server) getMonitorsHandler(c *fiber.Ctx) error {
	monitors := s.monitorManager.GetMonitors()

	var results []MonitorStatus
	for _, monitor := range monitors {
		status := MonitorStatus{
			Name:    monitor.GetName(),
			Type:    string(monitor.GetType()),
			Group:   monitor.GetGroup(),
			Enabled: monitor.IsEnabled(),
			Status:  "unknown",
		}

		// Get latest result from scheduler
		if latestResult := s.scheduler.GetLatestResult(monitor.GetName()); latestResult != nil {
			status.Status = string(latestResult.Status)
			timestamp := latestResult.Timestamp.Format("2006-01-02T15:04:05Z07:00")
			duration := latestResult.Duration.String()
			status.LastCheck = &timestamp
			status.Duration = &duration
			if latestResult.Error != "" {
				status.Error = &latestResult.Error
			}
			status.Metadata = latestResult.Metadata
		}

		results = append(results, status)
	}

	return c.JSON(fiber.Map{
		"monitors": results,
		"total":    len(results),
	})
}

// getMonitorHandler returns specific monitor status
func (s *Server) getMonitorHandler(c *fiber.Ctx) error {
	name := c.Params("name")
	monitor := s.monitorManager.GetMonitorByName(name)

	if monitor == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Monitor not found",
		})
	}

	status := MonitorStatus{
		Name:    monitor.GetName(),
		Type:    string(monitor.GetType()),
		Group:   monitor.GetGroup(),
		Enabled: monitor.IsEnabled(),
		Status:  "unknown",
	}

	// Get latest result from scheduler
	if latestResult := s.scheduler.GetLatestResult(monitor.GetName()); latestResult != nil {
		status.Status = string(latestResult.Status)
		timestamp := latestResult.Timestamp.Format("2006-01-02T15:04:05Z07:00")
		duration := latestResult.Duration.String()
		status.LastCheck = &timestamp
		status.Duration = &duration
		if latestResult.Error != "" {
			status.Error = &latestResult.Error
		}
		status.Metadata = latestResult.Metadata
	}

	return c.JSON(status)
}

// getGroupsHandler returns all group statuses
func (s *Server) getGroupsHandler(c *fiber.Ctx) error {
	groups := s.monitorManager.GetGroups()

	var results []GroupStatus
	for _, groupName := range groups {
		monitors := s.monitorManager.GetMonitorsByGroup(groupName)

		status := GroupStatus{
			Name:     groupName,
			Monitors: len(monitors),
			Status:   "unknown", // TODO: Calculate group status
		}
		results = append(results, status)
	}

	return c.JSON(fiber.Map{
		"groups": results,
		"total":  len(results),
	})
}

// getGroupHandler returns specific group status
func (s *Server) getGroupHandler(c *fiber.Ctx) error {
	groupName := c.Params("name")
	monitors := s.monitorManager.GetMonitorsByGroup(groupName)

	if len(monitors) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Group not found",
		})
	}

	var monitorStatuses []MonitorStatus
	for _, monitor := range monitors {
		status := MonitorStatus{
			Name:    monitor.GetName(),
			Type:    string(monitor.GetType()),
			Group:   monitor.GetGroup(),
			Enabled: monitor.IsEnabled(),
			Status:  "unknown",
		}

		// Get latest result from scheduler
		if latestResult := s.scheduler.GetLatestResult(monitor.GetName()); latestResult != nil {
			status.Status = string(latestResult.Status)
			timestamp := latestResult.Timestamp.Format("2006-01-02T15:04:05Z07:00")
			duration := latestResult.Duration.String()
			status.LastCheck = &timestamp
			status.Duration = &duration
			if latestResult.Error != "" {
				status.Error = &latestResult.Error
			}
			status.Metadata = latestResult.Metadata
		}

		monitorStatuses = append(monitorStatuses, status)
	}

	groupStatus := GroupStatus{
		Name:     groupName,
		Monitors: len(monitors),
		Status:   "unknown", // TODO: Calculate group status
	}

	return c.JSON(fiber.Map{
		"group":    groupStatus,
		"monitors": monitorStatuses,
	})
}

// reloadConfigHandler handles configuration reload requests
func (s *Server) reloadConfigHandler(c *fiber.Ctx) error {
	// TODO: Implement config reload
	s.logger.WithComponent("api").Info("Config reload requested")

	return c.JSON(fiber.Map{
		"message": "Configuration reload not yet implemented",
		"status":  "pending",
	})
}

// getConfigHandler returns current configuration (sanitized)
func (s *Server) getConfigHandler(c *fiber.Ctx) error {
	// Return sanitized configuration without sensitive data
	return c.JSON(fiber.Map{
		"server": fiber.Map{
			"port":            s.config.Server.Port,
			"host":            s.config.Server.Host,
			"enableDashboard": s.config.Server.EnableDashboard,
		},
		"metrics": s.config.Metrics,
		"logging": fiber.Map{
			"level":  s.config.Logging.Level,
			"format": s.config.Logging.Format,
		},
		"monitoring": fiber.Map{
			"defaultInterval": s.config.Monitoring.DefaultInterval,
			"defaultTimeout":  s.config.Monitoring.DefaultTimeout,
			"groups":          len(s.config.Monitoring.Groups),
		},
	})
}

// Grafana JSON API handlers
//
// These endpoints provide Grafana JSON Datasource API compatibility.
// Implementation status: PLACEHOLDER
//
// To fully implement:
// 1. grafanaQueryHandler: Query time-series data from result store
// 2. grafanaTagsHandler: Return available tag keys/values for filtering
// 3. grafanaAnnotationsHandler: Return events/annotations for timeline
//
// For now, these return minimal responses to avoid errors in Grafana.
// See: https://grafana.com/grafana/plugins/simpod-json-datasource/

func (s *Server) grafanaQueryHandler(c *fiber.Ctx) error {
	// Placeholder: Return empty result set
	// Future: Query result store and format as time-series data
	return c.JSON([]interface{}{})
}

func (s *Server) grafanaTagsHandler(c *fiber.Ctx) error {
	// Placeholder: Return empty tag list
	// Future: Return monitor names, types, groups as filterable tags
	return c.JSON([]interface{}{})
}

func (s *Server) grafanaAnnotationsHandler(c *fiber.Ctx) error {
	// Placeholder: Return empty annotations
	// Future: Return monitor state changes as timeline annotations
	return c.JSON([]interface{}{})
}

// API response models

// MonitorStatus represents the status of a single monitor
type MonitorStatus struct {
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Group     string      `json:"group"`
	Enabled   bool        `json:"enabled"`
	Status    string      `json:"status"`
	LastCheck *string     `json:"last_check,omitempty"`
	Duration  *string     `json:"duration,omitempty"`
	Error     *string     `json:"error,omitempty"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

// GroupStatus represents the status of a monitor group
type GroupStatus struct {
	Name     string  `json:"name"`
	Monitors int     `json:"monitors"`
	Status   string  `json:"status"`
	Uptime   *string `json:"uptime,omitempty"`
}
