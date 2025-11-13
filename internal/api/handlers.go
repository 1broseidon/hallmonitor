package api

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
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

// dashboardHandler serves the basic dashboard HTML
func (s *Server) dashboardHandler(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(dashboardHTML)
}

// dashboardAdvancedHandler serves the advanced dashboard HTML
func (s *Server) dashboardAdvancedHandler(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(dashboardAdvancedHTML)
}

// extractHostnameAndIP extracts hostname and IP address from a target or URL
func extractHostnameAndIP(target, urlStr string) (hostname, ipAddr *string) {
	var host string

	// Try URL first
	if urlStr != "" {
		parsedURL, err := url.Parse(urlStr)
		if err == nil && parsedURL.Host != "" {
			host = parsedURL.Host
			// Remove port if present
			if idx := strings.Index(host, ":"); idx != -1 {
				host = host[:idx]
			}
		}
	}

	// Fall back to target if no URL
	if host == "" && target != "" {
		// Check if target is in format "host:port" or "ip:port"
		if idx := strings.Index(target, ":"); idx != -1 {
			host = target[:idx]
		} else {
			host = target
		}
	}

	if host == "" {
		return nil, nil
	}

	// Check if host is already an IP address
	if net.ParseIP(host) != nil {
		ipAddr = &host
		return nil, ipAddr
	}

	// It's a hostname
	hostname = &host

	// Try to resolve to IP (non-blocking, best effort)
	if ips, err := net.LookupIP(host); err == nil && len(ips) > 0 {
		// Prefer IPv4
		for _, ip := range ips {
			if ip.To4() != nil {
				ipStr := ip.String()
				ipAddr = &ipStr
				break
			}
		}
		// Fall back to IPv6 if no IPv4
		if ipAddr == nil && len(ips) > 0 {
			ipStr := ips[0].String()
			ipAddr = &ipStr
		}
	}

	return hostname, ipAddr
}

