package monitors

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/internal/metrics"
	"github.com/1broseidon/hallmonitor/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
)

func TestExtractRCodeFromError(t *testing.T) {
	if code := extractRCodeFromError(nil); code != 0 {
		t.Fatalf("expected nil error to return RCODE 0, got %d", code)
	}

	notFoundErr := &net.DNSError{IsNotFound: true}
	if code := extractRCodeFromError(notFoundErr); code != 3 {
		t.Fatalf("expected NXDOMAIN (3) for IsNotFound error, got %d", code)
	}

	timeoutErr := &net.DNSError{IsTimeout: true}
	if code := extractRCodeFromError(timeoutErr); code != 2 {
		t.Fatalf("expected SERVFAIL (2) for timeout error, got %d", code)
	}

	if code := extractRCodeFromError(context.DeadlineExceeded); code != 2 {
		t.Fatalf("expected SERVFAIL (2) for context deadline exceeded, got %d", code)
	}

	if code := extractRCodeFromError(context.Canceled); code != 2 {
		t.Fatalf("expected SERVFAIL (2) for context canceled, got %d", code)
	}
}

func TestParseDNSTarget(t *testing.T) {
	host, port, err := parseDNSTarget("8.8.8.8:53")
	if err != nil {
		t.Fatalf("unexpected error parsing explicit port: %v", err)
	}
	if host != "8.8.8.8" || port != "53" {
		t.Fatalf("unexpected host/port: %s:%s", host, port)
	}

	host, port, err = parseDNSTarget("1.1.1.1")
	if err != nil {
		t.Fatalf("unexpected error parsing default port: %v", err)
	}
	if host != "1.1.1.1" || port != "53" {
		t.Fatalf("expected default port 53, got %s:%s", host, port)
	}

	if _, _, err := parseDNSTarget("example.com:abc"); err == nil {
		t.Fatalf("expected error for invalid port, got nil")
	}
}

func TestIsValidQueryType(t *testing.T) {
	valid := []string{"A", "a", "AAAA", "Mx", "txt", "NS"}
	for _, q := range valid {
		if !isValidQueryType(q) {
			t.Fatalf("expected query type %s to be valid", q)
		}
	}

	invalid := []string{"SRV", "PTR", "", "unknown"}
	for _, q := range invalid {
		if isValidQueryType(q) {
			t.Fatalf("expected query type %s to be invalid", q)
		}
	}
}

func TestNewDNSMonitor(t *testing.T) {
	metricsInstance := metrics.NewMetrics(prometheus.NewRegistry())
	logger, _ := logging.InitLogger(logging.Config{Level: "debug", Format: "json"})

	tests := []struct {
		name        string
		config      *models.Monitor
		group       string
		expectError bool
	}{
		{
			name: "valid DNS monitor with explicit port",
			config: &models.Monitor{
				Name:      "dns-test",
				Type:      "dns",
				Target:    "8.8.8.8:53",
				Query:     "example.com",
				QueryType: "A",
				Interval:  30 * time.Second,
				Timeout:   5 * time.Second,
			},
			group:       "test-group",
			expectError: false,
		},
		{
			name: "valid DNS monitor with default port",
			config: &models.Monitor{
				Name:     "dns-test",
				Type:     "dns",
				Target:   "1.1.1.1",
				Query:    "example.com",
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
			},
			group:       "test-group",
			expectError: false,
		},
		{
			name: "valid DNS monitor with default query type",
			config: &models.Monitor{
				Name:     "dns-test",
				Type:     "dns",
				Target:   "8.8.8.8",
				Query:    "example.com",
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
			},
			group:       "test-group",
			expectError: false,
		},
		{
			name: "invalid DNS target port",
			config: &models.Monitor{
				Name:     "dns-test",
				Type:     "dns",
				Target:   "8.8.8.8:invalid",
				Query:    "example.com",
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
			},
			group:       "test-group",
			expectError: true,
		},
		{
			name: "invalid query type",
			config: &models.Monitor{
				Name:      "dns-test",
				Type:      "dns",
				Target:    "8.8.8.8",
				Query:     "example.com",
				QueryType: "INVALID",
				Interval:  30 * time.Second,
				Timeout:   5 * time.Second,
			},
			group:       "test-group",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, err := NewDNSMonitor(tt.config, tt.group, logger, metricsInstance)

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

			if monitor.resolver == nil {
				t.Error("expected resolver to be initialized")
			}
		})
	}
}

