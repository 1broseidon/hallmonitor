package monitors

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/1broseidon/hallmonitor/pkg/models"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		expectHost string
		expectPort int
		expectErr  bool
	}{
		{
			name:       "valid ipv4 with port",
			target:     "192.168.1.1:22",
			expectHost: "192.168.1.1",
			expectPort: 22,
			expectErr:  false,
		},
		{
			name:       "valid hostname with port",
			target:     "example.com:443",
			expectHost: "example.com",
			expectPort: 443,
			expectErr:  false,
		},
		{
			name:       "valid ipv6 with port",
			target:     "[::1]:8080",
			expectHost: "::1",
			expectPort: 8080,
			expectErr:  false,
		},
		{
			name:       "valid ipv6 full address",
			target:     "[2001:db8::1]:443",
			expectHost: "2001:db8::1",
			expectPort: 443,
			expectErr:  false,
		},
		{
			name:      "missing port",
			target:    "example.com",
			expectErr: true,
		},
		{
			name:      "invalid port non-numeric",
			target:    "example.com:abc",
			expectErr: true,
		},
		{
			name:      "port too low",
			target:    "example.com:0",
			expectErr: true,
		},
		{
			name:      "port too high",
			target:    "example.com:65536",
			expectErr: true,
		},
		{
			name:      "empty host",
			target:    ":8080",
			expectErr: true,
		},
		{
			name:      "ipv6 missing closing bracket",
			target:    "[::1:8080",
			expectErr: true,
		},
		{
			name:      "ipv6 missing port separator",
			target:    "[::1]8080",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := parseTarget(tt.target)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if host != tt.expectHost {
				t.Fatalf("expected host %s, got %s", tt.expectHost, host)
			}

			if port != tt.expectPort {
				t.Fatalf("expected port %d, got %d", tt.expectPort, port)
			}
		})
	}
}

func TestTCPMonitorValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *models.Monitor
		expectErr bool
	}{
		{
			name: "valid target",
			config: &models.Monitor{
				Type:   models.MonitorTypeTCP,
				Name:   "test",
				Target: "localhost:22",
			},
			expectErr: false,
		},
		{
			name: "missing target",
			config: &models.Monitor{
				Type: models.MonitorTypeTCP,
				Name: "test",
			},
			expectErr: true,
		},
		{
			name: "invalid target format",
			config: &models.Monitor{
				Type:   models.MonitorTypeTCP,
				Name:   "test",
				Target: "invalid",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, err := NewTCPMonitor(tt.config, "test-group", nil, nil)

			// NewTCPMonitor itself might fail for invalid targets
			if tt.expectErr && err != nil {
				return
			}

			if err != nil {
				t.Fatalf("NewTCPMonitor failed: %v", err)
			}

			err = monitor.Validate()
			if tt.expectErr && err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("expected no validation error, got: %v", err)
			}
		})
	}
}

func TestTCPMonitorCheckSuccess(t *testing.T) {
	// Start a temporary TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	// Get the dynamically assigned port
	addr := listener.Addr().(*net.TCPAddr)

	config := &models.Monitor{
		Type:    models.MonitorTypeTCP,
		Name:    "test-monitor",
		Target:  addr.String(),
		Timeout:  models.Duration(5 * time.Second),
	}

	monitor, err := NewTCPMonitor(config, "test-group", nil, nil)
	if err != nil {
		t.Fatalf("NewTCPMonitor failed: %v", err)
	}

	ctx := context.Background()
	result, err := monitor.Check(ctx)

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if result.Status != models.StatusUp {
		t.Fatalf("expected status up, got %s (error: %s)", result.Status, result.Error)
	}

	if result.TCPResult == nil {
		t.Fatalf("expected TCPResult to be populated")
	}

	if !result.TCPResult.Connected {
		t.Fatalf("expected connected to be true")
	}

	if result.TCPResult.Port != addr.Port {
		t.Fatalf("expected port %d, got %d", addr.Port, result.TCPResult.Port)
	}

	if result.Duration == 0 {
		t.Fatalf("expected non-zero duration")
	}
}

func TestTCPMonitorCheckFailure(t *testing.T) {
	// Use a port that's very unlikely to be in use
	config := &models.Monitor{
		Type:    models.MonitorTypeTCP,
		Name:    "test-monitor",
		Target:  "127.0.0.1:54321",
		Timeout:  models.Duration(100 * time.Millisecond),
	}

	monitor, err := NewTCPMonitor(config, "test-group", nil, nil)
	if err != nil {
		t.Fatalf("NewTCPMonitor failed: %v", err)
	}

	ctx := context.Background()
	result, err := monitor.Check(ctx)

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if result.Status != models.StatusDown {
		t.Fatalf("expected status down, got %s", result.Status)
	}

	if result.TCPResult == nil {
		t.Fatalf("expected TCPResult to be populated")
	}

	if result.TCPResult.Connected {
		t.Fatalf("expected connected to be false")
	}
}

func TestTCPMonitorCheckTimeout(t *testing.T) {
	// Use a non-routable IP to force timeout (RFC 5737 TEST-NET-1)
	config := &models.Monitor{
		Type:    models.MonitorTypeTCP,
		Name:    "test-monitor",
		Target:  "192.0.2.1:80",
		Timeout:  models.Duration(100 * time.Millisecond),
	}

	monitor, err := NewTCPMonitor(config, "test-group", nil, nil)
	if err != nil {
		t.Fatalf("NewTCPMonitor failed: %v", err)
	}

	ctx := context.Background()
	start := time.Now()
	result, err := monitor.Check(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if result.Status != models.StatusDown {
		t.Fatalf("expected status down for timeout, got %s", result.Status)
	}

	// Verify timeout occurred (within reasonable margin)
	if elapsed > 500*time.Millisecond {
		t.Fatalf("timeout took too long: %s (expected ~100ms)", elapsed)
	}
}
