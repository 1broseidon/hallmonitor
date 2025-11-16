package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
)

func TestInitLoggerSetsDefaultsAndWritesJSON(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	prevLevel := zerolog.GlobalLevel()
	prevLogger := zerologlog.Logger
	t.Cleanup(func() {
		zerolog.SetGlobalLevel(prevLevel)
		zerologlog.Logger = prevLogger
	})

	logger, err := InitLogger(Config{
		Level:  "invalid-level",
		Format: "json",
		Output: logPath,
		Fields: map[string]string{
			"environment": "test",
		},
	})
	if err != nil {
		t.Fatalf("InitLogger returned error: %v", err)
	}

	logger.Info("structured message")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		t.Fatalf("expected log output to be written")
	}

	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry); err != nil {
		t.Fatalf("failed to decode log entry: %v", err)
	}

	if got := entry["service"]; got != "hallmonitor" {
		t.Fatalf("expected service field 'hallmonitor', got %v", got)
	}

	if got := entry["environment"]; got != "test" {
		t.Fatalf("expected environment field 'test', got %v", got)
	}

	if got := entry["message"]; got != "structured message" {
		t.Fatalf("expected message 'structured message', got %v", got)
	}

	if got := entry["level"]; got != "info" {
		t.Fatalf("expected level 'info', got %v", got)
	}

	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Fatalf("expected invalid level to fall back to info, got %s", zerolog.GlobalLevel())
	}
}

func TestInitLoggerFileOutputError(t *testing.T) {
	prevLevel := zerolog.GlobalLevel()
	prevLogger := zerologlog.Logger
	t.Cleanup(func() {
		zerolog.SetGlobalLevel(prevLevel)
		zerologlog.Logger = prevLogger
	})

	badPath := filepath.Join(t.TempDir(), "nested", "log.json")
	if _, err := InitLogger(Config{Output: badPath}); err == nil {
		t.Fatalf("expected error when log file path directory does not exist")
	}
}

func TestLoggerContextPropagation(t *testing.T) {
	var buf bytes.Buffer
	base := &Logger{logger: zerolog.New(&buf).With().Timestamp().Logger()}

	ctx := base.
		WithComponent(ComponentScheduler).
		WithMonitor("dns-check", "ping", "network").
		WithEvent(EventCheckFailed)

	ctx = ctx.WithFields(map[string]interface{}{
		"retries": 3,
		"timeout": 250 * time.Millisecond,
		"active":  true,
	})

	ctx = ctx.WithError(errors.New("network timeout"))

	ctx.Error("check failed")

	output := strings.TrimSpace(buf.String())
	if output == "" {
		t.Fatalf("expected logger to emit output")
	}

	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to decode log entry: %v", err)
	}

	if got := entry["component"]; got != string(ComponentScheduler) {
		t.Fatalf("expected component %s, got %v", ComponentScheduler, got)
	}

	if got := entry["monitor"]; got != "dns-check" {
		t.Fatalf("expected monitor 'dns-check', got %v", got)
	}

	if got := entry["type"]; got != "ping" {
		t.Fatalf("expected type 'ping', got %v", got)
	}

	if got := entry["group"]; got != "network" {
		t.Fatalf("expected group 'network', got %v", got)
	}

	if got := entry["event"]; got != string(EventCheckFailed) {
		t.Fatalf("expected event %s, got %v", EventCheckFailed, got)
	}

	if got := entry["retries"]; got != float64(3) {
		t.Fatalf("expected retries 3, got %v", got)
	}

	if got := entry["active"]; got != true {
		t.Fatalf("expected active true, got %v", got)
	}

	if got := entry["timeout"]; got == nil {
		t.Fatalf("expected timeout field to be present")
	} else {
		if val, ok := got.(float64); !ok || val <= 0 {
			t.Fatalf("expected timeout to be positive float, got %v", got)
		}
	}

	if !strings.Contains(output, "network timeout") {
		t.Fatalf("expected error context to include error message, got %s", output)
	}

	if got := entry["message"]; got != "check failed" {
		t.Fatalf("expected message 'check failed', got %v", got)
	}
}

func TestLoggerMethodsDebugInfoWarn(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{logger: zerolog.New(&buf).Level(zerolog.DebugLevel)}

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")

	output := buf.String()

	if !strings.Contains(output, "debug message") {
		t.Errorf("expected debug message in output")
	}
	if !strings.Contains(output, "info message") {
		t.Errorf("expected info message in output")
	}
	if !strings.Contains(output, "warn message") {
		t.Errorf("expected warn message in output")
	}

	// Verify log levels
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 log lines, got %d", len(lines))
	}

	for i, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("failed to decode log entry %d: %v", i, err)
		}

		level, ok := entry["level"].(string)
		if !ok {
			t.Fatalf("expected level to be string, got %T", entry["level"])
		}

		switch i {
		case 0:
			if level != "debug" {
				t.Errorf("expected first log to be debug level, got %s", level)
			}
		case 1:
			if level != "info" {
				t.Errorf("expected second log to be info level, got %s", level)
			}
		case 2:
			if level != "warn" {
				t.Errorf("expected third log to be warn level, got %s", level)
			}
		}
	}
}

func TestLoggerFormattedMethods(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{logger: zerolog.New(&buf).Level(zerolog.DebugLevel)}

	logger.Debugf("debug %s", "formatted")
	logger.Infof("info %d", 123)
	logger.Warnf("warn %v", true)
	logger.Errorf("error %s %d", "test", 456)

	output := buf.String()

	if !strings.Contains(output, "debug formatted") {
		t.Errorf("expected 'debug formatted' in output")
	}
	if !strings.Contains(output, "info 123") {
		t.Errorf("expected 'info 123' in output")
	}
	if !strings.Contains(output, "warn true") {
		t.Errorf("expected 'warn true' in output")
	}
	if !strings.Contains(output, "error test 456") {
		t.Errorf("expected 'error test 456' in output")
	}
}

func TestGetGlobalLogger(t *testing.T) {
	// Initialize logger first
	prevLevel := zerolog.GlobalLevel()
	prevLogger := zerologlog.Logger
	t.Cleanup(func() {
		zerolog.SetGlobalLevel(prevLevel)
		zerologlog.Logger = prevLogger
	})

	_, err := InitLogger(Config{
		Level:  "info",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}

	logger := GetGlobalLogger()
	if logger == nil {
		t.Fatal("expected GetGlobalLogger to return non-nil logger")
	}

	// Verify it's functional
	var buf bytes.Buffer
	logger.logger = zerolog.New(&buf)
	logger.Info("test global logger")

	output := buf.String()
	if !strings.Contains(output, "test global logger") {
		t.Errorf("expected global logger to log messages")
	}
}

func TestLoggerWithFieldsVariants(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{logger: zerolog.New(&buf)}

	// Test WithFields with multiple types
	enriched := logger.WithFields(map[string]interface{}{
		"string_field": "value",
		"int_field":    42,
		"bool_field":   true,
		"float_field":  3.14,
	})

	enriched.Info("fields test")

	output := buf.String()
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Fatalf("failed to decode log entry: %v", err)
	}

	if entry["string_field"] != "value" {
		t.Errorf("expected string_field to be 'value', got %v", entry["string_field"])
	}
	if entry["int_field"] != float64(42) {
		t.Errorf("expected int_field to be 42, got %v", entry["int_field"])
	}
	if entry["bool_field"] != true {
		t.Errorf("expected bool_field to be true, got %v", entry["bool_field"])
	}
}
