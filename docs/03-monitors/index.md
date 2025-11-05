# Monitor Types

Hall Monitor supports four monitor types for comprehensive infrastructure monitoring.

## Overview

| Type | Protocol | Use Cases | Status |
|------|----------|-----------|--------|
| [HTTP](#http-monitors) | HTTP/HTTPS | Web apps, APIs, SSL certs | Production Ready |
| [TCP](#tcp-monitors) | TCP | Port connectivity, services | Production Ready |
| [DNS](#dns-monitors) | DNS (UDP/TCP) | DNS servers, records | Production Ready |
| [Ping](#ping-monitors) | ICMP/UDP | Host reachability, latency | Production Ready |

## HTTP Monitors

Monitor HTTP and HTTPS endpoints.

### Features
- HTTP/HTTPS support
- Custom headers (Authorization, etc.)
- Expected status code validation
- SSL certificate expiry tracking
- Response time measurement

### Basic Configuration

```yaml
- type: "http"
  name: "api-server"
  url: "https://api.example.com/health"
  expectedStatus: 200
  timeout: "5s"
```

### With Authentication

```yaml
- type: "http"
  name: "authenticated-api"
  url: "https://api.example.com/secure"
  expectedStatus: 200
  headers:
    Authorization: "Bearer ${API_TOKEN}"
```

See [HTTP Monitors](./http.md) for detailed documentation.

## TCP Monitors

Test TCP port connectivity.

### Features
- Port reachability testing
- Connection time measurement
- IPv4 and IPv6 support

### Basic Configuration

```yaml
- type: "tcp"
  name: "database"
  target: "db.example.com:5432"
  timeout: "3s"
```

### Common Use Cases

- Database connectivity (PostgreSQL, MySQL, MongoDB)
- SSH access monitoring
- SMTP/IMAP servers
- Custom application ports

See [TCP Monitors](./tcp.md) for detailed documentation.

## DNS Monitors

Query DNS records and validate responses.

### Features
- Multiple query types (A, AAAA, CNAME, MX, TXT, NS)
- Custom DNS server
- Expected response validation
- Query time measurement

### Basic Configuration

```yaml
- type: "dns"
  name: "dns-server"
  target: "8.8.8.8:53"
  query: "example.com"
  queryType: "A"
  timeout: "3s"
```

### Supported Query Types

- **A**: IPv4 address records
- **AAAA**: IPv6 address records
- **CNAME**: Canonical name records
- **MX**: Mail exchange records
- **TXT**: Text records
- **NS**: Name server records

See [DNS Monitors](./dns.md) for detailed documentation.

## Ping Monitors

Test host reachability using ICMP ping.

### Features
- ICMP ping with automatic fallback to unprivileged mode
- Packet loss tracking
- Round-trip time statistics (min/max/avg)
- Configurable packet count

### Basic Configuration

```yaml
- type: "ping"
  name: "gateway"
  target: "192.168.1.1"
  count: 3
  timeout: "3s"
```

### Privilege Requirements

ICMP ping requires elevated privileges. Hall Monitor automatically falls back to unprivileged mode (UDP) if ICMP is not available.

To enable ICMP:
```bash
# Grant capability to binary
sudo setcap cap_net_raw+ep /usr/local/bin/hallmonitor

# Or run Docker with privileges
docker run --cap-add NET_RAW --cap-add NET_ADMIN --network host \
  -v $(pwd)/config.yml:/etc/hallmonitor/config.yml:ro \
  ghcr.io/1broseidon/hallmonitor:latest
```

See [Ping Monitors](./ping.md) for detailed documentation.

## Comparison

| Feature | HTTP | TCP | DNS | Ping |
|---------|------|-----|-----|------|
| Application Layer | Yes | No | Yes | No |
| Custom Headers | Yes | No | No | No |
| SSL Tracking | Yes | No | No | No |
| Port Check | N/A | Yes | Yes | No |
| Latency | Yes | Yes | Yes | Yes |
| Packet Loss | No | No | No | Yes |
| Privileges Required | No | No | No | Optional |

## Common Configuration Patterns

### Service Health Checks

```yaml
monitoring:
  groups:
    - name: "services"
      monitors:
        # Web application
        - type: "http"
          name: "web-app"
          url: "https://app.example.com/health"
          expectedStatus: 200

        # Database
        - type: "tcp"
          name: "database"
          target: "db.example.com:5432"

        # Cache
        - type: "tcp"
          name: "redis"
          target: "redis.example.com:6379"
```

### Network Infrastructure

```yaml
monitoring:
  groups:
    - name: "infrastructure"
      monitors:
        # Gateway
        - type: "ping"
          name: "gateway"
          target: "192.168.1.1"

        # DNS server
        - type: "dns"
          name: "dns"
          target: "192.168.1.1:53"
          query: "example.local"
          queryType: "A"

        # Web proxy
        - type: "tcp"
          name: "proxy"
          target: "proxy.example.com:3128"
```

### SSL Certificate Monitoring

```yaml
monitoring:
  defaultSSLCertExpiryWarningDays: 30

  groups:
    - name: "ssl-monitoring"
      interval: "3600s"  # Check hourly
      monitors:
        - type: "http"
          name: "main-site-ssl"
          url: "https://www.example.com"
          sslCertExpiryWarningDays: 30

        - type: "http"
          name: "api-ssl"
          url: "https://api.example.com"
          sslCertExpiryWarningDays: 14
```

## Monitor Labels

Add labels to monitors for organization and filtering:

```yaml
monitors:
  - type: "http"
    name: "payment-api"
    url: "https://api.example.com/payment"
    labels:
      environment: "production"
      team: "platform"
      criticality: "high"
      region: "us-east-1"
```

Labels are:
- Included in Prometheus metrics
- Available in API responses
- Useful for alert routing
- Helpful for dashboard filtering

## Best Practices

### Interval Selection

- **Critical services**: 10-15 seconds
- **Important services**: 30 seconds
- **Regular services**: 60 seconds
- **SSL certificates**: 3600 seconds (1 hour)

### Timeout Guidelines

- **Local services**: 2-3 seconds
- **Internal network**: 5 seconds
- **Internet services**: 10 seconds
- **Slow services**: 15-30 seconds

### Naming Conventions

Use descriptive, consistent names:

```yaml
# Good examples
name: "api-production-us-east"
name: "postgres-primary"
name: "router-datacenter-1"

# Avoid
name: "check1"
name: "test"
name: "monitor"
```

## Next Steps

- [HTTP Monitors](./http.md) - Detailed HTTP monitoring
- [TCP Monitors](./tcp.md) - TCP port monitoring
- [DNS Monitors](./dns.md) - DNS query monitoring
- [Ping Monitors](./ping.md) - ICMP ping monitoring
- [Examples](./examples.md) - Real-world configuration examples
