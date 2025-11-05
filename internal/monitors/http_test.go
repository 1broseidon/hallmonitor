package monitors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/1broseidon/hallmonitor/pkg/models"
)

func TestHTTPMonitorValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *models.Monitor
		expectErr bool
	}{
		{
			name: "valid http url",
			config: &models.Monitor{
				Type: models.MonitorTypeHTTP,
				Name: "test",
				URL:  "http://example.com",
			},
			expectErr: false,
		},
		{
			name: "valid https url",
			config: &models.Monitor{
				Type: models.MonitorTypeHTTP,
				Name: "test",
				URL:  "https://example.com",
			},
			expectErr: false,
		},
		{
			name: "missing url",
			config: &models.Monitor{
				Type: models.MonitorTypeHTTP,
				Name: "test",
			},
			expectErr: true,
		},
		{
			name: "invalid url scheme",
			config: &models.Monitor{
				Type: models.MonitorTypeHTTP,
				Name: "test",
				URL:  "ftp://example.com",
			},
			expectErr: true,
		},
		{
			name: "invalid status code too low",
			config: &models.Monitor{
				Type:           models.MonitorTypeHTTP,
				Name:           "test",
				URL:            "http://example.com",
				ExpectedStatus: 50,
			},
			expectErr: true,
		},
		{
			name: "invalid status code too high",
			config: &models.Monitor{
				Type:           models.MonitorTypeHTTP,
				Name:           "test",
				URL:            "http://example.com",
				ExpectedStatus: 600,
			},
			expectErr: true,
		},
		{
			name: "valid expected status",
			config: &models.Monitor{
				Type:           models.MonitorTypeHTTP,
				Name:           "test",
				URL:            "http://example.com",
				ExpectedStatus: 404,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, err := NewHTTPMonitor(tt.config, "test-group", nil, nil)
			if err != nil {
				t.Fatalf("NewHTTPMonitor failed: %v", err)
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

func TestHTTPMonitorCheckSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "HallMonitor/1.0" {
			t.Errorf("expected User-Agent header to be set")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	config := &models.Monitor{
		Type:    models.MonitorTypeHTTP,
		Name:    "test-monitor",
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}

	monitor, err := NewHTTPMonitor(config, "test-group", nil, nil)
	if err != nil {
		t.Fatalf("NewHTTPMonitor failed: %v", err)
	}

	ctx := context.Background()
	result, err := monitor.Check(ctx)

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if result.Status != models.StatusUp {
		t.Fatalf("expected status up, got %s", result.Status)
	}

	if result.HTTPResult == nil {
		t.Fatalf("expected HTTPResult to be populated")
	}

	if result.HTTPResult.StatusCode != 200 {
		t.Fatalf("expected status code 200, got %d", result.HTTPResult.StatusCode)
	}

	if result.Duration == 0 {
		t.Fatalf("expected non-zero duration")
	}
}

func TestHTTPMonitorCheckUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := &models.Monitor{
		Type:           models.MonitorTypeHTTP,
		Name:           "test-monitor",
		URL:            server.URL,
		ExpectedStatus: 200,
		Timeout:        5 * time.Second,
	}

	monitor, err := NewHTTPMonitor(config, "test-group", nil, nil)
	if err != nil {
		t.Fatalf("NewHTTPMonitor failed: %v", err)
	}

	ctx := context.Background()
	result, err := monitor.Check(ctx)

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if result.Status != models.StatusDown {
		t.Fatalf("expected status down for unexpected status code, got %s", result.Status)
	}

	if result.Error == "" {
		t.Fatalf("expected error message for unexpected status code")
	}

	if result.HTTPResult == nil || result.HTTPResult.StatusCode != 404 {
		t.Fatalf("expected HTTPResult with status code 404")
	}
}

func TestHTTPMonitorCheckTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Monitor{
		Type:    models.MonitorTypeHTTP,
		Name:    "test-monitor",
		URL:     server.URL,
		Timeout: 50 * time.Millisecond,
	}

	monitor, err := NewHTTPMonitor(config, "test-group", nil, nil)
	if err != nil {
		t.Fatalf("NewHTTPMonitor failed: %v", err)
	}

	ctx := context.Background()
	result, err := monitor.Check(ctx)

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if result.Status != models.StatusDown {
		t.Fatalf("expected status down for timeout, got %s", result.Status)
	}
}

func TestHTTPMonitorCheckCustomHeaders(t *testing.T) {
	receivedHeaders := make(map[string]string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders["X-Custom-Header"] = r.Header.Get("X-Custom-Header")
		receivedHeaders["Authorization"] = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Monitor{
		Type:    models.MonitorTypeHTTP,
		Name:    "test-monitor",
		URL:     server.URL,
		Timeout: 5 * time.Second,
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
			"Authorization":   "Bearer token123",
		},
	}

	monitor, err := NewHTTPMonitor(config, "test-group", nil, nil)
	if err != nil {
		t.Fatalf("NewHTTPMonitor failed: %v", err)
	}

	ctx := context.Background()
	_, err = monitor.Check(ctx)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if receivedHeaders["X-Custom-Header"] != "test-value" {
		t.Fatalf("expected custom header to be sent, got %s", receivedHeaders["X-Custom-Header"])
	}

	if receivedHeaders["Authorization"] != "Bearer token123" {
		t.Fatalf("expected authorization header to be sent, got %s", receivedHeaders["Authorization"])
	}
}
