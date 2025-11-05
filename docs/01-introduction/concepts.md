# Core Concepts

Understanding the core concepts of Hall Monitor will help you configure and use it effectively.

## Monitors

A **monitor** is a single check that tests the availability or health of a target service or resource. Each monitor has:

- **Type**: The protocol used (HTTP, TCP, DNS, or Ping)
- **Name**: A unique identifier for the monitor
- **Target**: What to monitor (URL, hostname, IP address)
- **Interval**: How often to run the check
- **Timeout**: Maximum time to wait for a response
- **Status**: Current state (up, down, or unknown)

### Monitor Types

#### HTTP Monitor
Checks HTTP/HTTPS endpoints by making GET requests.

**Use cases**:
- Web application health checks
- API endpoint monitoring
- SSL certificate expiry tracking
- Response time monitoring

**Configuration**:
```yaml
- type: "http"
  name: "api-server"
  url: "https://api.example.com/health"
  expectedStatus: 200
  timeout: "5s"
```

#### TCP Monitor
Tests TCP port connectivity.

**Use cases**:
- Database connection checks
- Service port availability
- SSH access monitoring
- Mail server monitoring

**Configuration**:
```yaml
- type: "tcp"
  name: "database"
  target: "db.example.com:5432"
  timeout: "3s"
```

#### DNS Monitor
Queries DNS records and validates responses.

**Use cases**:
- DNS server health checks
- Record propagation verification
- Domain resolution monitoring
- Internal DNS validation

**Configuration**:
```yaml
- type: "dns"
  name: "dns-server"
  target: "8.8.8.8:53"
  query: "example.com"
  queryType: "A"
  timeout: "3s"
```

#### Ping Monitor
Tests host reachability using ICMP ping.

**Use cases**:
- Network connectivity checks
- Gateway monitoring
- Server reachability tests
- Network latency tracking

**Configuration**:
```yaml
- type: "ping"
  name: "gateway"
  target: "192.168.1.1"
  count: 3
  timeout: "3s"
```

**Note**: ICMP ping requires elevated privileges. Hall Monitor falls back to unprivileged mode (UDP) if ICMP is not available.

## Monitor Groups

A **monitor group** is a collection of related monitors that share common configuration. Groups help organize monitors logically and apply consistent settings.

### Benefits of Groups

- **Organization**: Group related services together
- **Shared Configuration**: Apply interval and timeout to all monitors in the group
- **Logical Separation**: Separate critical vs. non-critical services
- **Labels**: Add metadata for filtering and alerting

### Group Configuration

```yaml
monitoring:
  groups:
    - name: "critical-services"
      interval: "10s"  # All monitors in this group run every 10s
      monitors:
        - type: "http"
          name: "payment-api"
          url: "https://api.example.com/payment"

    - name: "infrastructure"
      interval: "30s"  # Different interval for this group
      monitors:
        - type: "ping"
          name: "router"
          target: "192.168.1.1"
```

### Configuration Inheritance

Monitors inherit configuration from their group and global defaults:

1. **Global Defaults** (lowest priority)
   ```yaml
   monitoring:
     defaultInterval: "30s"
     defaultTimeout: "10s"
   ```

2. **Group Configuration** (medium priority)
   ```yaml
   groups:
     - name: "my-group"
       interval: "15s"  # Overrides global default
   ```

3. **Monitor Configuration** (highest priority)
   ```yaml
   monitors:
     - type: "http"
       name: "my-monitor"
       interval: "5s"  # Overrides group and global
   ```

## Monitor Status

Each monitor can be in one of three states:

### Up
The monitor check succeeded according to its criteria:
- HTTP: Received expected status code
- TCP: Port is open and accepting connections
- DNS: Query returned expected records
- Ping: Host is reachable with acceptable packet loss

### Down
The monitor check failed:
- HTTP: Wrong status code, timeout, or connection error
- TCP: Port is closed or unreachable
- DNS: Query failed or returned unexpected results
- Ping: High packet loss (≥50%) or complete failure

