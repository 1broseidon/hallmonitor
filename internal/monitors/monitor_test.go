package monitors

import (
	"testing"
	"time"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/internal/metrics"
	"github.com/1broseidon/hallmonitor/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
)

func ptr[T any](v T) *T {
	return &v
}

func setupTestManager(t *testing.T) *MonitorManager {
	t.Helper()

	metricsInstance := metrics.NewMetrics(prometheus.NewRegistry())
	logger, err := logging.InitLogger(logging.Config{
		Level:  "debug",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("failed to init logger: %v", err)
	}

	return NewMonitorManager(logger, metricsInstance)
}

func TestNewMonitorManager(t *testing.T) {
	manager := setupTestManager(t)

	if manager == nil {
		t.Fatal("expected NewMonitorManager to return non-nil manager")
	}

	if manager.monitors == nil {
		t.Error("expected monitors slice to be initialized")
	}

	if manager.factory == nil {
		t.Error("expected factory to be initialized")
	}

	if manager.logger == nil {
		t.Error("expected logger to be initialized")
	}

	if manager.metrics == nil {
		t.Error("expected metrics to be initialized")
	}
}

func TestMonitorManagerLoadMonitors(t *testing.T) {
	manager := setupTestManager(t)

	groups := []models.MonitorGroup{
		{
			Name: "test-group",
			Monitors: []models.Monitor{
				{
					Name:     "http-monitor",
					Type:     "http",
					Enabled:  ptr(true),
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					URL:      "https://example.com",
				},
				{
					Name:     "ping-monitor",
					Type:     "ping",
					Enabled:  ptr(true),
					Interval: 10 * time.Second,
					Timeout:  3 * time.Second,
					Target:   "8.8.8.8",
				},
			},
		},
	}

	err := manager.LoadMonitors(groups)
	if err != nil {
		t.Fatalf("LoadMonitors failed: %v", err)
	}

	monitors := manager.GetMonitors()
	if len(monitors) != 2 {
		t.Errorf("expected 2 monitors, got %d", len(monitors))
	}
}

func TestMonitorManagerLoadMonitorsSkipsDisabled(t *testing.T) {
	manager := setupTestManager(t)

	groups := []models.MonitorGroup{
		{
			Name: "test-group",
			Monitors: []models.Monitor{
				{
					Name:     "enabled-monitor",
					Type:     "http",
					Enabled:  ptr(true),
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					URL:      "https://example.com",
				},
				{
					Name:     "disabled-monitor",
					Type:     "http",
					Enabled:  ptr(false),
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					URL:      "https://disabled.com",
				},
			},
		},
	}

	err := manager.LoadMonitors(groups)
	if err != nil {
		t.Fatalf("LoadMonitors failed: %v", err)
	}

	monitors := manager.GetMonitors()
	if len(monitors) != 1 {
		t.Errorf("expected 1 monitor (disabled should be skipped), got %d", len(monitors))
	}

	if monitors[0].GetName() != "enabled-monitor" {
		t.Errorf("expected enabled-monitor, got %s", monitors[0].GetName())
	}
}

func TestMonitorManagerLoadMonitorsSkipsInvalid(t *testing.T) {
	manager := setupTestManager(t)

	groups := []models.MonitorGroup{
		{
			Name: "test-group",
			Monitors: []models.Monitor{
				{
					Name:     "valid-monitor",
					Type:     "http",
					Enabled:  ptr(true),
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					URL:      "https://example.com",
				},
				{
					Name:     "invalid-monitor",
					Type:     "http",
					Enabled:  ptr(true),
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					// Missing URL - validation should fail
				},
			},
		},
	}

	err := manager.LoadMonitors(groups)
	if err != nil {
		t.Fatalf("LoadMonitors failed: %v", err)
	}

	monitors := manager.GetMonitors()
	if len(monitors) != 1 {
		t.Errorf("expected 1 monitor (invalid should be skipped), got %d", len(monitors))
	}
}

func TestMonitorManagerGetMonitorByName(t *testing.T) {
	manager := setupTestManager(t)

	groups := []models.MonitorGroup{
		{
			Name: "test-group",
			Monitors: []models.Monitor{
				{
					Name:     "test-monitor",
					Type:     "http",
					Enabled:  ptr(true),
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					URL:      "https://example.com",
				},
			},
		},
	}

	manager.LoadMonitors(groups)

	monitor := manager.GetMonitorByName("test-monitor")
	if monitor == nil {
		t.Fatal("expected GetMonitorByName to return monitor")
	}

	if monitor.GetName() != "test-monitor" {
		t.Errorf("expected monitor name 'test-monitor', got %s", monitor.GetName())
	}

	if monitor.GetType() != "http" {
		t.Errorf("expected monitor type 'http', got %s", monitor.GetType())
	}

	if monitor.GetGroup() != "test-group" {
		t.Errorf("expected monitor group 'test-group', got %s", monitor.GetGroup())
	}
}

func TestMonitorManagerGetMonitorByNameNotFound(t *testing.T) {
	manager := setupTestManager(t)

	monitor := manager.GetMonitorByName("nonexistent")
	if monitor != nil {
		t.Error("expected GetMonitorByName to return nil for nonexistent monitor")
	}
}

func TestMonitorManagerGetMonitorsByGroup(t *testing.T) {
	manager := setupTestManager(t)

	groups := []models.MonitorGroup{
		{
			Name: "group-a",
			Monitors: []models.Monitor{
				{
					Name:     "monitor-a1",
					Type:     "http",
					Enabled:  ptr(true),
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					URL:      "https://a1.com",
				},
				{
					Name:     "monitor-a2",
					Type:     "http",
					Enabled:  ptr(true),
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					URL:      "https://a2.com",
				},
			},
		},
		{
			Name: "group-b",
			Monitors: []models.Monitor{
				{
					Name:     "monitor-b1",
					Type:     "http",
					Enabled:  ptr(true),
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					URL:      "https://b1.com",
				},
			},
		},
	}

	manager.LoadMonitors(groups)

	groupAMonitors := manager.GetMonitorsByGroup("group-a")
	if len(groupAMonitors) != 2 {
		t.Errorf("expected 2 monitors in group-a, got %d", len(groupAMonitors))
	}

	groupBMonitors := manager.GetMonitorsByGroup("group-b")
	if len(groupBMonitors) != 1 {
		t.Errorf("expected 1 monitor in group-b, got %d", len(groupBMonitors))
	}

	nonexistentMonitors := manager.GetMonitorsByGroup("nonexistent")
	if len(nonexistentMonitors) != 0 {
		t.Errorf("expected 0 monitors in nonexistent group, got %d", len(nonexistentMonitors))
	}
}

func TestMonitorManagerGetGroups(t *testing.T) {
	manager := setupTestManager(t)

	groups := []models.MonitorGroup{
		{
			Name: "core",
			Monitors: []models.Monitor{
				{
					Name:     "api",
					Type:     "http",
					Enabled:  ptr(true),
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					URL:      "https://api.example.com",
				},
			},
		},
		{
			Name: "services",
			Monitors: []models.Monitor{
				{
					Name:     "db",
					Type:     "tcp",
					Enabled:  ptr(true),
					Interval: 10 * time.Second,
					Timeout:  3 * time.Second,
					Target:   "localhost:5432",
				},
			},
		},
	}

	manager.LoadMonitors(groups)

	groupNames := manager.GetGroups()
	if len(groupNames) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groupNames))
	}

	expectedGroups := map[string]bool{"core": true, "services": true}
	for _, groupName := range groupNames {
		if !expectedGroups[groupName] {
			t.Errorf("unexpected group name: %s", groupName)
		}
	}
}