// getMonitorsHandler returns all monitor statuses
func (s *Server) getMonitorsHandler(c *fiber.Ctx) error {
	monitors := s.monitorManager.GetMonitors()

	var results []MonitorStatus
	for _, monitor := range monitors {
		config := monitor.GetConfig()
		status := MonitorStatus{
			Name:    monitor.GetName(),
			Type:    string(monitor.GetType()),
			Group:   monitor.GetGroup(),
			Enabled: monitor.IsEnabled(),
			Status:  "unknown",
		}

		// Add configuration details
		if config.Target != "" {
			status.Target = &config.Target
		}
		if config.URL != "" {
			status.URL = &config.URL
		}
		if config.Query != "" {
			status.Query = &config.Query
		}
		if config.QueryType != "" {
			status.QueryType = &config.QueryType
		}
		if config.Interval > 0 {
			intervalStr := config.Interval.String()
			status.Interval = &intervalStr
		}
		if config.Timeout > 0 {
			timeoutStr := config.Timeout.String()
			status.Timeout = &timeoutStr
		}
		if config.Port > 0 {
			status.Port = &config.Port
		}
		if config.Count > 0 {
			status.Count = &config.Count
		}
		if config.ExpectedStatus > 0 {
			status.ExpectedStatus = &config.ExpectedStatus
		}
		if config.ExpectedResponse != "" {
			status.ExpectedResponse = &config.ExpectedResponse
		}
		if len(config.Headers) > 0 {
			status.Headers = config.Headers
		}
		if len(config.Labels) > 0 {
			status.Labels = config.Labels
		}

		// Extract hostname and IP address
		hostname, ipAddr := extractHostnameAndIP(config.Target, config.URL)
		status.Hostname = hostname
		status.IPAddress = ipAddr

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

			// Add type-specific result data
			if latestResult.HTTPResult != nil {
				status.HTTPResult = latestResult.HTTPResult
			}
			if latestResult.PingResult != nil {
				status.PingResult = latestResult.PingResult
			}
			if latestResult.TCPResult != nil {
				status.TCPResult = latestResult.TCPResult
			}
			if latestResult.DNSResult != nil {
				status.DNSResult = latestResult.DNSResult
			}
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

	config := monitor.GetConfig()
	status := MonitorStatus{
		Name:    monitor.GetName(),
		Type:    string(monitor.GetType()),
		Group:   monitor.GetGroup(),
		Enabled: monitor.IsEnabled(),
		Status:  "unknown",
	}

	// Add configuration details
	if config.Target != "" {
		status.Target = &config.Target
	}
	if config.URL != "" {
		status.URL = &config.URL
	}
	if config.Query != "" {
		status.Query = &config.Query
	}
	if config.QueryType != "" {
		status.QueryType = &config.QueryType
	}
	if config.Interval > 0 {
		intervalStr := config.Interval.String()
		status.Interval = &intervalStr
	}
	if config.Timeout > 0 {
		timeoutStr := config.Timeout.String()
		status.Timeout = &timeoutStr
	}
	if config.Port > 0 {
		status.Port = &config.Port
	}
	if config.Count > 0 {
		status.Count = &config.Count
	}
	if config.ExpectedStatus > 0 {
		status.ExpectedStatus = &config.ExpectedStatus
	}
	if config.ExpectedResponse != "" {
		status.ExpectedResponse = &config.ExpectedResponse
	}
	if len(config.Headers) > 0 {
		status.Headers = config.Headers
	}
	if len(config.Labels) > 0 {
		status.Labels = config.Labels
	}

	// Extract hostname and IP address
	hostname, ipAddr := extractHostnameAndIP(config.Target, config.URL)
	status.Hostname = hostname
	status.IPAddress = ipAddr

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

		// Add type-specific result data
		if latestResult.HTTPResult != nil {
			status.HTTPResult = latestResult.HTTPResult
		}
		if latestResult.PingResult != nil {
			status.PingResult = latestResult.PingResult
		}
		if latestResult.TCPResult != nil {
			status.TCPResult = latestResult.TCPResult
		}
		if latestResult.DNSResult != nil {
			status.DNSResult = latestResult.DNSResult
		}
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

	// Configuration details
	Target           *string           `json:"target,omitempty"`
	URL              *string           `json:"url,omitempty"`
	Query            *string           `json:"query,omitempty"`
	QueryType        *string           `json:"query_type,omitempty"`
	Interval         *string           `json:"interval,omitempty"`
	Timeout          *string           `json:"timeout,omitempty"`
	Port             *int              `json:"port,omitempty"`
	Count            *int              `json:"count,omitempty"`
	ExpectedStatus   *int              `json:"expected_status,omitempty"`
	ExpectedResponse *string           `json:"expected_response,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`

	// Extracted/derived fields for easier frontend consumption
	Hostname  *string `json:"hostname,omitempty"`
	IPAddress *string `json:"ip_address,omitempty"`

	// Result-specific data
	HTTPResult interface{} `json:"http_result,omitempty"`
	PingResult interface{} `json:"ping_result,omitempty"`
	TCPResult  interface{} `json:"tcp_result,omitempty"`
	DNSResult  interface{} `json:"dns_result,omitempty"`
}

// GroupStatus represents the status of a monitor group
type GroupStatus struct {
	Name     string  `json:"name"`
	Monitors int     `json:"monitors"`
	Status   string  `json:"status"`
	Uptime   *string `json:"uptime,omitempty"`
}

// getMonitorHistoryHandler returns historical results for a monitor
func (s *Server) getMonitorHistoryHandler(c *fiber.Ctx) error {
	monitorName := c.Params("name")

	// Parse query parameters
	startStr := c.Query("start")
	endStr := c.Query("end")
	limitStr := c.Query("limit", "100")

	// Parse limit
	var limit int
	if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || limit <= 0 {
		limit = 100
	}
	if limit > 10000 {
		limit = 10000 // cap at 10000
	}

	// Parse timestamps
	var start, end time.Time
	var err error

	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   true,
				"message": "Invalid start timestamp format (use RFC3339)",
			})
		}
	} else {
		// Default: last 24 hours
		start = time.Now().Add(-24 * time.Hour)
	}

	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   true,
				"message": "Invalid end timestamp format (use RFC3339)",
			})
		}
	} else {
		end = time.Now()
	}

	// Validate time range
	if end.Before(start) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "End time must be after start time",
		})
	}

	// Get historical results
	results, err := s.scheduler.GetHistoricalResults(monitorName, start, end, limit)
	if err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to get historical results")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to retrieve historical data",
		})
	}

	return c.JSON(fiber.Map{
		"monitor": monitorName,
		"start":   start.Format(time.RFC3339),
		"end":     end.Format(time.RFC3339),
		"results": results,
		"total":   len(results),
	})
}

// getMonitorUptimeHandler returns uptime percentage for a monitor
func (s *Server) getMonitorUptimeHandler(c *fiber.Ctx) error {
	monitorName := c.Params("name")
	periodStr := c.Query("period", "24h")

	// Parse period
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid period format (use duration like 24h, 7d, 30d)",
		})
	}

	// Get results for the period
	start := time.Now().Add(-period)
	end := time.Now()

	results, err := s.scheduler.GetHistoricalResults(monitorName, start, end, 100000)
	if err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to get historical results for uptime")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to calculate uptime",
		})
	}

	// Calculate uptime
	totalChecks := len(results)
	upChecks := 0
	for _, result := range results {
		if result.Status == models.StatusUp {
			upChecks++
		}
	}

	uptimePercent := 0.0
	if totalChecks > 0 {
		uptimePercent = float64(upChecks) / float64(totalChecks) * 100.0
	}

	return c.JSON(fiber.Map{
		"monitor":        monitorName,
		"period":         periodStr,
		"start":          start.Format(time.RFC3339),
		"end":            end.Format(time.RFC3339),
		"total_checks":   totalChecks,
		"up_checks":      upChecks,
		"down_checks":    totalChecks - upChecks,
		"uptime_percent": uptimePercent,
	})
}
