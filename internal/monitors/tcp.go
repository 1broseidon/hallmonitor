package monitors

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/internal/metrics"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// TCPMonitor implements TCP port monitoring
type TCPMonitor struct {
	*BaseMonitor
	host string
	port int
}

// NewTCPMonitor creates a new TCP monitor
func NewTCPMonitor(config *models.Monitor, group string, logger *logging.Logger, metrics *metrics.Metrics) (*TCPMonitor, error) {
	// Parse target into host and port
	host, port, err := parseTarget(config.Target)
	if err != nil {
		return nil, fmt.Errorf("invalid target format: %w", err)
	}

	return &TCPMonitor{
		BaseMonitor: NewBaseMonitor(config, group, logger, metrics),
		host:        host,
		port:        port,
	}, nil
}

// Check performs the TCP check
func (t *TCPMonitor) Check(ctx context.Context) (*models.MonitorResult, error) {
	startTime := time.Now()

	// Set timeout from context or config
	timeout := t.Config.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// Create a dialer with timeout
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	// Attempt to connect
	address := net.JoinHostPort(t.host, strconv.Itoa(t.port))
	conn, err := dialer.DialContext(ctx, "tcp", address)
	duration := time.Since(startTime)

	// Create TCP result data
	tcpResult := &models.TCPResult{
		Port:         t.port,
		Connected:    err == nil,
		ResponseTime: duration,
	}

	var status models.MonitorStatus
	if err != nil {
		status = models.StatusDown
	} else {
		status = models.StatusUp
		// Close the connection immediately after successful connect
		if conn != nil {
			conn.Close()
		}
	}

	// Create monitor result
	result := t.CreateResult(status, duration, err)
	result.TCPResult = tcpResult

	// Record TCP-specific metrics
	if t.Metrics != nil {
		t.Metrics.RecordTCPCheck(
			t.Config.Name,
			t.Group,
			t.port,
			duration,
		)
	}

	t.RecordMetrics(result)
	t.LogResult(result)

	return result, nil
}

// Validate validates the TCP monitor configuration
func (t *TCPMonitor) Validate() error {
	if t.Config.Target == "" {
		return fmt.Errorf("TCP monitor requires target")
	}

	// Validate target format
	_, _, err := parseTarget(t.Config.Target)
	if err != nil {
		return fmt.Errorf("invalid target format: %w", err)
	}

	return nil
}

// parseTarget parses a target string into host and port
// Supports formats: "host:port", "ip:port", "[ipv6]:port"
func parseTarget(target string) (string, int, error) {
	// Handle IPv6 addresses in brackets
	if strings.HasPrefix(target, "[") {
		// IPv6 format: [::1]:8080
		closeBracket := strings.Index(target, "]")
		if closeBracket == -1 {
			return "", 0, fmt.Errorf("invalid IPv6 format, missing closing bracket")
		}

		host := target[1:closeBracket]
		portStr := target[closeBracket+1:]

		if !strings.HasPrefix(portStr, ":") {
			return "", 0, fmt.Errorf("invalid IPv6 format, missing port separator")
		}

		port, err := strconv.Atoi(portStr[1:])
		if err != nil {
			return "", 0, fmt.Errorf("invalid port number: %w", err)
		}

		return host, port, nil
	}

	// Regular format: host:port or ip:port
	parts := strings.Split(target, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("target must be in format 'host:port' or '[ipv6]:port'")
	}

	host := parts[0]
	if host == "" {
		return "", 0, fmt.Errorf("host cannot be empty")
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid port number: %w", err)
	}

	if port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}

	return host, port, nil
}
