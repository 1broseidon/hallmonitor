package monitors

import (
	"context"
	"net"
	"testing"
)

func TestExtractRCodeFromError(t *testing.T) {
	if code := extractRCodeFromError(nil); code != 0 {
		t.Fatalf("expected nil error to return RCODE 0, got %d", code)
	}

	notFoundErr := &net.DNSError{IsNotFound: true}
	if code := extractRCodeFromError(notFoundErr); code != 3 {
		t.Fatalf("expected NXDOMAIN (3) for IsNotFound error, got %d", code)
	}

	timeoutErr := &net.DNSError{IsTimeout: true}
	if code := extractRCodeFromError(timeoutErr); code != 2 {
		t.Fatalf("expected SERVFAIL (2) for timeout error, got %d", code)
	}

	if code := extractRCodeFromError(context.DeadlineExceeded); code != 2 {
		t.Fatalf("expected SERVFAIL (2) for context deadline exceeded, got %d", code)
	}

	if code := extractRCodeFromError(context.Canceled); code != 2 {
		t.Fatalf("expected SERVFAIL (2) for context canceled, got %d", code)
	}
}

func TestParseDNSTarget(t *testing.T) {
	host, port, err := parseDNSTarget("8.8.8.8:53")
	if err != nil {
		t.Fatalf("unexpected error parsing explicit port: %v", err)
	}
	if host != "8.8.8.8" || port != "53" {
		t.Fatalf("unexpected host/port: %s:%s", host, port)
	}

	host, port, err = parseDNSTarget("1.1.1.1")
	if err != nil {
		t.Fatalf("unexpected error parsing default port: %v", err)
	}
	if host != "1.1.1.1" || port != "53" {
		t.Fatalf("expected default port 53, got %s:%s", host, port)
	}

	if _, _, err := parseDNSTarget("example.com:abc"); err == nil {
		t.Fatalf("expected error for invalid port, got nil")
	}
}

func TestIsValidQueryType(t *testing.T) {
	valid := []string{"A", "a", "AAAA", "Mx", "txt", "NS"}
	for _, q := range valid {
		if !isValidQueryType(q) {
			t.Fatalf("expected query type %s to be valid", q)
		}
	}

	invalid := []string{"SRV", "PTR", "", "unknown"}
	for _, q := range invalid {
		if isValidQueryType(q) {
			t.Fatalf("expected query type %s to be invalid", q)
		}
	}
}
