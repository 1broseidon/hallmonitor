package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMonitorResultJSONMarshalling(t *testing.T) {
	expiry := time.Unix(1_700_000_000, 0)
	result := MonitorResult{
		Monitor:   "homepage",
		Type:      MonitorTypeHTTP,
		Group:     "default",
		Status:    StatusUp,
		Duration:  1500 * time.Millisecond,
		Timestamp: time.Unix(1700000000, 0),
		HTTPResult: &HTTPResult{
			StatusCode:    200,
			ResponseTime:  120 * time.Millisecond,
			ResponseSize:  512,
			Headers:       map[string]string{"Content-Type": "text/html"},
			SSLCertExpiry: &expiry,
		},
	}

	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal monitor result: %v", err)
	}

	jsonStr := string(payload)
	expectedSnippets := []string{
		`"monitor":"homepage"`,
		`"status":"up"`,
		`"http_result"`,
		`"ssl_cert_expiry"`,
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(jsonStr, snippet) {
			t.Fatalf("expected JSON payload to contain %s, got %s", snippet, jsonStr)
		}
	}
}

func TestMonitorEnabledPointer(t *testing.T) {
	enabled := true
	monitor := Monitor{
		Type:    MonitorTypeTCP,
		Name:    "redis",
		Target:  "localhost",
		Port:    6379,
		Enabled: &enabled,
	}

	if monitor.Enabled == nil || !*monitor.Enabled {
		t.Fatalf("expected monitor to be enabled by pointer")
	}

	disabled := false
	monitor.Enabled = &disabled
	if monitor.Enabled == nil || *monitor.Enabled {
		t.Fatalf("expected monitor to be disabled when pointer set to false")
	}
}