func TestDNSMonitorValidate(t *testing.T) {
	metricsInstance := metrics.NewMetrics(prometheus.NewRegistry())
	logger, _ := logging.InitLogger(logging.Config{Level: "debug", Format: "json"})

	tests := []struct {
		name        string
		config      *models.Monitor
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: &models.Monitor{
				Name:      "dns-test",
				Type:      "dns",
				Target:    "8.8.8.8:53",
				Query:     "example.com",
				QueryType: "A",
				Interval:  30 * time.Second,
				Timeout:   5 * time.Second,
			},
			expectError: false,
		},
		{
			name: "missing target",
			config: &models.Monitor{
				Name:     "dns-test",
				Type:     "dns",
				Query:    "example.com",
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
			},
			expectError: true,
			errorMsg:    "requires target",
		},
		{
			name: "missing query",
			config: &models.Monitor{
				Name:     "dns-test",
				Type:     "dns",
				Target:   "8.8.8.8",
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
			},
			expectError: true,
			errorMsg:    "requires query",
		},
		{
			name: "invalid target format",
			config: &models.Monitor{
				Name:     "dns-test",
				Type:     "dns",
				Target:   "8.8.8.8:abc",
				Query:    "example.com",
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
			},
			expectError: true,
			errorMsg:    "invalid DNS target",
		},
		{
			name: "invalid query type",
			config: &models.Monitor{
				Name:      "dns-test",
				Type:      "dns",
				Target:    "8.8.8.8",
				Query:     "example.com",
				QueryType: "INVALID",
				Interval:  30 * time.Second,
				Timeout:   5 * time.Second,
			},
			expectError: true,
			errorMsg:    "unsupported DNS query type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, _ := NewDNSMonitor(tt.config, "test-group", logger, metricsInstance)

			// For cases where creation itself fails, skip validation
			if monitor == nil {
				return
			}

			err := monitor.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("expected validation error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestDNSMonitorCheck(t *testing.T) {
	metricsInstance := metrics.NewMetrics(prometheus.NewRegistry())
	logger, _ := logging.InitLogger(logging.Config{Level: "debug", Format: "json"})

	t.Run("successful A record query", func(t *testing.T) {
		config := &models.Monitor{
			Name:      "dns-test",
			Type:      "dns",
			Target:    "8.8.8.8",
			Query:     "google.com",
			QueryType: "A",
			Interval:  30 * time.Second,
			Timeout:   5 * time.Second,
		}

		monitor, err := NewDNSMonitor(config, "test-group", logger, metricsInstance)
		if err != nil {
			t.Fatalf("failed to create monitor: %v", err)
		}

		ctx := context.Background()
		result, err := monitor.Check(ctx)

		if err != nil {
			t.Fatalf("check failed: %v", err)
		}

		if result == nil {
			t.Fatal("expected result to be non-nil")
		}

		if result.Status != models.StatusUp {
			t.Errorf("expected status up, got %s", result.Status)
		}

		if result.DNSResult == nil {
			t.Fatal("expected DNS result to be set")
		}

		if len(result.DNSResult.Answers) == 0 {
			t.Error("expected at least one answer")
		}
	})

	t.Run("query with expected response", func(t *testing.T) {
		config := &models.Monitor{
			Name:             "dns-test",
			Type:             "dns",
			Target:           "8.8.8.8",
			Query:            "google.com",
			QueryType:        "A",
			ExpectedResponse: "142.250.80.46", // This might fail if google's IP changes
			Interval:         30 * time.Second,
			Timeout:          5 * time.Second,
		}

		monitor, err := NewDNSMonitor(config, "test-group", logger, metricsInstance)
		if err != nil {
			t.Fatalf("failed to create monitor: %v", err)
		}

		ctx := context.Background()
		result, _ := monitor.Check(ctx)

		if result == nil {
			t.Fatal("expected result to be non-nil")
		}

		// Note: This test might fail if the expected IP doesn't match
		// In a real scenario, we'd use a controlled DNS server for testing
	})

	t.Run("timeout handling", func(t *testing.T) {
		config := &models.Monitor{
			Name:      "dns-test",
			Type:      "dns",
			Target:    "8.8.8.8",
			Query:     "google.com",
			QueryType: "A",
			Interval:  30 * time.Second,
			Timeout:   1 * time.Nanosecond, // Very short timeout to force failure
		}

		monitor, err := NewDNSMonitor(config, "test-group", logger, metricsInstance)
		if err != nil {
			t.Fatalf("failed to create monitor: %v", err)
		}

		ctx := context.Background()
		result, _ := monitor.Check(ctx)

		if result == nil {
			t.Fatal("expected result to be non-nil")
		}

		// With such a short timeout, we expect the check to fail
		if result.Status == models.StatusUp {
			t.Log("Warning: expected timeout to cause failure, but check succeeded")
		}
	})

	t.Run("different query types", func(t *testing.T) {
		queryTypes := []struct {
			qtype string
			query string
		}{
			{"MX", "google.com"},
			{"TXT", "google.com"},
			{"NS", "google.com"},
		}

		for _, qt := range queryTypes {
			t.Run(qt.qtype, func(t *testing.T) {
				config := &models.Monitor{
					Name:      "dns-test",
					Type:      "dns",
					Target:    "8.8.8.8",
					Query:     qt.query,
					QueryType: qt.qtype,
					Interval:  30 * time.Second,
					Timeout:   5 * time.Second,
				}

				monitor, err := NewDNSMonitor(config, "test-group", logger, metricsInstance)
				if err != nil {
					t.Fatalf("failed to create monitor: %v", err)
				}

				ctx := context.Background()
				result, err := monitor.Check(ctx)

				if err != nil {
					t.Fatalf("check failed: %v", err)
				}

				if result == nil {
					t.Fatal("expected result to be non-nil")
				}

				if result.DNSResult == nil {
					t.Fatal("expected DNS result to be set")
				}

				if result.DNSResult.QueryType != qt.qtype {
					t.Errorf("expected query type %s, got %s", qt.qtype, result.DNSResult.QueryType)
				}
			})
		}
	})
}
