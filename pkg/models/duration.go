package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration is a custom type that wraps time.Duration to handle JSON unmarshaling
// from duration strings like "30s", "1m", "2h", etc.
type Duration time.Duration

// UnmarshalJSON implements json.Unmarshaler for Duration
func (d *Duration) UnmarshalJSON(b []byte) error {
	// Try to unmarshal as a number first (nanoseconds)
	var n int64
	if err := json.Unmarshal(b, &n); err == nil {
		*d = Duration(n)
		return nil
	}

	// Try to unmarshal as a string (e.g., "30s", "1m")
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("duration must be a number or string: %w", err)
	}

	// Parse the duration string
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration string %q: %w", s, err)
	}

	*d = Duration(dur)
	return nil
}

// MarshalJSON implements json.Marshaler for Duration
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalYAML implements yaml.Unmarshaler for Duration
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try to unmarshal as a number first
	var n int64
	if err := unmarshal(&n); err == nil {
		*d = Duration(n)
		return nil
	}

	// Try to unmarshal as a string
	var s string
	if err := unmarshal(&s); err != nil {
		return fmt.Errorf("duration must be a number or string: %w", err)
	}

	// Parse the duration string
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration string %q: %w", s, err)
	}

	*d = Duration(dur)
	return nil
}

// MarshalYAML implements yaml.Marshaler for Duration
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// String returns the string representation of the duration
func (d Duration) String() string {
	return time.Duration(d).String()
}

// ToDuration converts Duration to time.Duration
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}

// Nanoseconds returns the duration as an int64 nanosecond count
func (d Duration) Nanoseconds() int64 {
	return int64(d)
}
