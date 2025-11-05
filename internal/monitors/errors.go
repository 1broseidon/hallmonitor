package monitors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common monitor failures
var (
	// ErrMonitorDisabled indicates a monitor is disabled
	ErrMonitorDisabled = errors.New("monitor is disabled")

	// ErrMonitorNotFound indicates a monitor was not found
	ErrMonitorNotFound = errors.New("monitor not found")

	// ErrInvalidConfig indicates invalid monitor configuration
	ErrInvalidConfig = errors.New("invalid monitor configuration")

	// ErrTimeout indicates a monitor check timed out
	ErrTimeout = errors.New("monitor check timed out")

	// ErrConnectionFailed indicates a connection failure
	ErrConnectionFailed = errors.New("connection failed")

	// ErrDNSResolutionFailed indicates DNS resolution failure
	ErrDNSResolutionFailed = errors.New("DNS resolution failed")

	// ErrUnexpectedResponse indicates an unexpected response
	ErrUnexpectedResponse = errors.New("unexpected response")
)

// MonitorError represents a structured monitor error with context
type MonitorError struct {
	Monitor   string // Monitor name
	Type      string // Monitor type
	Operation string // Operation that failed
	Err       error  // Underlying error
}

// Error implements the error interface
func (e *MonitorError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("monitor %s (%s) failed during %s: %v", e.Monitor, e.Type, e.Operation, e.Err)
	}
	return fmt.Sprintf("monitor %s (%s) failed during %s", e.Monitor, e.Type, e.Operation)
}

// Unwrap implements error unwrapping
func (e *MonitorError) Unwrap() error {
	return e.Err
}

// NewMonitorError creates a new MonitorError
func NewMonitorError(monitor, monitorType, operation string, err error) *MonitorError {
	return &MonitorError{
		Monitor:   monitor,
		Type:      monitorType,
		Operation: operation,
		Err:       err,
	}
}

// TimeoutError represents a timeout error with context
type TimeoutError struct {
	Monitor string
	Type    string
	Timeout string
}

// Error implements the error interface
func (e *TimeoutError) Error() string {
	return fmt.Sprintf("monitor %s (%s) timed out after %s", e.Monitor, e.Type, e.Timeout)
}

// Is allows error comparison
func (e *TimeoutError) Is(target error) bool {
	return target == ErrTimeout
}

// ConnectionError represents a connection error with details
type ConnectionError struct {
	Monitor string
	Type    string
	Target  string
	Err     error
}

// Error implements the error interface
func (e *ConnectionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("monitor %s (%s) failed to connect to %s: %v", e.Monitor, e.Type, e.Target, e.Err)
	}
	return fmt.Sprintf("monitor %s (%s) failed to connect to %s", e.Monitor, e.Type, e.Target)
}

// Unwrap implements error unwrapping
func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// Is allows error comparison
func (e *ConnectionError) Is(target error) bool {
	return target == ErrConnectionFailed
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Monitor string
	Field   string
	Reason  string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for monitor %s: %s - %s", e.Monitor, e.Field, e.Reason)
}

// Is allows error comparison
func (e *ValidationError) Is(target error) bool {
	return target == ErrInvalidConfig
}
