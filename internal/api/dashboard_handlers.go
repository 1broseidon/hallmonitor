package api

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// DashboardOverviewResponse represents the overall dashboard overview
type DashboardOverviewResponse struct {
	TotalMonitors    int             `json:"total_monitors"`
	EnabledMonitors  int             `json:"enabled_monitors"`
	DisabledMonitors int             `json:"disabled_monitors"`
	UpMonitors       int             `json:"up_monitors"`
	DownMonitors     int             `json:"down_monitors"`
	UnknownMonitors  int             `json:"unknown_monitors"`
	OverallUptime    float64         `json:"overall_uptime"`
	LastUpdated      time.Time       `json:"last_updated"`
	Groups           []GroupOverview `json:"groups"`
	RecentAlerts     []RecentAlert   `json:"recent_alerts"`
}

// GroupOverview represents a summary of a monitor group
type GroupOverview struct {
	Name         string  `json:"name"`
	MonitorCount int     `json:"monitor_count"`
	UpCount      int     `json:"up_count"`
	DownCount    int     `json:"down_count"`
	Uptime       float64 `json:"uptime"`
}

// RecentAlert represents a recent alert/state change
type RecentAlert struct {
	Timestamp   time.Time `json:"timestamp"`
	MonitorName string    `json:"monitor_name"`
	OldStatus   string    `json:"old_status"`
	NewStatus   string    `json:"new_status"`
	Message     string    `json:"message"`
}

// TimeSeriesRequest represents a request for time series data
type TimeSeriesRequest struct {
	MonitorName string    `json:"monitor_name,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Metric      string    `json:"metric"`      // "response_time", "status", "error_rate"
	Granularity string    `json:"granularity"` // "1m", "5m", "1h", "1d"
}

// TimeSeriesResponse represents time series data response
type TimeSeriesResponse struct {
	MonitorName string      `json:"monitor_name,omitempty"`
	Metric      string      `json:"metric"`
	Granularity string      `json:"granularity"`
	DataPoints  []DataPoint `json:"data_points"`
}

// DataPoint represents a single data point in time series
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Status    string    `json:"status,omitempty"`
}

// InsightsResponse represents analytical insights about monitors
type InsightsResponse struct {
	MonitorName string    `json:"monitor_name"`
	Period      string    `json:"period"`
	Trend       string    `json:"trend"` // "improving", "degrading", "stable"
	AvgResponse float64   `json:"avg_response"`
	P95Response float64   `json:"p95_response"`
	ErrorRate   float64   `json:"error_rate"`
	Anomalies   []Anomaly `json:"anomalies"`
}

// Anomaly represents an detected anomaly in monitoring data
type Anomaly struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`     // "high_latency", "frequent_errors", "downtime"
	Severity  string    `json:"severity"` // "low", "medium", "high"`
	Message   string    `json:"message"`
}

// dashboardOverviewHandler returns comprehensive dashboard overview
func (s *Server) dashboardOverviewHandler(c *fiber.Ctx) error {
	monitors := s.monitorManager.GetMonitors()

	response := DashboardOverviewResponse{
		LastUpdated: time.Now(),
	}

	// Calculate monitor statistics
	total := len(monitors)
	response.TotalMonitors = total

	for _, monitor := range monitors {
		if monitor.IsEnabled() {
			response.EnabledMonitors++
		} else {
			response.DisabledMonitors++
		}

		// Get latest result
		if latestResult := s.scheduler.GetLatestResult(monitor.GetName()); latestResult != nil {
			switch latestResult.Status {
			case models.StatusUp:
				response.UpMonitors++
			case models.StatusDown:
				response.DownMonitors++
			default:
				response.UnknownMonitors++
			}
		} else {
			response.UnknownMonitors++
		}
	}

	// Calculate overall uptime for last 24 hours
	uptime24h := s.calculateOverallUptime(24 * time.Hour)
	response.OverallUptime = uptime24h

	// Get group overview
	response.Groups = s.getGroupOverviews()

	// Get recent alerts (placeholder - would need to implement state change tracking)
	response.RecentAlerts = []RecentAlert{} // TODO: Implement alert tracking

	return c.JSON(response)
}

// dashboardTimeSeriesHandler returns time series data for charts
func (s *Server) dashboardTimeSeriesHandler(c *fiber.Ctx) error {
	if s.aggregator == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   true,
			"message": "Time series data not available - aggregator not initialized",
		})
	}

	// Parse query parameters
	monitorName := c.Query("monitor")
	metric := c.Query("metric", "response_time")
	granularity := c.Query("granularity", "5m")
	startStr := c.Query("start", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	endStr := c.Query("end", time.Now().Format(time.RFC3339))

	// Parse timestamps
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid start timestamp format (use RFC3339)",
		})
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid end timestamp format (use RFC3339)",
		})
	}

	if monitorName == "" {
		// Return time series for all monitors (aggregate view)
		return s.getAggregateTimeSeries(c, metric, granularity, start, end)
	}

	// Return time series for specific monitor
	return s.getMonitorTimeSeries(c, monitorName, metric, granularity, start, end)
}

// dashboardInsightsHandler returns analytical insights
func (s *Server) dashboardInsightsHandler(c *fiber.Ctx) error {
	periodStr := c.Query("period", "24h")
	monitorName := c.Query("monitor")

	// Parse period
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid period format (use duration like 24h, 7d, 30d)",
		})
	}

	if monitorName != "" {
		// Return insights for specific monitor
		insight, err := s.calculateMonitorInsights(monitorName, period)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   true,
				"message": "Failed to calculate insights",
			})
		}
		return c.JSON(insight)
	}

	// Return insights for all monitors
	monitorList := s.monitorManager.GetMonitors()
	var insights []InsightsResponse

	for _, monitor := range monitorList {
		insight, err := s.calculateMonitorInsights(monitor.GetName(), period)
		if err != nil {
			s.logger.WithComponent(logging.ComponentAPI).
				WithError(err).
				WithFields(map[string]interface{}{"monitor": monitor.GetName()}).
				Error("Failed to calculate insights for monitor")
			continue
		}
		insights = append(insights, insight)
	}

	return c.JSON(insights)
}

