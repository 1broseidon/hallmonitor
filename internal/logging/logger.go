// Package logging provides structured logging using zerolog with configurable
// levels and output formats including JSON and console modes.
package logging

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger wraps zerolog with additional context for Hall Monitor
type Logger struct {
	logger zerolog.Logger
}

// LogEvent represents a monitoring event type
type LogEvent string

const (
	EventCheckStarted   LogEvent = "check_started"
	EventCheckCompleted LogEvent = "check_completed"
	EventCheckFailed    LogEvent = "check_failed"
	EventConfigReload   LogEvent = "config_reload"
	EventServerStart    LogEvent = "server_start"
	EventServerStop     LogEvent = "server_stop"
	EventAlertFired     LogEvent = "alert_fired"
	EventAlertResolved  LogEvent = "alert_resolved"
)

// LogComponent represents a component of the application
type LogComponent string

const (
	ComponentMonitor   LogComponent = "monitor"
	ComponentScheduler LogComponent = "scheduler"
	ComponentAPI       LogComponent = "api"
	ComponentConfig    LogComponent = "config"
	ComponentMetrics   LogComponent = "metrics"
	ComponentAlert     LogComponent = "alert"
)

// Config represents logging configuration
type Config struct {
	Level  string            `yaml:"level"`
	Format string            `yaml:"format"` // json or text
	Output string            `yaml:"output"` // stdout, stderr, or file path
	Fields map[string]string `yaml:"fields"` // Additional fields for all logs
}

// InitLogger initializes the global logger
func InitLogger(config Config) (*Logger, error) {
	// Set log level
	level, err := zerolog.ParseLevel(config.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Configure output
	var output io.Writer = os.Stdout
	switch strings.ToLower(config.Output) {
	case "stderr":
		output = os.Stderr
	case "stdout", "":
		output = os.Stdout
	default:
		// Assume it's a file path
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		output = file
	}

	// Configure format
	var logger zerolog.Logger
	switch strings.ToLower(config.Format) {
	case "text", "console":
		logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
		})
	case "json", "":
		logger = zerolog.New(output)
	default:
		logger = zerolog.New(output)
	}

	// Add timestamp and additional fields
	logger = logger.With().
		Timestamp().
		Str("service", "hallmonitor").
		Logger()

	// Add configured fields
	for key, value := range config.Fields {
		logger = logger.With().Str(key, value).Logger()
	}

	// Set as global logger
	log.Logger = logger

	return &Logger{logger: logger}, nil
}

// GetGlobalLogger returns a logger instance with global context
func GetGlobalLogger() *Logger {
	return &Logger{logger: log.Logger}
}

// WithComponent adds component context to the logger
func (l *Logger) WithComponent(component LogComponent) *Logger {
	return &Logger{
		logger: l.logger.With().Str("component", string(component)).Logger(),
	}
}

// WithMonitor adds monitor context to the logger
func (l *Logger) WithMonitor(monitor, monitorType, group string) *Logger {
	return &Logger{
		logger: l.logger.With().
			Str("monitor", monitor).
			Str("type", monitorType).
			Str("group", group).
			Logger(),
	}
}

// WithEvent adds event context to the logger
func (l *Logger) WithEvent(event LogEvent) *Logger {
	return &Logger{
		logger: l.logger.With().Str("event", string(event)).Logger(),
	}
}

// WithError adds error context to the logger
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		logger: l.logger.With().AnErr("error", err).Logger(),
	}
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	event := l.logger.With()
	for key, value := range fields {
		switch v := value.(type) {
		case string:
			event = event.Str(key, v)
		case int:
			event = event.Int(key, v)
		case int64:
			event = event.Int64(key, v)
		case float64:
			event = event.Float64(key, v)
		case bool:
			event = event.Bool(key, v)
		case time.Duration:
			event = event.Dur(key, v)
		case time.Time:
			event = event.Time(key, v)
		default:
			event = event.Interface(key, v)
		}
	}
	return &Logger{logger: event.Logger()}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.logger.Error().Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string) {
	l.logger.Fatal().Msg(msg)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logger.Debug().Msgf(format, args...)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.Info().Msgf(format, args...)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logger.Warn().Msgf(format, args...)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.Error().Msgf(format, args...)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatal().Msgf(format, args...)
}

// MonitorCheck logs a monitor check event with structured data
func (l *Logger) MonitorCheck(monitor, monitorType, group, status string, duration time.Duration, err error) {
	event := l.logger.Info().
		Str("event", string(EventCheckCompleted)).
		Str("monitor", monitor).
		Str("type", monitorType).
		Str("group", group).
		Str("status", status).
		Dur("duration_ms", duration)

	if err != nil {
		event = event.AnErr("error", err).Str("level", "error")
		event.Msg("Monitor check failed")
	} else {
		event.Msg("Monitor check completed")
	}
}

// ConfigEvent logs configuration-related events
func (l *Logger) ConfigEvent(event LogEvent, msg string, fields map[string]interface{}) {
	logEvent := l.logger.Info().
		Str("event", string(event)).
		Str("component", string(ComponentConfig))

	for key, value := range fields {
		switch v := value.(type) {
		case string:
			logEvent = logEvent.Str(key, v)
		case int:
			logEvent = logEvent.Int(key, v)
		case bool:
			logEvent = logEvent.Bool(key, v)
		default:
			logEvent = logEvent.Interface(key, v)
		}
	}

	logEvent.Msg(msg)
}

// AlertEvent logs alerting events
func (l *Logger) AlertEvent(event LogEvent, monitor, rule string, labels map[string]string) {
	logEvent := l.logger.Warn().
		Str("event", string(event)).
		Str("component", string(ComponentAlert)).
		Str("monitor", monitor).
		Str("rule", rule)

	for key, value := range labels {
		logEvent = logEvent.Str("label_"+key, value)
	}

	if event == EventAlertFired {
		logEvent.Msg("Alert fired")
	} else {
		logEvent.Msg("Alert resolved")
	}
}
