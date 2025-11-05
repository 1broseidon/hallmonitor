package monitors

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/internal/metrics"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// HTTPMonitor implements HTTP/HTTPS monitoring
type HTTPMonitor struct {
	*BaseMonitor
	client *http.Client
}

// NewHTTPMonitor creates a new HTTP monitor
func NewHTTPMonitor(config *models.Monitor, group string, logger *logging.Logger, metrics *metrics.Metrics) (*HTTPMonitor, error) {
	// Create HTTP client with timeout
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	// Configure TLS for SSL certificate checking
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false, // Always verify certificates
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig:    tlsConfig,
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
			DisableKeepAlives:  false,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 5 redirects
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	return &HTTPMonitor{
		BaseMonitor: NewBaseMonitor(config, group, logger, metrics),
		client:      client,
	}, nil
}

// Check performs the HTTP check
func (h *HTTPMonitor) Check(ctx context.Context) (*models.MonitorResult, error) {
	startTime := time.Now()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", h.Config.URL, nil)
	if err != nil {
		duration := time.Since(startTime)
		result := h.CreateResult(models.StatusDown, duration, err)
		h.RecordMetrics(result)
		h.LogResult(result)
		return result, nil
	}

	// Add custom headers
	if h.Config.Headers != nil {
		for key, value := range h.Config.Headers {
			req.Header.Set(key, value)
		}
	}

	// Set User-Agent
	req.Header.Set("User-Agent", "HallMonitor/1.0")

	// Perform the request
	resp, err := h.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		result := h.CreateResult(models.StatusDown, duration, err)
		h.RecordMetrics(result)
		h.LogResult(result)
		return result, nil
	}
	defer resp.Body.Close()

	// Create HTTP result data
	httpResult := &models.HTTPResult{
		StatusCode:   resp.StatusCode,
		ResponseTime: duration,
		ResponseSize: resp.ContentLength,
		Headers:      make(map[string]string),
	}

	// Capture important response headers
	for key, values := range resp.Header {
		if len(values) > 0 {
			switch strings.ToLower(key) {
			case "content-type", "server", "cache-control":
				httpResult.Headers[key] = values[0]
			}
		}
	}

	// Check SSL certificate if HTTPS
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		httpResult.SSLCertExpiry = &cert.NotAfter

		// Record SSL certificate expiry in metrics
		if h.Metrics != nil {
			h.Metrics.RecordSSLCertExpiry(
				h.Config.Name,
				h.Group,
				cert.Subject.CommonName,
				cert.NotAfter,
			)
		}

		// Check if certificate expires soon (configurable threshold)
		warningDays := h.Config.SSLCertExpiryWarningDays
		if warningDays == 0 {
			warningDays = 30 // Fallback default
		}
		warningThreshold := time.Now().Add(time.Duration(warningDays) * 24 * time.Hour)
		if cert.NotAfter.Before(warningThreshold) {
			h.Logger.WithComponent(logging.ComponentMonitor).
				WithFields(map[string]interface{}{
					"monitor":                h.Config.Name,
					"expires_at":             cert.NotAfter,
					"days_left":              int(time.Until(cert.NotAfter).Hours() / 24),
					"warning_threshold_days": warningDays,
				}).
				Warn("SSL certificate expires soon")
		}
	}

	// Determine status based on response
	var status models.MonitorStatus
	var checkError error

	expectedStatus := h.Config.ExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = 200 // Default expected status
	}

	if resp.StatusCode == expectedStatus {
		status = models.StatusUp
	} else {
		status = models.StatusDown
		checkError = fmt.Errorf("unexpected status code: %d (expected %d)", resp.StatusCode, expectedStatus)
	}

	// Create monitor result
	result := h.CreateResult(status, duration, checkError)
	result.HTTPResult = httpResult

	// Record metrics
	if h.Metrics != nil {
		// Record HTTP-specific metrics
		method := "GET"
		h.Metrics.RecordHTTPCheck(
			h.Config.Name,
			h.Group,
			method,
			resp.StatusCode,
			duration,
		)
	}

	h.RecordMetrics(result)
	h.LogResult(result)

	return result, nil
}

// Validate validates the HTTP monitor configuration
func (h *HTTPMonitor) Validate() error {
	if h.Config.URL == "" {
		return fmt.Errorf("HTTP monitor requires url")
	}

	// Validate URL format
	parsedURL, err := url.Parse(h.Config.URL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Ensure scheme is HTTP or HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}

	// Validate expected status code if provided
	if h.Config.ExpectedStatus != 0 {
		if h.Config.ExpectedStatus < 100 || h.Config.ExpectedStatus > 599 {
			return fmt.Errorf("invalid expected status code: %d", h.Config.ExpectedStatus)
		}
	}

	return nil
}
