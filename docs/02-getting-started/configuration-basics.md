# Configuration Basics

Learn the fundamentals of configuring Hall Monitor.

## Configuration File Structure

Hall Monitor uses YAML configuration files with the following top-level sections:

```yaml
server:           # Server settings (port, host, dashboard)
metrics:          # Prometheus metrics configuration
logging:          # Log level, format, output
monitoring:       # Monitor groups and checks
alerting:         # Alert rules (optional)
webhooks:         # Webhook notifications (optional)
```

## Minimal Configuration

The simplest working configuration:

```yaml
server:
  port: "7878"
  host: "0.0.0.0"
  enableDashboard: true

monitoring:
  defaultInterval: "30s"
  defaultTimeout: "10s"

  groups:
    - name: "basic-checks"
      monitors:
        - type: "http"
          name: "example"
          url: "https://example.com"
          expectedStatus: 200
```

## Server Configuration

Configure the HTTP server:

```yaml
server:
  port: "7878"                    # Port to listen on
  host: "0.0.0.0"                 # Interface to bind (0.0.0.0 = all)
  enableDashboard: true           # Enable web dashboard
  corsOrigins:                    # CORS allowed origins
    - "http://localhost:3000"
```

## Logging Configuration

Control log output:

```yaml
logging:
  level: "info"                   # debug, info, warn, error
  format: "json"                  # json or text
  output: "stdout"                # stdout, stderr, or file path
  fields:                         # Additional fields in logs
    app: "hallmonitor"
    environment: "production"
```

**Log Levels**:
- `debug`: Verbose output for troubleshooting
- `info`: Normal operational messages
- `warn`: Warning messages
- `error`: Error messages only

**Formats**:
- `json`: Structured JSON (recommended for production)
- `text`: Human-readable (good for development)

## Metrics Configuration

Configure Prometheus metrics:

```yaml
metrics:
  enabled: true                     # Enable metrics export
  path: "/metrics"                  # Metrics endpoint path
  includeProcessMetrics: true       # Include Go process metrics
  includeGoMetrics: true            # Include Go runtime metrics
```

## Monitoring Configuration

### Global Defaults

Set defaults for all monitors:

```yaml
monitoring:
  defaultInterval: "30s"                    # How often to check
  defaultTimeout: "10s"                     # Max time per check
  defaultSSLCertExpiryWarningDays: 30       # SSL cert warning threshold
```

### Monitor Groups

Organize monitors into logical groups:

```yaml
monitoring:
  groups:
    - name: "critical-services"
      interval: "10s"              # Group-level interval
      monitors:
        - type: "http"
          name: "api"
          url: "https://api.example.com"

    - name: "infrastructure"
      interval: "30s"
      monitors:
        - type: "ping"
          name: "router"
          target: "192.168.1.1"
```

## Monitor Configuration

### Common Fields

All monitors support these fields:

```yaml
- type: "http"                    # Monitor type (required)
  name: "my-monitor"              # Unique name (required)
  interval: "30s"                 # Check interval (optional)
  timeout: "10s"                  # Check timeout (optional)
  enabled: true                   # Enable/disable (default: true)
  labels:                         # Custom labels (optional)
    env: "production"
    team: "platform"
```

### HTTP Monitors

```yaml
- type: "http"
  name: "web-server"
  url: "https://example.com"              # Required
  expectedStatus: 200                      # Expected HTTP status
  headers:                                 # Custom headers (optional)
    Authorization: "Bearer token"
  sslCertExpiryWarningDays: 30            # SSL warning threshold
```

### TCP Monitors

```yaml
- type: "tcp"
  name: "database"
  target: "db.example.com:5432"           # host:port (required)
  timeout: "5s"
```

### DNS Monitors

```yaml
- type: "dns"
  name: "dns-check"
  target: "8.8.8.8:53"                    # DNS server (required)
  query: "example.com"                     # Query domain (required)
  queryType: "A"                           # A, AAAA, CNAME, MX, TXT, NS
  expectedResponse: "93.184.216.34"       # Expected answer (optional)
```

### Ping Monitors

```yaml
- type: "ping"
  name: "gateway"
  target: "192.168.1.1"                   # IP or hostname (required)
  count: 3                                 # Number of pings (default: 3)
  timeout: "3s"
```

## Configuration Inheritance

Configuration values are inherited in this order (highest priority first):

1. **Monitor-specific** configuration
2. **Group-specific** configuration
3. **Global defaults**
4. **Built-in defaults**

Example:

```yaml
monitoring:
  defaultInterval: "60s"        # Global default
  defaultTimeout: "10s"

  groups:
    - name: "fast-checks"
      interval: "10s"           # Group override
      monitors:
        - type: "http"
          name: "local-api"
          url: "http://localhost:8080"
          timeout: "2s"         # Monitor override
          # interval: inherited from group (10s)

        - type: "http"
          name: "external-api"
          url: "https://api.example.com"
          # interval: inherited from group (10s)
          # timeout: inherited from global (10s)
```

## Environment Variables

Use environment variables in your configuration:

```yaml
server:
  port: "${SERVER_PORT}"

monitoring:
  groups:
    - name: "external-services"
      monitors:
        - type: "http"
          name: "api"
          url: "${API_URL}"
          headers:
            Authorization: "Bearer ${API_TOKEN}"
```

Set variables before starting:

```bash
export SERVER_PORT=7878
export API_URL="https://api.example.com"
export API_TOKEN="secret-token"

hallmonitor --config config.yml
```

## Configuration Validation

Hall Monitor validates configuration on startup and checks:

- Required fields are present
- Monitor names are unique
- URLs and targets are valid
- Intervals and timeouts are reasonable
- Query types are supported

Invalid configuration prevents startup with a helpful error message:

```
Error: Invalid configuration: monitor 'api-server' in group 'web-services': url is required for HTTP monitors
```

## Multiple Configuration Files

You can use different configuration files for different environments:

```
config.yml                # Your custom configuration
config.example.yml        # Example template
```

Specify the config file:

```bash
# Binary
hallmonitor --config config.yml

# Docker
docker run -d --network host --cap-add NET_RAW --cap-add NET_ADMIN \
  -v $(pwd)/config.yml:/etc/hallmonitor/config.yml:ro \
  ghcr.io/1broseidon/hallmonitor:latest

# Kubernetes (use Helm)
helm install hallmonitor ./k8s/helm/hallmonitor -f custom-values.yaml
```

## Configuration Examples

### Home Lab

```yaml
server:
  port: "8080"
  enableDashboard: true

logging:
  level: "info"
  format: "text"

monitoring:
  defaultInterval: "30s"
  defaultTimeout: "10s"

  groups:
    - name: "network"
      monitors:
        - type: "ping"
          name: "router"
          target: "192.168.1.1"

    - name: "services"
      monitors:
        - type: "http"
          name: "nas"
          url: "http://192.168.1.10:5000"
```

### Kubernetes Cluster

```yaml
server:
  port: "8080"
  enableDashboard: true

logging:
  level: "info"
  format: "json"

monitoring:
  defaultInterval: "30s"

  groups:
    - name: "cluster-services"
      monitors:
        - type: "http"
          name: "kubernetes-api"
          url: "https://kubernetes.default.svc.cluster.local/healthz"

        - type: "dns"
          name: "coredns"
          target: "10.96.0.10:53"
          query: "kubernetes.default.svc.cluster.local"
```

### Production Microservices

```yaml
server:
  port: "8080"
  enableDashboard: false  # Use Grafana instead

logging:
  level: "warn"
  format: "json"

metrics:
  enabled: true

monitoring:
  defaultInterval: "15s"
  defaultTimeout: "5s"

  groups:
    - name: "api-services"
      interval: "10s"
      monitors:
        - type: "http"
          name: "user-api"
          url: "http://user-service:8080/health"
          labels:
            criticality: "high"

        - type: "http"
          name: "order-api"
          url: "http://order-service:8080/health"
          labels:
            criticality: "high"
```

## Best Practices

### Naming Conventions

Use consistent, descriptive names:

```yaml
# Good
name: "payment-api-production"
name: "postgres-primary"
name: "router-datacenter-1"

# Avoid
name: "check1"
name: "test"
name: "monitor"
```

### Interval Selection

- **Critical services**: 5-15 seconds
- **Important services**: 15-30 seconds
- **Regular services**: 30-60 seconds
- **Background checks**: 60-300 seconds

### Timeout Settings

- Timeout should be less than interval
- Add buffer for network latency
- Consider slow startup times

```yaml
# Good: Timeout < Interval
interval: "30s"
timeout: "10s"

# Bad: Timeout ≥ Interval
interval: "10s"
timeout: "15s"  # Checks will overlap!
```

### Label Usage

Add labels for organization and filtering:

```yaml
labels:
  environment: "production"
  team: "platform"
  criticality: "high"
  region: "us-east-1"
```

## Next Steps

- [First Monitor](./first-monitor.md) - Set up your first monitor
- [Monitor Types](../03-monitors/index.md) - Learn about each monitor type