### Unknown
The monitor has not been checked yet or the status cannot be determined. This is typically seen only during startup.

## Metrics and Observability

Hall Monitor exposes metrics in Prometheus format for long-term storage and visualization.

### Key Metrics

#### Monitor Status
```
hallmonitor_monitor_up{monitor="api-server",group="critical",type="http"} 1
```
Value: 1 (up) or 0 (down)

#### Response Time
```
hallmonitor_http_response_time_seconds{monitor="api-server",group="critical"} 0.123
```
Duration of the check in seconds

#### Check Duration
```
hallmonitor_monitor_check_duration_seconds{monitor="api-server",group="critical"} 0.125
```
Total time taken to execute the check including overhead

#### SSL Certificate Expiry
```
hallmonitor_http_ssl_cert_expiry_timestamp{monitor="api-server",common_name="example.com"} 1735689600
```
Unix timestamp when the SSL certificate expires

### Metric Labels

All metrics include standard labels:
- `monitor`: Monitor name
- `group`: Group name
- `type`: Monitor type (http, tcp, dns, ping)
- Custom labels from monitor configuration

## Scheduling and Execution

### Worker Pools

Hall Monitor uses goroutine-based worker pools to execute monitors concurrently:

- Each monitor runs in its own goroutine
- Monitors are scheduled independently based on their interval
- Timeouts prevent hanging checks from blocking others

### Check Intervals

Intervals control how frequently monitors run:

```yaml
monitoring:
  defaultInterval: "30s"  # Default for all monitors

  groups:
    - name: "critical"
      interval: "10s"  # Critical checks every 10 seconds
      monitors:
        - type: "http"
          name: "payment-api"
          interval: "5s"  # This monitor every 5 seconds
```

**Interval Guidelines**:
- **Critical services**: 5-15 seconds
- **Important services**: 15-30 seconds
- **Non-critical services**: 30-60 seconds
- **Background checks**: 60-300 seconds

### Timeouts

Timeouts prevent checks from running indefinitely:

```yaml
monitoring:
  defaultTimeout: "10s"  # Default timeout

  groups:
    - name: "fast-services"
      monitors:
        - type: "http"
          name: "local-api"
          timeout: "2s"  # Quick timeout for local services

        - type: "http"
          name: "external-api"
          timeout: "15s"  # Longer timeout for external services
```

**Timeout Guidelines**:
- **Local services**: 1-3 seconds
- **Internal network**: 3-5 seconds
- **Internet services**: 5-15 seconds
- **Slow services**: 15-30 seconds

Never set timeout greater than interval to avoid overlapping checks.

## Result Storage

Monitor check results are stored in-memory for quick access:

- **Recent Results**: Last 100 results per monitor
- **Current Status**: Most recent check result
- **History**: Used for dashboard and API queries
- **No Persistence**: Results are not saved to disk

For long-term storage, use Prometheus to scrape metrics.

## Labels and Metadata

Labels add metadata to monitors for organization and filtering:

```yaml
monitors:
  - type: "http"
    name: "payment-api"
    url: "https://api.example.com/payment"
    labels:
      severity: "critical"
      team: "platform"
      environment: "production"
```

Labels are:
- Exported as Prometheus metric labels
- Available in the API responses
- Used for filtering in dashboards
- Helpful for alert routing

## Configuration Management

### Environment Variable Substitution

Use environment variables in your configuration:

```yaml
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

Set variables:
```bash
export API_URL="https://api.example.com"
export API_TOKEN="secret-token-here"
```

### Configuration Validation

Hall Monitor validates configuration on startup:
- Required fields are present
- Monitor names are unique
- Intervals and timeouts are reasonable
- URLs and targets are valid
- Query types are supported

Invalid configuration prevents startup with a clear error message.

## Next Steps

- [Use Cases](./use-cases.md) - Common monitoring scenarios
- [Getting Started](../02-getting-started/index.md) - Install and configure
- [Configuration Basics](../02-getting-started/configuration-basics.md) - Complete configuration guide
