package metrics

import (
	"math"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
)

func newTestMetrics(t *testing.T) (*Metrics, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	return NewMetrics(reg), reg
}

func getHistogram(t *testing.T, reg *prometheus.Registry, name string, labels map[string]string) *dto.Histogram {
	t.Helper()
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	for _, family := range families {
		if family.GetName() != name {
			continue
		}

		for _, metric := range family.Metric {
			if metricMatchesLabels(metric, labels) {
				return metric.GetHistogram()
			}
		}
	}

	return nil
}

func metricMatchesLabels(metric *dto.Metric, labels map[string]string) bool {
	if len(metric.GetLabel()) != len(labels) {
		return false
	}

	for _, lp := range metric.GetLabel() {
		if labels[lp.GetName()] != lp.GetValue() {
			return false
		}
	}

	return true
}

func TestNewMetricsRegistersCollectors(t *testing.T) {
	_, reg := newTestMetrics(t)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	if len(families) == 0 {
		t.Fatalf("expected registered collectors, got none")
	}
}

func TestRecordCheckUpdatesCountersAndHistogram(t *testing.T) {
	metrics, reg := newTestMetrics(t)

	metrics.RecordCheck("homepage", "http", "default", "up", 500*time.Millisecond)

	if got := testutil.ToFloat64(metrics.ChecksTotal.WithLabelValues("homepage", "http", "default", "up")); got != 1 {
		t.Fatalf("expected ChecksTotal counter to be 1, got %v", got)
	}

	hist := getHistogram(t, reg, "hallmonitor_check_duration_seconds", map[string]string{
		"monitor": "homepage",
		"type":    "http",
		"group":   "default",
	})

	if hist == nil {
		t.Fatalf("expected histogram data for check duration")
	}

	if hist.GetSampleCount() != 1 {
		t.Fatalf("expected histogram sample count 1, got %d", hist.GetSampleCount())
	}

	if math.Abs(hist.GetSampleSum()-0.5) > 0.0001 {
		t.Fatalf("expected histogram sum close to 0.5, got %f", hist.GetSampleSum())
	}
}

func TestSetMonitorStatus(t *testing.T) {
	metrics, _ := newTestMetrics(t)

	metrics.SetMonitorStatus("dns", "ping", "network", true)
	if got := testutil.ToFloat64(metrics.MonitorUp.WithLabelValues("dns", "ping", "network")); got != 1 {
		t.Fatalf("expected gauge to be 1 when monitor up, got %v", got)
	}

	metrics.SetMonitorStatus("dns", "ping", "network", false)
	if got := testutil.ToFloat64(metrics.MonitorUp.WithLabelValues("dns", "ping", "network")); got != 0 {
		t.Fatalf("expected gauge to be 0 when monitor down, got %v", got)
	}
}

func TestRecordHTTPCheckUpdatesMetrics(t *testing.T) {
	metrics, reg := newTestMetrics(t)

	metrics.RecordHTTPCheck("homepage", "default", "GET", 204, 200*time.Millisecond)

	if got := testutil.ToFloat64(metrics.HTTPStatusCodes.WithLabelValues("homepage", "default", "204", "GET")); got != 1 {
		t.Fatalf("expected HTTPStatusCodes counter to be 1, got %v", got)
	}

	hist := getHistogram(t, reg, "hallmonitor_http_response_time_seconds", map[string]string{
		"monitor":     "homepage",
		"group":       "default",
		"method":      "GET",
		"status_code": "204",
	})

	if hist == nil {
		t.Fatalf("expected histogram data for HTTP response time")
	}

	if hist.GetSampleCount() != 1 {
		t.Fatalf("expected HTTP histogram sample count 1, got %d", hist.GetSampleCount())
	}
}

func TestRecordDNSCheckUpdatesMetrics(t *testing.T) {
	metrics, reg := newTestMetrics(t)

	metrics.RecordDNSCheck("resolver", "edge", "A", "1.1.1.1", 0, 150*time.Millisecond)

	if got := testutil.ToFloat64(metrics.DNSResponseCodes.WithLabelValues("resolver", "edge", "0", "A")); got != 1 {
		t.Fatalf("expected DNS response counter to be 1, got %v", got)
	}

	hist := getHistogram(t, reg, "hallmonitor_dns_query_time_seconds", map[string]string{
		"monitor":    "resolver",
		"group":      "edge",
		"query_type": "A",
		"server":     "1.1.1.1",
	})

	if hist == nil {
		t.Fatalf("expected histogram data for DNS query time")
	}
}

