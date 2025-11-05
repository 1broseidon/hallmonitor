package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for Hall Monitor
type Metrics struct {
	// Counters
	ChecksTotal *prometheus.CounterVec
	ErrorsTotal *prometheus.CounterVec
	AlertsTotal *prometheus.CounterVec

	// Gauges
	MonitorUp          *prometheus.GaugeVec
	MonitorsConfigured *prometheus.GaugeVec
	MonitorsEnabled    *prometheus.GaugeVec
	MonitorsRunning    prometheus.Gauge
	ConfigReloads      prometheus.Gauge
	LastConfigReload   prometheus.Gauge

	// Histograms
	CheckDuration    *prometheus.HistogramVec
	HTTPResponseTime *prometheus.HistogramVec
	DNSQueryTime     *prometheus.HistogramVec
	PingRTT          *prometheus.HistogramVec
	TCPConnectTime   *prometheus.HistogramVec

	// Monitor-specific metrics
	HTTPStatusCodes  *prometheus.CounterVec
	DNSResponseCodes *prometheus.CounterVec
	PingPacketLoss   *prometheus.GaugeVec
	SSLCertExpiry    *prometheus.GaugeVec
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(registry prometheus.Registerer) *Metrics {
	m := &Metrics{
		// Counters
		ChecksTotal: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "hallmonitor_checks_total",
				Help: "Total number of monitor checks performed",
			},
			[]string{"monitor", "type", "group", "status"},
		),

		ErrorsTotal: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "hallmonitor_errors_total",
				Help: "Total number of monitor check errors",
			},
			[]string{"monitor", "type", "group", "error_type"},
		),

		AlertsTotal: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "hallmonitor_alerts_total",
				Help: "Total number of alerts fired",
			},
			[]string{"monitor", "type", "group", "severity", "rule"},
		),

		// Gauges
		MonitorUp: promauto.With(registry).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hallmonitor_monitor_up",
				Help: "Whether a monitor is up (1) or down (0)",
			},
			[]string{"monitor", "type", "group"},
		),

		MonitorsConfigured: promauto.With(registry).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hallmonitor_monitors_configured",
				Help: "Number of configured monitors by type",
			},
			[]string{"type"},
		),

		MonitorsEnabled: promauto.With(registry).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hallmonitor_monitors_enabled",
				Help: "Number of enabled monitors by type",
			},
			[]string{"type"},
		),

		MonitorsRunning: promauto.With(registry).NewGauge(
			prometheus.GaugeOpts{
				Name: "hallmonitor_monitors_running",
				Help: "Number of currently running monitor checks",
			},
		),

		ConfigReloads: promauto.With(registry).NewGauge(
			prometheus.GaugeOpts{
				Name: "hallmonitor_config_reloads_total",
				Help: "Total number of configuration reloads",
			},
		),

		LastConfigReload: promauto.With(registry).NewGauge(
			prometheus.GaugeOpts{
				Name: "hallmonitor_last_config_reload_timestamp",
				Help: "Timestamp of the last configuration reload",
			},
		),

		// Histograms with default buckets
		CheckDuration: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "hallmonitor_check_duration_seconds",
				Help:    "Duration of monitor checks in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"monitor", "type", "group"},
		),

		HTTPResponseTime: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "hallmonitor_http_response_time_seconds",
				Help:    "HTTP response time in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"monitor", "group", "method", "status_code"},
		),

		DNSQueryTime: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "hallmonitor_dns_query_time_seconds",
				Help:    "DNS query time in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
			[]string{"monitor", "group", "query_type", "server"},
		),

		PingRTT: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "hallmonitor_ping_rtt_seconds",
				Help:    "Ping round-trip time in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
			[]string{"monitor", "group"},
		),

		TCPConnectTime: promauto.With(registry).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "hallmonitor_tcp_connect_time_seconds",
				Help:    "TCP connection time in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"monitor", "group", "port"},
		),

		// Monitor-specific counters
		HTTPStatusCodes: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "hallmonitor_http_status_codes_total",
				Help: "Total HTTP responses by status code",
			},
			[]string{"monitor", "group", "status_code", "method"},
		),

		DNSResponseCodes: promauto.With(registry).NewCounterVec(
			prometheus.CounterOpts{
				Name: "hallmonitor_dns_response_codes_total",
				Help: "Total DNS responses by response code",
			},
			[]string{"monitor", "group", "rcode", "query_type"},
		),

		// Monitor-specific gauges
		PingPacketLoss: promauto.With(registry).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hallmonitor_ping_packet_loss_percent",
				Help: "Ping packet loss percentage",
			},
			[]string{"monitor", "group"},
		),

		SSLCertExpiry: promauto.With(registry).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hallmonitor_ssl_cert_expiry_seconds",
				Help: "SSL certificate expiry time in seconds from epoch",
			},
			[]string{"monitor", "group", "subject"},
		),
	}

	return m
}