// Helper functions

func (s *Server) calculateOverallUptime(period time.Duration) float64 {
	monitors := s.monitorManager.GetMonitors()
	start := time.Now().Add(-period)
	end := time.Now()

	totalChecks := 0
	upChecks := 0

	for _, monitor := range monitors {
		if !monitor.IsEnabled() {
			continue
		}

		results, err := s.scheduler.GetHistoricalResults(monitor.GetName(), start, end, 100000)
		if err != nil {
			continue
		}

		totalChecks += len(results)
		for _, result := range results {
			if result.Status == models.StatusUp {
				upChecks++
			}
		}
	}

	if totalChecks == 0 {
		return 0.0
	}

	return float64(upChecks) / float64(totalChecks) * 100.0
}

func (s *Server) getGroupOverviews() []GroupOverview {
	groups := s.monitorManager.GetGroups()
	var overviews []GroupOverview

	for _, groupName := range groups {
		monitors := s.monitorManager.GetMonitorsByGroup(groupName)

		overview := GroupOverview{
			Name:         groupName,
			MonitorCount: len(monitors),
		}

		// Calculate uptime for the group
		start := time.Now().Add(-24 * time.Hour)
		end := time.Now()

		totalChecks := 0
		upChecks := 0

		for _, monitor := range monitors {
			if !monitor.IsEnabled() {
				continue
			}

			// Check current status
			if latestResult := s.scheduler.GetLatestResult(monitor.GetName()); latestResult != nil {
				switch latestResult.Status {
				case models.StatusUp:
					overview.UpCount++
				case models.StatusDown:
					overview.DownCount++
				}
			}

			// Get historical results for uptime calculation
			results, err := s.scheduler.GetHistoricalResults(monitor.GetName(), start, end, 100000)
			if err != nil {
				continue
			}

			totalChecks += len(results)
			for _, result := range results {
				if result.Status == models.StatusUp {
					upChecks++
				}
			}
		}

		if totalChecks > 0 {
			overview.Uptime = float64(upChecks) / float64(totalChecks) * 100.0
		}

		overviews = append(overviews, overview)
	}

	return overviews
}

func (s *Server) getMonitorTimeSeries(c *fiber.Ctx, monitorName, metric, granularity string, start, end time.Time) error {
	// Get historical results for the monitor
	results, err := s.scheduler.GetHistoricalResults(monitorName, start, end, 100000)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to retrieve time series data",
		})
	}

	// Convert results to time series data points based on metric
	var dataPoints []DataPoint
	for _, result := range results {
		var value float64
		switch metric {
		case "response_time":
			value = float64(result.Duration.Milliseconds())
		case "status":
			value = 0 // We'll use status field instead
		case "error_rate":
			value = 0 // Will need to aggregate over time windows
		}

		status := string(result.Status)
		dataPoints = append(dataPoints, DataPoint{
			Timestamp: result.Timestamp,
			Value:     value,
			Status:    status,
		})
	}

	// TODO: Implement data aggregation based on granularity
	// For now, return raw data points

	response := TimeSeriesResponse{
		MonitorName: monitorName,
		Metric:      metric,
		Granularity: granularity,
		DataPoints:  dataPoints,
	}

	return c.JSON(response)
}

func (s *Server) getAggregateTimeSeries(c *fiber.Ctx, metric, granularity string, start, end time.Time) error {
	// Return aggregate time series across all monitors
	// TODO: Implement multi-monitor aggregation logic
	return c.JSON(TimeSeriesResponse{
		Metric:      metric,
		Granularity: granularity,
		DataPoints:  []DataPoint{},
	})
}

func (s *Server) calculateMonitorInsights(monitorName string, period time.Duration) (InsightsResponse, error) {
	start := time.Now().Add(-period)
	end := time.Now()

	results, err := s.scheduler.GetHistoricalResults(monitorName, start, end, 100000)
	if err != nil {
		return InsightsResponse{}, err
	}

	insight := InsightsResponse{
		MonitorName: monitorName,
		Period:      period.String(),
	}

	if len(results) == 0 {
		return insight, nil
	}

	// Calculate response time statistics
	var durations []time.Duration
	upCount := 0
	errorCount := 0

	for _, result := range results {
		durations = append(durations, result.Duration)
		if result.Status == models.StatusUp {
			upCount++
		} else if result.Status == models.StatusDown || result.Error != "" {
			errorCount++
		}
	}

	// Calculate average and P95 response times
	var totalDuration time.Duration
	for _, d := range durations {
		totalDuration += d
	}

	insight.AvgResponse = float64(totalDuration.Milliseconds()) / float64(len(durations))
	insight.ErrorRate = float64(errorCount) / float64(len(results)) * 100.0

	// Simple P95 calculation (would need proper implementation in production)
	if len(durations) > 0 {
		// This is a simplified P95 - proper implementation would sort the slice
		insight.P95Response = insight.AvgResponse * 1.5 // Placeholder calculation
	}

	// Determine trend (simplified)
	insight.Trend = "stable" // TODO: Implement proper trend analysis

	// TODO: Implement anomaly detection
	insight.Anomalies = []Anomaly{}

	return insight, nil
}
