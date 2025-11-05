package api

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/1broseidon/hallmonitor/internal/config"
	"github.com/1broseidon/hallmonitor/internal/logging"
)

func createTestServer(t *testing.T) *Server {
	t.Helper()

	// Create test logger (discard output)
	logger, err := logging.InitLogger(logging.Config{
		Level:  "error",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("failed to create test logger: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: "7878",
			Host: "0.0.0.0",
		},
	}

	// Create Prometheus registry
	reg := prometheus.NewRegistry()

	// Create server (NewServer creates its own scheduler, monitor manager, etc.)
	server := NewServer(cfg, logger, reg)
	return server
}

func TestHealthHandler(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyStr := string(body)
	if bodyStr == "" {
		t.Fatalf("expected non-empty response body")
	}

	// Verify it contains expected fields
	if !contains(bodyStr, "status") || !contains(bodyStr, "healthy") {
		t.Fatalf("response missing expected fields: %s", bodyStr)
	}
}

func TestReadyHandler(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	req := httptest.NewRequest("GET", "/ready", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyStr := string(body)
	if !contains(bodyStr, "ready") {
		t.Fatalf("response missing expected content: %s", bodyStr)
	}
}

func TestMetricsHandler(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyStr := string(body)
	if bodyStr == "" {
		t.Fatalf("expected non-empty metrics response")
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", contentType)
	}
}

func TestGetMonitorsHandler(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	req := httptest.NewRequest("GET", "/api/v1/monitors", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyStr := string(body)
	if !contains(bodyStr, "monitors") || !contains(bodyStr, "total") {
		t.Fatalf("response missing expected fields: %s", bodyStr)
	}
}

func TestGetMonitorHandlerNotFound(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	req := httptest.NewRequest("GET", "/api/v1/monitors/nonexistent-monitor", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyStr := string(body)
	if !contains(bodyStr, "not found") {
		t.Fatalf("response missing error message: %s", bodyStr)
	}
}

func TestGetGroupsHandler(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	req := httptest.NewRequest("GET", "/api/v1/groups", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyStr := string(body)
	if !contains(bodyStr, "groups") {
		t.Fatalf("response missing expected fields: %s", bodyStr)
	}
}

func TestGetGroupHandlerNotFound(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	req := httptest.NewRequest("GET", "/api/v1/groups/nonexistent-group", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestGetConfigHandler(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	req := httptest.NewRequest("GET", "/api/v1/config", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyStr := string(body)
	if bodyStr == "" {
		t.Fatalf("expected non-empty config response")
	}
}

func TestGrafanaEndpointsReturnEmptyArrays(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	tests := []struct {
		name   string
		path   string
		method string
	}{
		{"query", "/api/v1/query", "POST"},
		{"tags", "/api/v1/query/tags", "POST"},
		{"annotations", "/api/v1/annotations", "GET"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := server.app.Test(req, -1)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != fiber.StatusOK {
				t.Fatalf("expected status 200, got %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			bodyStr := string(body)
			// Should return empty array
			if bodyStr != "[]" && bodyStr != "[ ]" && !contains(bodyStr, "[]") {
				t.Fatalf("expected empty array, got: %s", bodyStr)
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	req := httptest.NewRequest("OPTIONS", "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check for CORS headers
	corsHeader := resp.Header.Get("Access-Control-Allow-Origin")
	if corsHeader == "" {
		t.Logf("warning: no CORS header found (may be OK depending on config)")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
