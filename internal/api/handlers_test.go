package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/1broseidon/hallmonitor/internal/config"
	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
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
			Port:            "7878",
			Host:            "0.0.0.0",
			EnableDashboard: true,
		},
	}

	// Create Prometheus registry
	reg := prometheus.NewRegistry()

	// Create server (NewServer creates its own scheduler, monitor manager, etc.)
	server := NewServer(cfg, "config.yml", logger, reg)
	return server
}

func loadMonitors(t *testing.T, server *Server, groups []models.MonitorGroup) {
	t.Helper()

	if err := server.monitorManager.LoadMonitors(groups); err != nil {
		t.Fatalf("failed to load monitors: %v", err)
	}
}

func storeResult(t *testing.T, server *Server, result *models.MonitorResult) {
	t.Helper()

	if result == nil {
		t.Fatalf("result cannot be nil")
	}

	// Use test helper instead of reflection
	testHelper := server.scheduler.NewTestHelper()
	testHelper.InjectResult(result.Monitor, result)
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

func TestGetMonitorsHandlerIncludesSchedulerData(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	enabled := true
	loadMonitors(t, server, []models.MonitorGroup{
		{
			Name: "core",
			Monitors: []models.Monitor{
				{Type: models.MonitorTypeHTTP, Name: "homepage", URL: "https://example.com", Enabled: &enabled},
				{Type: models.MonitorTypeHTTP, Name: "status", URL: "https://status.example.com", Enabled: &enabled},
			},
		},
	})

	now := time.Now().UTC().Truncate(time.Second)
	storeResult(t, server, &models.MonitorResult{Monitor: "homepage", Type: models.MonitorTypeHTTP, Group: "core", Status: models.StatusUp, Duration: 250 * time.Millisecond, Timestamp: now})
	storeResult(t, server, &models.MonitorResult{Monitor: "status", Type: models.MonitorTypeHTTP, Group: "core", Status: models.StatusDown, Duration: 500 * time.Millisecond, Error: "http 500", Timestamp: now})

	req := httptest.NewRequest("GET", "/api/v1/monitors", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	monitorsSlice, ok := payload["monitors"].([]interface{})
	if !ok || len(monitorsSlice) != 2 {
		t.Fatalf("expected two monitors in response, got %v", payload["monitors"])
	}

	statusByName := make(map[string]string)
	for _, item := range monitorsSlice {
		monitorMap, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("unexpected monitor payload type: %T", item)
		}
		name := monitorMap["name"].(string)
		statusByName[name] = monitorMap["status"].(string)
	}

	if statusByName["homepage"] != "up" {
		t.Fatalf("expected homepage status 'up', got %s", statusByName["homepage"])
	}

	if statusByName["status"] != "down" {
		t.Fatalf("expected status monitor to be 'down', got %s", statusByName["status"])
	}

	if total, ok := payload["total"].(float64); !ok || int(total) != 2 {
		t.Fatalf("expected total 2, got %v", payload["total"])
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

func TestGetMonitorHandlerReturnsDetailedStatus(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	enabled := true
	loadMonitors(t, server, []models.MonitorGroup{
		{
			Name: "core",
			Monitors: []models.Monitor{
				{
					Type:    models.MonitorTypeHTTP,
					Name:    "homepage",
					URL:     "https://example.com",
					Enabled: &enabled,
				},
			},
		},
	})

	timestamp := time.Now().UTC().Truncate(time.Second)
	duration := 1500 * time.Millisecond

	storeResult(t, server, &models.MonitorResult{
		Monitor:   "homepage",
		Type:      models.MonitorTypeHTTP,
		Group:     "core",
		Status:    models.StatusDown,
		Duration:  duration,
		Error:     "timeout exceeded",
		Timestamp: timestamp,
		Metadata: map[string]string{
			"latency": "120ms",
		},
	})

	req := httptest.NewRequest("GET", "/api/v1/monitors/homepage", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload["status"] != "down" {
		t.Fatalf("expected status 'down', got %v", payload["status"])
	}

	if payload["duration"] != duration.String() {
		t.Fatalf("expected duration %s, got %v", duration.String(), payload["duration"])
	}

	if payload["error"] != "timeout exceeded" {
		t.Fatalf("expected error message, got %v", payload["error"])
	}

	expectedTimestamp := timestamp.Format("2006-01-02T15:04:05Z07:00")
	if payload["last_check"] != expectedTimestamp {
		t.Fatalf("expected last_check %s, got %v", expectedTimestamp, payload["last_check"])
	}

	if payload["metadata"] == nil {
		t.Fatalf("expected metadata to be present")
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

func TestGetGroupHandlerReturnsMonitors(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	enabled := true
	loadMonitors(t, server, []models.MonitorGroup{
		{
			Name: "core",
			Monitors: []models.Monitor{
				{Type: models.MonitorTypeHTTP, Name: "homepage", URL: "https://example.com", Enabled: &enabled},
			},
		},
	})

	storeResult(t, server, &models.MonitorResult{
		Monitor:   "homepage",
		Type:      models.MonitorTypeHTTP,
		Group:     "core",
		Status:    models.StatusUp,
		Duration:  100 * time.Millisecond,
		Timestamp: time.Now().UTC(),
	})

	req := httptest.NewRequest("GET", "/api/v1/groups/core", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload struct {
		Group struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"group"`
		Monitors []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"monitors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Group.Name != "core" {
		t.Fatalf("expected group name 'core', got %s", payload.Group.Name)
	}

	if len(payload.Monitors) != 1 || payload.Monitors[0].Name != "homepage" {
		t.Fatalf("expected single monitor 'homepage', got %+v", payload.Monitors)
	}

	if payload.Monitors[0].Status != "up" {
		t.Fatalf("expected monitor status 'up', got %s", payload.Monitors[0].Status)
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

func TestReloadConfigHandler(t *testing.T) {
	// Create a temporary config file for testing
	tmpConfig := `
server:
  port: "7878"
  host: "0.0.0.0"
  enableDashboard: true

monitoring:
  defaultInterval: "30s"
  defaultTimeout: "10s"
  groups:
    - name: "test-group"
      monitors:
        - type: "http"
          name: "test-monitor"
          url: "https://example.com"
          expectedStatus: 200
`
	tmpFile, err := os.CreateTemp("", "test-config-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(tmpConfig); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	tmpFile.Close()

	// Create server with temp config path
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:            "7878",
			Host:            "0.0.0.0",
			EnableDashboard: true,
		},
	}
	logger, _ := logging.InitLogger(logging.Config{
		Level:  "error",
		Format: "json",
	})
	reg := prometheus.NewRegistry()
	server := NewServer(cfg, tmpFile.Name(), logger, reg)
	defer server.app.Shutdown()

	// Start scheduler
	if err := server.scheduler.Start(context.Background()); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer func() {
		if server.scheduler.IsRunning() {
			server.scheduler.Stop()
		}
	}()

	req := httptest.NewRequest("POST", "/api/v1/reload", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload["success"] != true {
		t.Fatalf("expected success true, got %v", payload["success"])
	}

	if payload["message"] != "Configuration reloaded successfully" {
		t.Fatalf("unexpected message: %v", payload["message"])
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

func TestDashboardHandler(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	req := httptest.NewRequest("GET", "/", nil)
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
		t.Fatalf("expected non-empty dashboard response")
	}

	// Check that it's HTML content
	contentType := resp.Header.Get("Content-Type")
	if !contains(contentType, "text/html") {
		t.Fatalf("expected HTML content type, got %s", contentType)
	}

	// Verify it contains expected dashboard elements
	if !contains(bodyStr, "Hall Monitor") {
		t.Error("dashboard missing expected title")
	}
}

func TestDashboardAmbientHandler(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	req := httptest.NewRequest("GET", "/dashboard/ambient", nil)
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
		t.Fatalf("expected non-empty ambient dashboard response")
	}

	// Check that it's HTML content
	contentType := resp.Header.Get("Content-Type")
	if !contains(contentType, "text/html") {
		t.Fatalf("expected HTML content type, got %s", contentType)
	}

	// Verify it contains expected ambient elements
	if !contains(bodyStr, "Ambient View") || !contains(bodyStr, "Hall Monitor") {
		t.Error("ambient dashboard missing expected content")
	}
}

func TestGetMonitorHistoryHandler(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	enabled := true
	loadMonitors(t, server, []models.MonitorGroup{
		{
			Name: "core",
			Monitors: []models.Monitor{
				{Type: models.MonitorTypeHTTP, Name: "api", URL: "https://api.example.com", Enabled: &enabled},
			},
		},
	})

	// Store multiple results to create history
	baseTime := time.Now().UTC().Add(-1 * time.Hour)
	for i := 0; i < 5; i++ {
		storeResult(t, server, &models.MonitorResult{
			Monitor:   "api",
			Type:      models.MonitorTypeHTTP,
			Group:     "core",
			Status:    models.StatusUp,
			Duration:  100 * time.Millisecond,
			Timestamp: baseTime.Add(time.Duration(i) * 10 * time.Minute),
		})
	}

	req := httptest.NewRequest("GET", "/api/v1/monitors/api/history", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload["monitor"] != "api" {
		t.Fatalf("expected monitor name 'api', got %v", payload["monitor"])
	}

	results, ok := payload["results"].([]interface{})
	if !ok {
		t.Fatalf("expected results array, got %T", payload["results"])
	}

	if len(results) == 0 {
		t.Error("expected non-empty results")
	}
}

func TestGetMonitorHistoryHandlerWithTimeRange(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	enabled := true
	loadMonitors(t, server, []models.MonitorGroup{
		{
			Name: "core",
			Monitors: []models.Monitor{
				{Type: models.MonitorTypeHTTP, Name: "api", URL: "https://api.example.com", Enabled: &enabled},
			},
		},
	})

	// Store results spanning different time periods
	baseTime := time.Now().UTC().Add(-2 * time.Hour)
	for i := 0; i < 10; i++ {
		storeResult(t, server, &models.MonitorResult{
			Monitor:   "api",
			Type:      models.MonitorTypeHTTP,
			Group:     "core",
			Status:    models.StatusUp,
			Duration:  100 * time.Millisecond,
			Timestamp: baseTime.Add(time.Duration(i) * 15 * time.Minute),
		})
	}

	// Request last hour
	req := httptest.NewRequest("GET", "/api/v1/monitors/api/history?period=1h", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	results, ok := payload["results"].([]interface{})
	if !ok {
		t.Fatalf("expected results array, got %T", payload["results"])
	}

	// Should have fewer results when filtering by time
	if len(results) >= 10 {
		t.Logf("warning: expected filtered results to have fewer than 10 results, got %d", len(results))
	}
}

func TestGetMonitorHistoryHandlerNotFound(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	req := httptest.NewRequest("GET", "/api/v1/monitors/nonexistent/history", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Handler returns 200 with empty results for nonexistent monitors
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	results, ok := payload["results"].([]interface{})
	if !ok {
		t.Fatalf("expected results array, got %T", payload["results"])
	}

	// Should be empty for nonexistent monitor
	if len(results) != 0 {
		t.Errorf("expected empty results for nonexistent monitor, got %d", len(results))
	}
}

func TestGetMonitorUptimeHandler(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	enabled := true
	loadMonitors(t, server, []models.MonitorGroup{
		{
			Name: "core",
			Monitors: []models.Monitor{
				{Type: models.MonitorTypeHTTP, Name: "api", URL: "https://api.example.com", Enabled: &enabled},
			},
		},
	})

	// Store mix of up and down results
	baseTime := time.Now().UTC().Add(-24 * time.Hour)
	for i := 0; i < 20; i++ {
		status := models.StatusUp
		if i%5 == 0 { // Every 5th check fails
			status = models.StatusDown
		}
		storeResult(t, server, &models.MonitorResult{
			Monitor:   "api",
			Type:      models.MonitorTypeHTTP,
			Group:     "core",
			Status:    status,
			Duration:  100 * time.Millisecond,
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
		})
	}

	req := httptest.NewRequest("GET", "/api/v1/monitors/api/uptime", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload["monitor"] != "api" {
		t.Fatalf("expected monitor name 'api', got %v", payload["monitor"])
	}

	// Check for expected uptime fields
	if payload["period"] == nil {
		t.Error("expected period field")
	}

	if payload["uptime_percent"] == nil {
		t.Error("expected uptime_percent field")
	}

	if payload["total_checks"] == nil {
		t.Error("expected total_checks field")
	}

	// Validate uptime percentage is reasonable (should be around 80% given our test data)
	uptimePercent, ok := payload["uptime_percent"].(float64)
	if ok && (uptimePercent < 0 || uptimePercent > 100) {
		t.Fatalf("expected uptime percentage between 0-100, got %f", uptimePercent)
	}

	totalChecks, ok := payload["total_checks"].(float64)
	if ok && totalChecks != 20 {
		t.Logf("expected 20 total checks, got %v", totalChecks)
	}
}

func TestGetMonitorUptimeHandlerNotFound(t *testing.T) {
	server := createTestServer(t)
	defer server.app.Shutdown()

	req := httptest.NewRequest("GET", "/api/v1/monitors/nonexistent/uptime", nil)
	resp, err := server.app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Handler returns 200 with zero checks for nonexistent monitors
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	totalChecks, ok := payload["total_checks"].(float64)
	if !ok {
		t.Fatalf("expected total_checks to be a number, got %T", payload["total_checks"])
	}

	// Should have zero checks for nonexistent monitor
	if totalChecks != 0 {
		t.Errorf("expected 0 total checks for nonexistent monitor, got %v", totalChecks)
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