// RecordCheck records a monitor check
func (m *Metrics) RecordCheck(monitor, monitorType, group, status string, duration time.Duration) {
	labels := prometheus.Labels{
		"monitor": monitor,
		"type":    monitorType,
		"group":   group,
		"status":  status,
	}

	m.ChecksTotal.With(labels).Inc()
	m.CheckDuration.With(prometheus.Labels{
		"monitor": monitor,
		"type":    monitorType,
		"group":   group,
	}).Observe(duration.Seconds())
}

// RecordError records a monitor error
func (m *Metrics) RecordError(monitor, monitorType, group, errorType string) {
	m.ErrorsTotal.With(prometheus.Labels{
		"monitor":    monitor,
		"type":       monitorType,
		"group":      group,
		"error_type": errorType,
	}).Inc()
}

// SetMonitorStatus sets the up/down status of a monitor
func (m *Metrics) SetMonitorStatus(monitor, monitorType, group string, up bool) {
	value := 0.0
	if up {
		value = 1.0
	}
	m.MonitorUp.With(prometheus.Labels{
		"monitor": monitor,
		"type":    monitorType,
		"group":   group,
	}).Set(value)
}

// RecordHTTPCheck records HTTP-specific metrics
func (m *Metrics) RecordHTTPCheck(monitor, group, method string, statusCode int, duration time.Duration) {
	statusStr := strconv.Itoa(statusCode)

	m.HTTPResponseTime.With(prometheus.Labels{
		"monitor":     monitor,
		"group":       group,
		"method":      method,
		"status_code": statusStr,
	}).Observe(duration.Seconds())

	m.HTTPStatusCodes.With(prometheus.Labels{
		"monitor":     monitor,
		"group":       group,
		"status_code": statusStr,
		"method":      method,
	}).Inc()
}

// RecordDNSCheck records DNS-specific metrics
func (m *Metrics) RecordDNSCheck(monitor, group, queryType, server string, rcode int, duration time.Duration) {
	m.DNSQueryTime.With(prometheus.Labels{
		"monitor":    monitor,
		"group":      group,
		"query_type": queryType,
		"server":     server,
	}).Observe(duration.Seconds())

	m.DNSResponseCodes.With(prometheus.Labels{
		"monitor":    monitor,
		"group":      group,
		"rcode":      strconv.Itoa(rcode),
		"query_type": queryType,
	}).Inc()
}

// RecordPingCheck records ping-specific metrics
func (m *Metrics) RecordPingCheck(monitor, group string, rtt time.Duration, packetLoss float64) {
	m.PingRTT.With(prometheus.Labels{
		"monitor": monitor,
		"group":   group,
	}).Observe(rtt.Seconds())

	m.PingPacketLoss.With(prometheus.Labels{
		"monitor": monitor,
		"group":   group,
	}).Set(packetLoss)
}

// RecordTCPCheck records TCP-specific metrics
func (m *Metrics) RecordTCPCheck(monitor, group string, port int, duration time.Duration) {
	m.TCPConnectTime.With(prometheus.Labels{
		"monitor": monitor,
		"group":   group,
		"port":    strconv.Itoa(port),
	}).Observe(duration.Seconds())
}

// RecordSSLCertExpiry records SSL certificate expiry
func (m *Metrics) RecordSSLCertExpiry(monitor, group, subject string, expiry time.Time) {
	m.SSLCertExpiry.With(prometheus.Labels{
		"monitor": monitor,
		"group":   group,
		"subject": subject,
	}).Set(float64(expiry.Unix()))
}

// RecordAlert records an alert firing
func (m *Metrics) RecordAlert(monitor, monitorType, group, severity, rule string) {
	m.AlertsTotal.With(prometheus.Labels{
		"monitor":  monitor,
		"type":     monitorType,
		"group":    group,
		"severity": severity,
		"rule":     rule,
	}).Inc()
}

// UpdateMonitorCounts updates the configured and enabled monitor counts
func (m *Metrics) UpdateMonitorCounts(monitorCounts map[string]int, enabledCounts map[string]int) {
	// Reset all counts to 0 first
	for _, monitorType := range []string{"ping", "http", "tcp", "dns"} {
		m.MonitorsConfigured.With(prometheus.Labels{"type": monitorType}).Set(0)
		m.MonitorsEnabled.With(prometheus.Labels{"type": monitorType}).Set(0)
	}

	// Set actual counts
	for monitorType, count := range monitorCounts {
		m.MonitorsConfigured.With(prometheus.Labels{"type": monitorType}).Set(float64(count))
	}

	for monitorType, count := range enabledCounts {
		m.MonitorsEnabled.With(prometheus.Labels{"type": monitorType}).Set(float64(count))
	}
}

// IncrementRunningMonitors increments the running monitors counter
func (m *Metrics) IncrementRunningMonitors() {
	m.MonitorsRunning.Inc()
}

// DecrementRunningMonitors decrements the running monitors counter
func (m *Metrics) DecrementRunningMonitors() {
	m.MonitorsRunning.Dec()
}

// RecordConfigReload records a configuration reload
func (m *Metrics) RecordConfigReload() {
	m.ConfigReloads.Inc()
	m.LastConfigReload.SetToCurrentTime()
}
