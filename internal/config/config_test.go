package config

import (
	"os"
	"testing"
	"time"

	"github.com/1broseidon/hallmonitor/pkg/models"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "hallmonitor-config-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}

	if _, err := file.WriteString(content); err != nil {
		file.Close()
		t.Fatalf("failed to write temp config file: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("failed to close temp config file: %v", err)
	}

	return file.Name()
}

func TestLoadConfigAppliesDefaults(t *testing.T) {
	configYAML := `
monitoring:
  defaultInterval: "1m"
  defaultTimeout: "5s"
  defaultSSLCertExpiryWarningDays: 45
  groups:
    - name: "default"
      monitors:
        - type: "http"
          name: "homepage"
          url: "https://example.com"
`

	path := writeTempConfig(t, configYAML)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.Server.Port != "7878" {
		t.Fatalf("expected default server port 7878, got %s", cfg.Server.Port)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("expected default server host 0.0.0.0, got %s", cfg.Server.Host)
	}

	if got := cfg.Monitoring.DefaultSSLCertExpiryWarningDays; got != 45 {
		t.Fatalf("expected default SSL expiry warning days 45, got %d", got)
	}

	if len(cfg.Monitoring.Groups) != 1 {
		t.Fatalf("expected 1 monitoring group, got %d", len(cfg.Monitoring.Groups))
	}

	group := cfg.Monitoring.Groups[0]
	if group.Interval != time.Minute {
		t.Fatalf("expected group interval to default to 1m, got %s", group.Interval)
	}

	if len(group.Monitors) != 1 {
		t.Fatalf("expected 1 monitor, got %d", len(group.Monitors))
	}

	monitor := group.Monitors[0]
	if monitor.Interval != time.Minute {
		t.Fatalf("expected monitor interval to default to group interval 1m, got %s", monitor.Interval)
	}

	if monitor.Timeout != 5*time.Second {
		t.Fatalf("expected monitor timeout to default to 5s, got %s", monitor.Timeout)
	}

	if monitor.SSLCertExpiryWarningDays != 45 {
		t.Fatalf("expected monitor SSL expiry warning days 45, got %d", monitor.SSLCertExpiryWarningDays)
	}

	if monitor.Enabled == nil || !*monitor.Enabled {
		t.Fatalf("expected monitor to default to enabled")
	}
}

func TestConfigValidateErrors(t *testing.T) {
	baseConfig := &Config{
		Server: ServerConfig{Port: ""},
	}

	if err := baseConfig.Validate(); err == nil {
		t.Fatalf("expected error when server port is missing")
	}

	dupMonitorConfig := &Config{
		Server: ServerConfig{Port: "7878"},
		Monitoring: MonitoringConfig{
			Groups: []models.MonitorGroup{
				{
					Name: "group-a",
					Monitors: []models.Monitor{
						{Type: models.MonitorTypeHTTP, Name: "duplicate", URL: "https://example.com"},
					},
				},
				{
					Name: "group-b",
					Monitors: []models.Monitor{
						{Type: models.MonitorTypeHTTP, Name: "duplicate", URL: "https://example.org"},
					},
				},
			},
		},
	}

	if err := dupMonitorConfig.Validate(); err == nil {
		t.Fatalf("expected duplicate monitor name validation error")
	}

	invalidTimeoutConfig := &Config{
		Server: ServerConfig{Port: "7878"},
		Monitoring: MonitoringConfig{
			DefaultInterval: time.Second,
			DefaultTimeout:  time.Second,
			Groups: []models.MonitorGroup{
				{
					Name: "group",
					Monitors: []models.Monitor{
						{Type: models.MonitorTypeHTTP, Name: "bad-timeout", URL: "https://example.com", Timeout: -1},
					},
				},
			},
		},
	}

	if err := invalidTimeoutConfig.Validate(); err == nil {
		t.Fatalf("expected negative timeout validation error")
	}

	invalidIntervalConfig := &Config{
		Server: ServerConfig{Port: "7878"},
		Monitoring: MonitoringConfig{
			DefaultInterval: time.Second,
			DefaultTimeout:  time.Second,
			Groups: []models.MonitorGroup{
				{
					Name: "group",
					Monitors: []models.Monitor{
						{Type: models.MonitorTypeHTTP, Name: "short-interval", URL: "https://example.com", Interval: 500 * time.Millisecond},
					},
				},
			},
		},
	}

	if err := invalidIntervalConfig.Validate(); err == nil {
		t.Fatalf("expected short interval validation error")
	}
}
