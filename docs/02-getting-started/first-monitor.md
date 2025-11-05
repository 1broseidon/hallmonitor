# Your First Monitor

This guide walks you through setting up your first monitor step-by-step.

## Step 1: Choose What to Monitor

Let's start with something simple - monitoring a public website. We'll use `https://example.com` which is designed to be always available.

## Step 2: Create Configuration File

If you haven't already, create a configuration file:

```bash
# Copy the example
cp config.example.yml config.yml

# Or create a new file
nano config.yml
```

## Step 3: Add Basic Configuration

Add the server and logging configuration:

```yaml
server:
  port: "8080"
  host: "0.0.0.0"
  enableDashboard: true

logging:
  level: "info"
  format: "text"
  output: "stdout"

metrics:
  enabled: true
  path: "/metrics"
```

## Step 4: Add Your First Monitor

Add a monitoring section with a single HTTP monitor:

```yaml
monitoring:
  defaultInterval: "30s"
  defaultTimeout: "10s"

  groups:
    - name: "my-first-monitors"
      monitors:
        - type: "http"
          name: "example-website"
          url: "https://example.com"
          expectedStatus: 200
          enabled: true
```

## Step 5: Start Hall Monitor

### Using Binary

```bash
./hallmonitor --config config.yml
```

### Using Docker

```bash
# First, ensure you have config.yml in current directory
cp config.example.yml config.yml
# Edit config.yml with your monitor

docker run -d \
  --name hallmonitor \
  --network host \
  --cap-add NET_RAW \
  --cap-add NET_ADMIN \
  -v $(pwd)/config.yml:/etc/hallmonitor/config.yml:ro \
  ghcr.io/1broseidon/hallmonitor:latest
```

### Using Docker Compose

```bash
# Ensure config.yml has your monitor
docker compose up -d
```

## Step 6: Verify It's Working

### Check the Health Endpoint

```bash
curl http://localhost:8080/health
```

Expected output:
```json
{
  "status": "healthy",
  "timestamp": "2025-01-04T10:30:00Z"
}
```

### View Monitor Status

```bash
curl http://localhost:8080/api/v1/monitors | jq
```

Expected output:
```json
[
  {
    "monitor": "example-website",
    "type": "http",
    "group": "my-first-monitors",
    "status": "up",
    "duration": "125ms",
    "timestamp": "2025-01-04T10:30:00Z"
  }
]
```

### Check the Dashboard

Open your browser to:
```
http://localhost:8080
```

You should see your monitor listed with a green "UP" status.

### View Metrics

```bash
curl http://localhost:8080/metrics | grep example
```

Expected output:
```
hallmonitor_monitor_up{group="my-first-monitors",monitor="example-website",type="http"} 1
hallmonitor_http_response_time_seconds{group="my-first-monitors",monitor="example-website"} 0.125
```

## Step 7: Add More Monitors

Now that your first monitor is working, let's add more:

```yaml
monitoring:
  defaultInterval: "30s"
  defaultTimeout: "10s"

  groups:
    - name: "my-first-monitors"
      monitors:
        # HTTP monitor (already added)
        - type: "http"
          name: "example-website"
          url: "https://example.com"
          expectedStatus: 200

        # DNS monitor
        - type: "dns"
          name: "google-dns"
          target: "8.8.8.8:53"
          query: "example.com"
          queryType: "A"

        # Ping monitor
        - type: "ping"
          name: "cloudflare-dns"
          target: "1.1.1.1"
          count: 3
```

Restart Hall Monitor to load the new configuration:

```bash
# Binary
# Stop with Ctrl+C, then restart
./hallmonitor --config config.yml

# Docker Compose
docker compose restart

# Kubernetes
kubectl rollout restart deployment/hallmonitor -n hallmonitor
```

## Understanding Monitor Results

### Monitor Status

Each monitor can be:
- **Up (green)**: Check succeeded
- **Down (red)**: Check failed
- **Unknown (gray)**: Not checked yet

### Response Time

The time it took to complete the check:
- HTTP: Time to receive response
- TCP: Time to establish connection
- DNS: Time to resolve query
- Ping: Average round-trip time

### Check Interval

How often the monitor runs:
- Configured via `interval` field
- Default from `defaultInterval`
- Shows in dashboard as "Last checked X seconds ago"

## Troubleshooting

### Monitor Shows as Down