func TestRecordPingCheckUpdatesMetrics(t *testing.T) {
	metrics, reg := newTestMetrics(t)

	metrics.RecordPingCheck("icmp", "core", 20*time.Millisecond, 12.5)

	if got := testutil.ToFloat64(metrics.PingPacketLoss.WithLabelValues("icmp", "core")); math.Abs(got-12.5) > 0.0001 {
		t.Fatalf("expected packet loss gauge 12.5, got %v", got)
	}

	hist := getHistogram(t, reg, "hallmonitor_ping_rtt_seconds", map[string]string{
		"monitor": "icmp",
		"group":   "core",
	})

	if hist == nil {
		t.Fatalf("expected ping RTT histogram data")
	}
}

func TestRecordTCPCheckUpdatesMetrics(t *testing.T) {
	metrics, reg := newTestMetrics(t)

	metrics.RecordTCPCheck("redis", "cache", 6379, 30*time.Millisecond)

	hist := getHistogram(t, reg, "hallmonitor_tcp_connect_time_seconds", map[string]string{
		"monitor": "redis",
		"group":   "cache",
		"port":    "6379",
	})

	if hist == nil {
		t.Fatalf("expected TCP connect histogram data")
	}
}

func TestRecordSSLCertExpiry(t *testing.T) {
	metrics, _ := newTestMetrics(t)
	expiry := time.Unix(1700000000, 0)

	metrics.RecordSSLCertExpiry("homepage", "default", "example.com", expiry)

	if got := testutil.ToFloat64(metrics.SSLCertExpiry.WithLabelValues("homepage", "default", "example.com")); got != float64(expiry.Unix()) {
		t.Fatalf("expected SSL expiry gauge to equal unix timestamp, got %v", got)
	}
}

func TestRecordAlertIncrementsCounter(t *testing.T) {
	metrics, _ := newTestMetrics(t)

	metrics.RecordAlert("db", "tcp", "infra", "critical", "high-latency")

	if got := testutil.ToFloat64(metrics.AlertsTotal.WithLabelValues("db", "tcp", "infra", "critical", "high-latency")); got != 1 {
		t.Fatalf("expected alert counter to be 1, got %v", got)
	}
}

func TestUpdateMonitorCountsResetsAndSets(t *testing.T) {
	metrics, _ := newTestMetrics(t)

	metrics.UpdateMonitorCounts(map[string]int{"http": 2}, map[string]int{"http": 1})

	if got := testutil.ToFloat64(metrics.MonitorsConfigured.WithLabelValues("http")); got != 2 {
		t.Fatalf("expected configured count 2, got %v", got)
	}

	if got := testutil.ToFloat64(metrics.MonitorsEnabled.WithLabelValues("http")); got != 1 {
		t.Fatalf("expected enabled count 1, got %v", got)
	}

	if got := testutil.ToFloat64(metrics.MonitorsConfigured.WithLabelValues("ping")); got != 0 {
		t.Fatalf("expected ping configured count reset to 0, got %v", got)
	}
}

func TestRunningMonitorsCounter(t *testing.T) {
	metrics, _ := newTestMetrics(t)

	metrics.IncrementRunningMonitors()
	metrics.IncrementRunningMonitors()
	metrics.DecrementRunningMonitors()

	if got := testutil.ToFloat64(metrics.MonitorsRunning); got != 1 {
		t.Fatalf("expected running monitors gauge to be 1, got %v", got)
	}
}

func TestRecordConfigReload(t *testing.T) {
	metrics, _ := newTestMetrics(t)

	metrics.RecordConfigReload()

	if got := testutil.ToFloat64(metrics.ConfigReloads); got != 1 {
		t.Fatalf("expected config reload counter to be 1, got %v", got)
	}

	if got := testutil.ToFloat64(metrics.LastConfigReload); got <= 0 {
		t.Fatalf("expected last config reload timestamp to be set, got %v", got)
	}
}
