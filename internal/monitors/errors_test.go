package monitors

import (
	"errors"
	"testing"
)

func TestTimeoutErrorIs(t *testing.T) {
	err := &TimeoutError{Monitor: "router", Type: "ping", Timeout: "5s"}
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected TimeoutError to compare equal to ErrTimeout")
	}

	if got := err.Error(); got == "" {
		t.Fatalf("expected TimeoutError message to be non-empty")
	}
}

func TestConnectionErrorIs(t *testing.T) {
	err := &ConnectionError{Monitor: "router", Type: "ping", Target: "192.168.1.1"}
	if !errors.Is(err, ErrConnectionFailed) {
		t.Fatalf("expected ConnectionError to compare equal to ErrConnectionFailed")
	}

	wrapped := &ConnectionError{Monitor: "router", Type: "ping", Target: "192.168.1.1", Err: ErrConnectionFailed}
	if !errors.Is(wrapped, ErrConnectionFailed) {
		t.Fatalf("expected wrapped ConnectionError to unwrap to ErrConnectionFailed")
	}

	if got := wrapped.Error(); got == "" {
		t.Fatalf("expected ConnectionError message to be non-empty")
	}
}

func TestValidationErrorIs(t *testing.T) {
	err := &ValidationError{Monitor: "router", Field: "target", Reason: "required"}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ValidationError to compare equal to ErrInvalidConfig")
	}

	if got := err.Error(); got == "" {
		t.Fatalf("expected ValidationError message to be non-empty")
	}
}

func TestMonitorErrorWraps(t *testing.T) {
	underlying := ErrTimeout
	err := NewMonitorError("router", "ping", "check", underlying)

	if err.Monitor != "router" || err.Type != "ping" || err.Operation != "check" {
		t.Fatalf("monitor error fields not initialized correctly: %+v", err)
	}

	if got := err.Error(); got == "" {
		t.Fatalf("expected MonitorError message to be non-empty")
	}

	if !errors.Is(err, underlying) {
		t.Fatalf("expected MonitorError to unwrap to underlying error")
	}
}
