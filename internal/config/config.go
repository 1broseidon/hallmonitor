package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"

	"github.com/1broseidon/hallmonitor/pkg/models"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `yaml:"server" mapstructure:"server"`
	Metrics    MetricsConfig    `yaml:"metrics" mapstructure:"metrics"`
	Logging    LoggingConfig    `yaml:"logging" mapstructure:"logging"`
	Monitoring MonitoringConfig `yaml:"monitoring" mapstructure:"monitoring"`
	Alerting   AlertingConfig   `yaml:"alerting" mapstructure:"alerting"`
	Webhooks   []WebhookConfig  `yaml:"webhooks" mapstructure:"webhooks"`
}

// ServerConfig contains server configuration
type ServerConfig struct {
	Port            string   `yaml:"port" mapstructure:"port" json:"port"`
	Host            string   `yaml:"host" mapstructure:"host" json:"host"`
	CORSOrigins     []string `yaml:"corsOrigins" mapstructure:"corsOrigins" json:"corsOrigins"`
	EnableDashboard bool     `yaml:"enableDashboard" mapstructure:"enableDashboard" json:"enableDashboard"`
}

// MetricsConfig contains Prometheus metrics configuration
type MetricsConfig struct {
	Enabled               bool   `yaml:"enabled" mapstructure:"enabled"`
	Path                  string `yaml:"path" mapstructure:"path"`
	IncludeProcessMetrics bool   `yaml:"includeProcessMetrics" mapstructure:"includeProcessMetrics"`
	IncludeGoMetrics      bool   `yaml:"includeGoMetrics" mapstructure:"includeGoMetrics"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string            `yaml:"level" mapstructure:"level"`
	Format string            `yaml:"format" mapstructure:"format"`
	Output string            `yaml:"output" mapstructure:"output"`
	Fields map[string]string `yaml:"fields" mapstructure:"fields"`
}

// MonitoringConfig contains monitoring configuration
type MonitoringConfig struct {
	DefaultInterval                 time.Duration         `yaml:"defaultInterval" mapstructure:"defaultInterval"`
	DefaultTimeout                  time.Duration         `yaml:"defaultTimeout" mapstructure:"defaultTimeout"`
	DefaultSSLCertExpiryWarningDays int                   `yaml:"defaultSSLCertExpiryWarningDays" mapstructure:"defaultSSLCertExpiryWarningDays"`
	Groups                          []models.MonitorGroup `yaml:"groups" mapstructure:"groups"`
}

// AlertingConfig contains alerting configuration
type AlertingConfig struct {
	Enabled            bool          `yaml:"enabled" mapstructure:"enabled"`
	EvaluationInterval time.Duration `yaml:"evaluationInterval" mapstructure:"evaluationInterval"`
	Rules              []AlertRule   `yaml:"rules" mapstructure:"rules"`
}

// AlertRule represents an alerting rule
type AlertRule struct {
	Name        string            `yaml:"name" mapstructure:"name"`
	Expr        string            `yaml:"expr" mapstructure:"expr"`
	For         time.Duration     `yaml:"for" mapstructure:"for"`
	Labels      map[string]string `yaml:"labels" mapstructure:"labels"`
	Annotations map[string]string `yaml:"annotations" mapstructure:"annotations"`
}

// WebhookConfig contains webhook configuration
type WebhookConfig struct {
	URL    string   `yaml:"url" mapstructure:"url"`
	Events []string `yaml:"events" mapstructure:"events"`
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", "7878")
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.corsOrigins", []string{"http://localhost:3000", "http://localhost:7878"})
	v.SetDefault("server.enableDashboard", true)
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
	v.SetDefault("metrics.includeProcessMetrics", true)
	v.SetDefault("metrics.includeGoMetrics", true)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("monitoring.defaultInterval", "30s")
	v.SetDefault("monitoring.defaultTimeout", "10s")
	v.SetDefault("monitoring.defaultSSLCertExpiryWarningDays", 30)
	v.SetDefault("alerting.enabled", false)
	v.SetDefault("alerting.evaluationInterval", "10s")

	// Enable environment variable substitution
	v.AutomaticEnv()

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/hallmonitor")
	}

	// Read config
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply defaults to monitors
	for i := range config.Monitoring.Groups {
		group := &config.Monitoring.Groups[i]
		if group.Interval == 0 {
			group.Interval = config.Monitoring.DefaultInterval
		}

		for j := range group.Monitors {
			monitor := &group.Monitors[j]
			if monitor.Interval == 0 {
				monitor.Interval = group.Interval
			}
			if monitor.Timeout == 0 {
				monitor.Timeout = config.Monitoring.DefaultTimeout
			}
			if monitor.SSLCertExpiryWarningDays == 0 {
				monitor.SSLCertExpiryWarningDays = config.Monitoring.DefaultSSLCertExpiryWarningDays
			}
			// Default to enabled if not explicitly set
			if monitor.Enabled == nil {
				enabled := true
				monitor.Enabled = &enabled
			}
		}
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port == "" {
		return fmt.Errorf("server.port is required")
	}

	// Validate monitoring groups
	monitorNames := make(map[string]bool)
	for _, group := range c.Monitoring.Groups {
		if group.Name == "" {
			return fmt.Errorf("group name is required")
		}

		for _, monitor := range group.Monitors {
			if monitor.Name == "" {
				return fmt.Errorf("monitor name is required in group %s", group.Name)
			}

			// Check for duplicate monitor names
			if monitorNames[monitor.Name] {
				return fmt.Errorf("duplicate monitor name: %s", monitor.Name)
			}
			monitorNames[monitor.Name] = true

			// Validate monitor type
			switch monitor.Type {
			case models.MonitorTypePing:
				if monitor.Target == "" {
					return fmt.Errorf("ping monitor %s requires target", monitor.Name)
				}
			case models.MonitorTypeHTTP:
				if monitor.URL == "" {
					return fmt.Errorf("http monitor %s requires url", monitor.Name)
				}
			case models.MonitorTypeTCP:
				if monitor.Target == "" {
					return fmt.Errorf("tcp monitor %s requires target", monitor.Name)
				}
			case models.MonitorTypeDNS:
				if monitor.Target == "" || monitor.Query == "" {
					return fmt.Errorf("dns monitor %s requires target and query", monitor.Name)
				}
			default:
				return fmt.Errorf("invalid monitor type: %s", monitor.Type)
			}

			// Validate timeout and interval
			if monitor.Timeout < 0 {
				return fmt.Errorf("monitor %s has negative timeout: %v", monitor.Name, monitor.Timeout)
			}
			if monitor.Timeout > 5*time.Minute {
				return fmt.Errorf("monitor %s timeout too long (max 5 minutes): %v", monitor.Name, monitor.Timeout)
			}
			if monitor.Interval < 0 {
				return fmt.Errorf("monitor %s has negative interval: %v", monitor.Name, monitor.Interval)
			}
			if monitor.Interval > 0 && monitor.Interval < time.Second {
				return fmt.Errorf("monitor %s interval too short (min 1 second): %v", monitor.Name, monitor.Interval)
			}
		}
	}

	// Validate global defaults
	if c.Monitoring.DefaultTimeout < 0 {
		return fmt.Errorf("monitoring.defaultTimeout cannot be negative")
	}
	if c.Monitoring.DefaultInterval < 0 {
		return fmt.Errorf("monitoring.defaultInterval cannot be negative")
	}
	if c.Monitoring.DefaultInterval > 0 && c.Monitoring.DefaultInterval < time.Second {
		return fmt.Errorf("monitoring.defaultInterval too short (min 1 second)")
	}

	return nil
}