func TestMonitorFactoryCreateMonitor(t *testing.T) {
	metricsInstance := metrics.NewMetrics(prometheus.NewRegistry())
	logger, _ := logging.InitLogger(logging.Config{Level: "debug", Format: "json"})
	factory := NewMonitorFactory(logger, metricsInstance)

	tests := []struct {
		name        string
		config      *models.Monitor
		group       string
		wantType    models.MonitorType
		expectError bool
	}{
		{
			name: "create http monitor",
			config: &models.Monitor{
				Name:     "http-test",
				Type:     "http",
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
				URL:      "https://example.com",
			},
			group:       "test-group",
			wantType:    models.MonitorTypeHTTP,
			expectError: false,
		},
		{
			name: "create ping monitor",
			config: &models.Monitor{
				Name:     "ping-test",
				Type:     "ping",
				Interval: 10 * time.Second,
				Timeout:  3 * time.Second,
				Target:   "8.8.8.8",
			},
			group:       "test-group",
			wantType:    models.MonitorTypePing,
			expectError: false,
		},
		{
			name: "create tcp monitor",
			config: &models.Monitor{
				Name:     "tcp-test",
				Type:     "tcp",
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
				Target:   "localhost:80",
			},
			group:       "test-group",
			wantType:    models.MonitorTypeTCP,
			expectError: false,
		},
		{
			name: "unsupported monitor type",
			config: &models.Monitor{
				Name:     "unknown-test",
				Type:     "unsupported",
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
			},
			group:       "test-group",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, err := factory.CreateMonitor(tt.config, tt.group)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if monitor == nil {
				t.Fatal("expected monitor to be created")
			}

			if monitor.GetName() != tt.config.Name {
				t.Errorf("expected monitor name %s, got %s", tt.config.Name, monitor.GetName())
			}

			if monitor.GetType() != tt.wantType {
				t.Errorf("expected monitor type %s, got %s", tt.wantType, monitor.GetType())
			}

			if monitor.GetGroup() != tt.group {
				t.Errorf("expected monitor group %s, got %s", tt.group, monitor.GetGroup())
			}

			if !monitor.IsEnabled() {
				t.Error("expected monitor to be enabled by default")
			}

			config := monitor.GetConfig()
			if config == nil {
				t.Error("expected config to be set")
			}
		})
	}
}

func TestBaseMonitorGetters(t *testing.T) {
	metricsInstance := metrics.NewMetrics(prometheus.NewRegistry())
	logger, _ := logging.InitLogger(logging.Config{Level: "debug", Format: "json"})

	config := &models.Monitor{
		Name:     "test-monitor",
		Type:     "http",
		Enabled:  ptr(true),
		Interval: 30 * time.Second,
		Timeout:  5 * time.Second,
		URL:      "https://example.com",
	}

	base := NewBaseMonitor(config, "test-group", logger, metricsInstance)

	if base.GetName() != "test-monitor" {
		t.Errorf("expected name 'test-monitor', got %s", base.GetName())
	}

	if base.GetType() != models.MonitorTypeHTTP {
		t.Errorf("expected type 'http', got %s", base.GetType())
	}

	if base.GetGroup() != "test-group" {
		t.Errorf("expected group 'test-group', got %s", base.GetGroup())
	}

	if !base.IsEnabled() {
		t.Error("expected monitor to be enabled")
	}

	if base.GetConfig() != config {
		t.Error("expected GetConfig to return the same config pointer")
	}
}
