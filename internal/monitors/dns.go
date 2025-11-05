package monitors

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/internal/metrics"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// DNSMonitor implements DNS query monitoring
type DNSMonitor struct {
	*BaseMonitor
	resolver  *net.Resolver
	server    string
	port      string
	queryType string
}

// NewDNSMonitor creates a new DNS monitor
func NewDNSMonitor(config *models.Monitor, group string, logger *logging.Logger, metrics *metrics.Metrics) (*DNSMonitor, error) {
	// Parse DNS server and port
	server, port, err := parseDNSTarget(config.Target)
	if err != nil {
		return nil, fmt.Errorf("invalid DNS target: %w", err)
	}

	// Set default query type
	queryType := config.QueryType
	if queryType == "" {
		queryType = "A"
	}

	// Validate query type
	if !isValidQueryType(queryType) {
		return nil, fmt.Errorf("unsupported DNS query type: %s", queryType)
	}

	// Create custom resolver with timeout
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: timeout,
			}
			return d.DialContext(ctx, network, net.JoinHostPort(server, port))
		},
	}

	return &DNSMonitor{
		BaseMonitor: NewBaseMonitor(config, group, logger, metrics),
		resolver:    resolver,
		server:      server,
		port:        port,
		queryType:   queryType,
	}, nil
}

// Check performs the DNS check
func (d *DNSMonitor) Check(ctx context.Context) (*models.MonitorResult, error) {
	startTime := time.Now()

	var answers []string
	var err error
	var rcode int = 0 // NOERROR

	// Perform DNS query based on type
	switch strings.ToUpper(d.queryType) {
	case "A":
		ips, lookupErr := d.resolver.LookupIPAddr(ctx, d.Config.Query)
		if lookupErr != nil {
			err = lookupErr
			rcode = extractRCodeFromError(lookupErr)
		} else {
			for _, ip := range ips {
				answers = append(answers, ip.IP.String())
			}
		}

	case "AAAA":
		ips, lookupErr := d.resolver.LookupIPAddr(ctx, d.Config.Query)
		if lookupErr != nil {
			err = lookupErr
			rcode = extractRCodeFromError(lookupErr)
		} else {
			for _, ip := range ips {
				if ip.IP.To4() == nil { // IPv6 address
					answers = append(answers, ip.IP.String())
				}
			}
		}

	case "CNAME":
		cname, lookupErr := d.resolver.LookupCNAME(ctx, d.Config.Query)
		if lookupErr != nil {
			err = lookupErr
			rcode = extractRCodeFromError(lookupErr)
		} else {
			answers = append(answers, cname)
		}

	case "MX":
		mxRecords, lookupErr := d.resolver.LookupMX(ctx, d.Config.Query)
		if lookupErr != nil {
			err = lookupErr
			rcode = extractRCodeFromError(lookupErr)
		} else {
			for _, mx := range mxRecords {
				answers = append(answers, fmt.Sprintf("%d %s", mx.Pref, mx.Host))
			}
		}

	case "TXT":
		txtRecords, lookupErr := d.resolver.LookupTXT(ctx, d.Config.Query)
		if lookupErr != nil {
			err = lookupErr
			rcode = extractRCodeFromError(lookupErr)
		} else {
			answers = txtRecords
		}

	case "NS":
		nsRecords, lookupErr := d.resolver.LookupNS(ctx, d.Config.Query)
		if lookupErr != nil {
			err = lookupErr
			rcode = extractRCodeFromError(lookupErr)
		} else {
			for _, ns := range nsRecords {
				answers = append(answers, ns.Host)
			}
		}

	default:
		err = fmt.Errorf("unsupported query type: %s", d.queryType)
	}

	duration := time.Since(startTime)

	// Create DNS result data
	dnsResult := &models.DNSResult{
		QueryType:    d.queryType,
		ResponseCode: rcode,
		ResponseTime: duration,
		Answers:      answers,
		ResponseSize: len(answers),
	}

	// Determine status
	var status models.MonitorStatus
	if err != nil {
		status = models.StatusDown
	} else if len(answers) == 0 {
		status = models.StatusDown
		err = fmt.Errorf("no DNS records found")
	} else {
		status = models.StatusUp
		// Check expected response if configured
		if d.Config.ExpectedResponse != "" {
			found := false
			for _, answer := range answers {
				if answer == d.Config.ExpectedResponse {
					found = true
					break
				}
			}
			if !found {
				status = models.StatusDown
				err = fmt.Errorf("expected response '%s' not found in answers: %v",
					d.Config.ExpectedResponse, answers)
			}
		}
	}

	// Create monitor result
	result := d.CreateResult(status, duration, err)
	result.DNSResult = dnsResult

	// Record DNS-specific metrics
	if d.Metrics != nil {
		d.Metrics.RecordDNSCheck(
			d.Config.Name,
			d.Group,
			d.queryType,
			d.server,
			rcode,
			duration,
		)
	}

	d.RecordMetrics(result)
	d.LogResult(result)

	return result, nil
}

// Validate validates the DNS monitor configuration
func (d *DNSMonitor) Validate() error {
	if d.Config.Target == "" {
		return fmt.Errorf("DNS monitor requires target")
	}

	if d.Config.Query == "" {
		return fmt.Errorf("DNS monitor requires query")
	}

	// Validate DNS server target
	_, _, err := parseDNSTarget(d.Config.Target)
	if err != nil {
		return fmt.Errorf("invalid DNS target: %w", err)
	}

	// Validate query type if specified
	if d.Config.QueryType != "" && !isValidQueryType(d.Config.QueryType) {
		return fmt.Errorf("unsupported DNS query type: %s", d.Config.QueryType)
	}

	return nil
}

// parseDNSTarget parses DNS server target into host and port
func parseDNSTarget(target string) (string, string, error) {
	// Default port for DNS
	defaultPort := "53"

	if strings.Contains(target, ":") {
		host, port, err := net.SplitHostPort(target)
		if err != nil {
			return "", "", err
		}

		// Validate port
		if portNum, err := strconv.Atoi(port); err != nil || portNum < 1 || portNum > 65535 {
			return "", "", fmt.Errorf("invalid port: %s", port)
		}

		return host, port, nil
	}

	// No port specified, use default
	return target, defaultPort, nil
}

// isValidQueryType checks if the DNS query type is supported
func isValidQueryType(queryType string) bool {
	validTypes := map[string]bool{
		"A":     true,
		"AAAA":  true,
		"CNAME": true,
		"MX":    true,
		"TXT":   true,
		"NS":    true,
	}
	return validTypes[strings.ToUpper(queryType)]
}

// extractRCodeFromError attempts to extract DNS response code from error
func extractRCodeFromError(err error) int {
	if err == nil {
		return 0 // NOERROR
	}

	// Try to get DNS-specific error
	if dnsErr, ok := err.(*net.DNSError); ok {
		switch {
		case dnsErr.IsNotFound:
			return 3 // NXDOMAIN
		case dnsErr.IsTimeout:
			return 2 // SERVFAIL (approximation for timeout)
		case dnsErr.IsTemporary:
			return 2 // SERVFAIL (temporary failure)
		}
	}

	// Check for context errors
	if err == context.DeadlineExceeded {
		return 2 // SERVFAIL (timeout)
	}
	if err == context.Canceled {
		return 2 // SERVFAIL (canceled)
	}

	// Default to SERVFAIL for unknown errors
	return 2
}
