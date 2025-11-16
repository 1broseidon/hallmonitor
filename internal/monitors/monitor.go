// Package monitors provides implementations for health check monitors including
// HTTP, DNS, TCP, and ICMP ping monitoring with configurable intervals and timeouts.
package monitors

import (
	"context"
	"strings"
	"time"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/internal/metrics"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// Monitor interface defines the contract for all monitor implementations
type Monitor interface {
	// Check performs the monitor check and returns the result
	Check(ctx context.Context) (*models.MonitorResult, error)

	// GetConfig returns the monitor configuration
	GetConfig() *models.Monitor

	// GetName returns the monitor name
	GetName() string

	// GetType returns the monitor type
	GetType() models.MonitorType

	// GetGroup returns the group name
	GetGroup() string

	// IsEnabled returns whether the monitor is enabled
	IsEnabled() bool

	// Validate validates the monitor configuration
	Validate() error
}

// BaseMonitor provides common functionality for all monitor implementations
type BaseMonitor struct {
	Config  *models.Monitor
	Group   string
	Logger  *logging.Logger
	Metrics *metrics.Metrics
}

// NewBaseMonitor creates a new base monitor
func NewBaseMonitor(config *models.Monitor, group string, logger *logging.Logger, metrics *metrics.Metrics) *BaseMonitor {
	return &BaseMonitor{
		Config:  config,
		Group:   group,
		Logger:  logger,
		Metrics: metrics,
	}
}

// GetConfig returns the monitor configuration
func (b *BaseMonitor) GetConfig() *models.Monitor {
	return b.Config
}

// GetName returns the monitor name
func (b *BaseMonitor) GetName() string {
	return b.Config.Name
}

// GetType returns the monitor type
func (b *BaseMonitor) GetType() models.MonitorType {
	return b.Config.Type
}

// GetGroup returns the group name
func (b *BaseMonitor) GetGroup() string {
	return b.Group
}

// IsEnabled returns whether the monitor is enabled
func (b *BaseMonitor) IsEnabled() bool {
	if b.Config.Enabled == nil {
		return true // Default to enabled
	}
	return *b.Config.Enabled
}

// CreateResult creates a monitor result with common fields populated
func (b *BaseMonitor) CreateResult(status models.MonitorStatus, duration time.Duration, err error) *models.MonitorResult {
	result := &models.MonitorResult{
		Monitor:   b.Config.Name,
		Type:      b.Config.Type,
		Group:     b.Group,
		Status:    status,
		Duration:  duration,
		Timestamp: time.Now(),
	}

	if err != nil {
		result.Error = err.Error()
		result.Status = models.StatusDown
	}

	return result
}

// RecordMetrics records common metrics for a check
func (b *BaseMonitor) RecordMetrics(result *models.MonitorResult) {
	if b.Metrics == nil {
		return
	}

	// Record check metrics
	status := "success"
	if result.Status == models.StatusDown {
		status = "failure"
	}

	b.Metrics.RecordCheck(
		result.Monitor,
		string(result.Type),
		result.Group,
		status,
		result.Duration,
	)

	// Set monitor status
	b.Metrics.SetMonitorStatus(
		result.Monitor,
		string(result.Type),
		result.Group,
		result.Status == models.StatusUp,
	)

	// Record errors if any
	if result.Error != "" {
		errorType := "unknown"
		if result.Error != "" {
			// Simple error categorization
			errorMsg := result.Error
			switch {
			case contains(errorMsg, "timeout"):
				errorType = "timeout"
			case contains(errorMsg, "connection"):
				errorType = "connection"
			case contains(errorMsg, "dns"):
				errorType = "dns"
			case contains(errorMsg, "ssl", "tls", "certificate"):
				errorType = "ssl"
			case contains(errorMsg, "status"):
				errorType = "status"
			default:
				errorType = "unknown"
			}
		}

		b.Metrics.RecordError(
			result.Monitor,
			string(result.Type),
			result.Group,
			errorType,
		)
	}
}

// LogResult logs the monitor check result
func (b *BaseMonitor) LogResult(result *models.MonitorResult) {
	if b.Logger == nil {
		return
	}

	var err error
	if result.Error != "" {
		// Create a simple error for logging
		err = &CheckError{Message: result.Error}
	}

	b.Logger.MonitorCheck(
		result.Monitor,
		string(result.Type),
		result.Group,
		string(result.Status),
		result.Duration,
		err,
	)
}

// CheckError represents a monitor check error
type CheckError struct {
	Message string
}

func (e *CheckError) Error() string {
	return e.Message
}

// MonitorFactory creates monitor instances based on configuration
type MonitorFactory struct {
	logger  *logging.Logger
	metrics *metrics.Metrics
}

// NewMonitorFactory creates a new monitor factory
func NewMonitorFactory(logger *logging.Logger, metrics *metrics.Metrics) *MonitorFactory {
	return &MonitorFactory{
		logger:  logger,
		metrics: metrics,
	}
}

// CreateMonitor creates a monitor instance based on the configuration
func (f *MonitorFactory) CreateMonitor(config *models.Monitor, group string) (Monitor, error) {
	switch config.Type {
	case models.MonitorTypePing:
		return NewPingMonitor(config, group, f.logger, f.metrics)
	case models.MonitorTypeHTTP:
		return NewHTTPMonitor(config, group, f.logger, f.metrics)
	case models.MonitorTypeTCP:
		return NewTCPMonitor(config, group, f.logger, f.metrics)
	case models.MonitorTypeDNS:
		return NewDNSMonitor(config, group, f.logger, f.metrics)
	default:
		return nil, &UnsupportedMonitorTypeError{Type: config.Type}
	}
}

// UnsupportedMonitorTypeError represents an error for unsupported monitor types
type UnsupportedMonitorTypeError struct {
	Type models.MonitorType
}

func (e *UnsupportedMonitorTypeError) Error() string {
	return "unsupported monitor type: " + string(e.Type)
}

// MonitorManager manages multiple monitors
type MonitorManager struct {
	monitors []Monitor
	factory  *MonitorFactory
	logger   *logging.Logger
	metrics  *metrics.Metrics
}

// NewMonitorManager creates a new monitor manager
func NewMonitorManager(logger *logging.Logger, metrics *metrics.Metrics) *MonitorManager {
	return &MonitorManager{
		monitors: make([]Monitor, 0),
		factory:  NewMonitorFactory(logger, metrics),
		logger:   logger,
		metrics:  metrics,
	}
}

// LoadMonitors loads monitors from configuration
func (m *MonitorManager) LoadMonitors(groups []models.MonitorGroup) error {
	var newMonitors []Monitor

	for _, group := range groups {
		for _, monitorConfig := range group.Monitors {
			// Skip disabled monitors
			if monitorConfig.Enabled != nil && !*monitorConfig.Enabled {
				continue
			}

			monitor, err := m.factory.CreateMonitor(&monitorConfig, group.Name)
			if err != nil {
				m.logger.WithComponent(logging.ComponentMonitor).
					WithFields(map[string]interface{}{
						"monitor": monitorConfig.Name,
						"type":    string(monitorConfig.Type),
						"group":   group.Name,
					}).
					WithError(err).
					Error("Failed to create monitor")
				continue
			}

			// Validate monitor configuration
			if err := monitor.Validate(); err != nil {
				m.logger.WithComponent(logging.ComponentMonitor).
					WithFields(map[string]interface{}{
						"monitor": monitorConfig.Name,
						"type":    string(monitorConfig.Type),
						"group":   group.Name,
					}).
					WithError(err).
					Error("Monitor configuration validation failed")
				continue
			}

			newMonitors = append(newMonitors, monitor)
		}
	}

	// Replace current monitors
	m.monitors = newMonitors

	// Update metrics
	m.updateMonitorCountMetrics()

	m.logger.WithComponent(logging.ComponentMonitor).
		WithFields(map[string]interface{}{
			"total_monitors": len(m.monitors),
		}).
		Info("Monitors loaded successfully")

	return nil
}

// GetMonitors returns all loaded monitors
func (m *MonitorManager) GetMonitors() []Monitor {
	return m.monitors
}

// GetMonitorByName returns a monitor by name
func (m *MonitorManager) GetMonitorByName(name string) Monitor {
	for _, monitor := range m.monitors {
		if monitor.GetName() == name {
			return monitor
		}
	}
	return nil
}

// GetMonitorsByGroup returns monitors in a specific group
func (m *MonitorManager) GetMonitorsByGroup(group string) []Monitor {
	var groupMonitors []Monitor
	for _, monitor := range m.monitors {
		if monitor.GetGroup() == group {
			groupMonitors = append(groupMonitors, monitor)
		}
	}
	return groupMonitors
}

// GetGroups returns all unique group names
func (m *MonitorManager) GetGroups() []string {
	groups := make(map[string]bool)
	for _, monitor := range m.monitors {
		groups[monitor.GetGroup()] = true
	}

	var groupList []string
	for group := range groups {
		groupList = append(groupList, group)
	}
	return groupList
}

// updateMonitorCountMetrics updates Prometheus metrics for monitor counts
func (m *MonitorManager) updateMonitorCountMetrics() {
	if m.metrics == nil {
		return
	}

	monitorCounts := make(map[string]int)
	enabledCounts := make(map[string]int)

	for _, monitor := range m.monitors {
		monitorType := string(monitor.GetType())
		monitorCounts[monitorType]++
		if monitor.IsEnabled() {
			enabledCounts[monitorType]++
		}
	}

	m.metrics.UpdateMonitorCounts(monitorCounts, enabledCounts)
}

// Helper function to check if a string contains any of the given substrings (case-insensitive)
func contains(s string, substrings ...string) bool {
	sLower := strings.ToLower(s)
	for _, substring := range substrings {
		if strings.Contains(sLower, strings.ToLower(substring)) {
			return true
		}
	}
	return false
}
