// Package models defines core data structures for monitors, configurations,
// and results shared across the application.
package models

import (
	"time"
)

// MonitorType represents the type of monitor
type MonitorType string

const (
	MonitorTypePing MonitorType = "ping"
	MonitorTypeHTTP MonitorType = "http"
	MonitorTypeTCP  MonitorType = "tcp"
	MonitorTypeDNS  MonitorType = "dns"
)

// MonitorStatus represents the current status of a monitor
type MonitorStatus string

const (
	StatusUp      MonitorStatus = "up"
	StatusDown    MonitorStatus = "down"
	StatusUnknown MonitorStatus = "unknown"
)

// Monitor represents a single monitoring target
type Monitor struct {
	Type     MonitorType           `yaml:"type" json:"type"`
	Name     string                `yaml:"name" json:"name"`
	Target   string                `yaml:"target,omitempty" json:"target,omitempty"`
	URL      string                `yaml:"url,omitempty" json:"url,omitempty"`
	Query    string                `yaml:"query,omitempty" json:"query,omitempty"`
	Interval time.Duration         `yaml:"interval,omitempty" json:"interval,omitempty"`
	Timeout  time.Duration         `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Enabled  *bool                 `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Headers  map[string]string     `yaml:"headers,omitempty" json:"headers,omitempty"`
	Metrics  *MonitorMetricsConfig `yaml:"metrics,omitempty" json:"metrics,omitempty"`
	Labels   map[string]string     `yaml:"labels,omitempty" json:"labels,omitempty"`

	// Monitor-specific fields
	ExpectedStatus           int       `yaml:"expectedStatus,omitempty" json:"expectedStatus,omitempty"`
	ExpectedResponse         string    `yaml:"expectedResponse,omitempty" json:"expectedResponse,omitempty"`
	QueryType                string    `yaml:"queryType,omitempty" json:"queryType,omitempty"`
	Count                    int       `yaml:"count,omitempty" json:"count,omitempty"`
	Port                     int       `yaml:"port,omitempty" json:"port,omitempty"`
	SSLCertExpiryWarningDays int       `yaml:"sslCertExpiryWarningDays,omitempty" json:"sslCertExpiryWarningDays,omitempty"`
	HistogramBuckets         []float64 `yaml:"histogram_buckets,omitempty" json:"histogram_buckets,omitempty"`
}

// MonitorMetricsConfig configures metrics collection for a monitor
type MonitorMetricsConfig struct {
	Enabled           bool              `yaml:"enabled" json:"enabled"`
	Labels            map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	TrackResponseSize bool              `yaml:"track_response_size,omitempty" json:"track_response_size,omitempty"`
	TrackRCode        bool              `yaml:"track_rcode,omitempty" json:"track_rcode,omitempty"`
	HistogramBuckets  []float64         `yaml:"histogram_buckets,omitempty" json:"histogram_buckets,omitempty"`
}

// MonitorGroup represents a group of related monitors
type MonitorGroup struct {
	Name     string        `yaml:"name" json:"name"`
	Interval time.Duration `yaml:"interval,omitempty" json:"interval,omitempty"`
	Monitors []Monitor     `yaml:"monitors" json:"monitors"`
}

// MonitorResult represents the result of a monitor check
type MonitorResult struct {
	Monitor   string        `json:"monitor"`
	Type      MonitorType   `json:"type"`
	Group     string        `json:"group"`
	Status    MonitorStatus `json:"status"`
	Duration  time.Duration `json:"duration"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Metadata  interface{}   `json:"metadata,omitempty"`

	// Type-specific result data
	HTTPResult *HTTPResult `json:"http_result,omitempty"`
	PingResult *PingResult `json:"ping_result,omitempty"`
	TCPResult  *TCPResult  `json:"tcp_result,omitempty"`
	DNSResult  *DNSResult  `json:"dns_result,omitempty"`
}

// HTTPResult contains HTTP-specific check results
type HTTPResult struct {
	StatusCode    int               `json:"status_code"`
	ResponseTime  time.Duration     `json:"response_time"`
	ResponseSize  int64             `json:"response_size"`
	Headers       map[string]string `json:"headers,omitempty"`
	SSLCertExpiry *time.Time        `json:"ssl_cert_expiry,omitempty"`
}

// PingResult contains ping-specific check results
type PingResult struct {
	PacketsSent     int           `json:"packets_sent"`
	PacketsReceived int           `json:"packets_received"`
	PacketLoss      float64       `json:"packet_loss"`
	MinRTT          time.Duration `json:"min_rtt"`
	MaxRTT          time.Duration `json:"max_rtt"`
	AvgRTT          time.Duration `json:"avg_rtt"`
}

// TCPResult contains TCP-specific check results
type TCPResult struct {
	Port         int           `json:"port"`
	Connected    bool          `json:"connected"`
	ResponseTime time.Duration `json:"response_time"`
}

// DNSResult contains DNS-specific check results
type DNSResult struct {
	QueryType    string        `json:"query_type"`
	ResponseCode int           `json:"response_code"`
	ResponseTime time.Duration `json:"response_time"`
	Answers      []string      `json:"answers,omitempty"`
	ResponseSize int           `json:"response_size"`
}

// AggregateResult represents aggregated monitoring data over a time period
type AggregateResult struct {
	Monitor       string        `json:"monitor"`
	PeriodStart   time.Time     `json:"period_start"`
	PeriodEnd     time.Time     `json:"period_end"`
	PeriodType    string        `json:"period_type"` // "hour" or "day"
	TotalChecks   int           `json:"total_checks"`
	UpChecks      int           `json:"up_checks"`
	DownChecks    int           `json:"down_checks"`
	UptimePercent float64       `json:"uptime_percent"`
	AvgDuration   time.Duration `json:"avg_duration"`
	MinDuration   time.Duration `json:"min_duration"`
	MaxDuration   time.Duration `json:"max_duration"`
}