**HTTP Monitor**:
```yaml
# Check if URL is correct
- type: "http"
  name: "my-site"
  url: "https://example.com"  # Must be full URL with protocol
  expectedStatus: 200          # Must match actual response
```

**DNS Monitor**:
```yaml
# Ensure DNS server is reachable
- type: "dns"
  name: "dns-check"
  target: "8.8.8.8:53"  # Must include port
  query: "example.com"   # Must be valid domain
```

**Ping Monitor**:
```yaml
# Note: Requires elevated privileges for ICMP
- type: "ping"
  name: "gateway"
  target: "192.168.1.1"  # Must be reachable IP or hostname
```

### Monitor Not Appearing

1. Check configuration syntax:
```bash
# Validate YAML
python3 -c "import yaml; yaml.safe_load(open('config.yml'))"
```

2. Check Hall Monitor logs:
```bash
# Docker Compose
docker compose logs hallmonitor

# Kubernetes
kubectl logs -f deployment/hallmonitor -n hallmonitor

# Binary
# Logs go to stdout/stderr
```

3. Verify monitor is enabled:
```yaml
- type: "http"
  name: "my-monitor"
  url: "https://example.com"
  enabled: true  # Must be true or omitted
```

### Timeout Errors

If monitors are timing out:

```yaml
monitoring:
  defaultTimeout: "15s"  # Increase from 10s

  groups:
    - name: "slow-services"
      monitors:
        - type: "http"
          name: "slow-api"
          url: "https://slow.example.com"
          timeout: "30s"  # Override for this monitor
```

## Next Steps

### Add Real Monitors

Replace the example with your actual infrastructure:

```yaml
monitoring:
  groups:
    - name: "my-infrastructure"
      monitors:
        # Your router
        - type: "ping"
          name: "router"
          target: "192.168.1.1"

        # Your NAS or server
        - type: "http"
          name: "nas-web"
          url: "http://192.168.1.10:5000"

        # Your application
        - type: "http"
          name: "my-app"
          url: "https://myapp.example.com"
          headers:
            Authorization: "Bearer ${API_TOKEN}"
```

### Set Up Observability

1. **Enable Prometheus scraping**: See [Observability Overview](../04-observability/index.md)
2. **Create Grafana dashboards**: Set up your own Grafana with Prometheus data source
3. **Configure alerts**: Set up alerting in your Prometheus/Alertmanager instance

### Learn Monitor Types

Explore different monitor types:
- See [Monitor Types](../03-monitors/index.md) for HTTP, TCP, DNS, and Ping monitoring

### Production Deployment

- Use Helm with production values for high availability deployments
- See [Installation Guide](./installation.md) for Kubernetes deployment instructions

## Example Configurations

### Home Lab

```yaml
monitoring:
  defaultInterval: "30s"

  groups:
    - name: "network"
      monitors:
        - type: "ping"
          name: "router"
          target: "192.168.1.1"

    - name: "servers"
      monitors:
        - type: "http"
          name: "nas"
          url: "http://192.168.1.10:5000"

        - type: "tcp"
          name: "server-ssh"
          target: "192.168.1.20:22"
```

### Kubernetes Cluster

```yaml
monitoring:
  defaultInterval: "30s"

  groups:
    - name: "k8s-cluster"
      monitors:
        - type: "http"
          name: "k8s-api"
          url: "https://kubernetes.default.svc.cluster.local/healthz"

        - type: "dns"
          name: "coredns"
          target: "10.96.0.10:53"
          query: "kubernetes.default.svc.cluster.local"

        - type: "http"
          name: "my-app"
          url: "http://my-app.default.svc.cluster.local:8080/health"
```

### Web Application

```yaml
monitoring:
  defaultInterval: "15s"

  groups:
    - name: "web-app"
      monitors:
        - type: "http"
          name: "frontend"
          url: "https://www.example.com"

        - type: "http"
          name: "api"
          url: "https://api.example.com/health"

        - type: "tcp"
          name: "database"
          target: "db.example.com:5432"

        - type: "tcp"
          name: "redis"
          target: "redis.example.com:6379"
```

## Summary

You've successfully:
1. Created a configuration file
2. Added your first monitor
3. Started Hall Monitor
4. Verified it's working
5. Learned to troubleshoot common issues

Continue with:
- [Monitor Types](../03-monitors/index.md) - Detailed monitor documentation
- [Configuration Basics](./configuration-basics.md) - Advanced configuration
- [Use Cases](../01-introduction/use-cases.md) - Real-world examples
