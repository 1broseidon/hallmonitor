# Observability

Hall Monitor provides comprehensive observability through metrics, dashboards, and integration with popular monitoring tools.

## Overview

Hall Monitor offers multiple ways to observe your monitoring infrastructure:

1. **Built-in Dashboard** - Lightweight web UI for quick visibility
2. **Prometheus Metrics** - Standard metrics export for long-term storage
3. **Grafana Integration** - Rich visualization and alerting
4. **Webhooks** - Event notifications to external systems

## Built-in Dashboard

Hall Monitor includes a lightweight web dashboard accessible at the root path.

### Features

- Real-time monitor status
- Response time charts
- Uptime statistics
- Dark mode support
- Mobile responsive
- Auto-refresh every 30 seconds

### Access

Open your browser to:
```
http://localhost:8080/
```

See [Dashboard Documentation](./dashboard.md) for details.

## Prometheus Metrics

Hall Monitor exposes metrics in Prometheus format at `/metrics`.

### Key Metrics

```
# Monitor status (1=up, 0=down)
hallmonitor_monitor_up{monitor="api",group="services",type="http"} 1

# Response time in seconds
hallmonitor_http_response_time_seconds{monitor="api",group="services"} 0.123

# Check duration (including overhead)
hallmonitor_monitor_check_duration_seconds{monitor="api",group="services"} 0.125

# SSL certificate expiry timestamp
hallmonitor_http_ssl_cert_expiry_timestamp{monitor="api",common_name="api.example.com"} 1735689600
```

### Scraping Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'hallmonitor'
    static_configs:
      - targets: ['hallmonitor:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

See [Metrics Documentation](./metrics.md) for complete metric reference.

## Grafana Integration

### Pre-built Dashboards

Hall Monitor includes pre-configured Grafana dashboards:

1. **Overview Dashboard** - System-wide status and trends
2. **Monitor Details** - Drill-down for specific monitors
3. **SSL Certificate Tracking** - Certificate expiry monitoring

### Installation

Set up your own Grafana instance and connect it to Hall Monitor's `/metrics` endpoint using Prometheus as a data source.

You can create custom dashboards to visualize monitor status, response times, and alerting.

## Alerting

### Alert Configuration

Configure alerts in your `config.yml`:

```yaml
alerting:
  enabled: true
  evaluationInterval: "10s"
  rules:
    - name: "ServiceDown"
      expr: "hallmonitor_monitor_up == 0"
      for: "2m"
      labels:
        severity: "critical"
      annotations:
        summary: "Monitor {{.monitor}} is down"
```

### Webhooks

Send notifications to Discord, Slack, or custom webhooks:

```yaml
webhooks:
  - url: "${DISCORD_WEBHOOK}"
    events: ["down", "recovered"]

  - url: "${SLACK_WEBHOOK}"
    events: ["down"]
```

Configure alerting in your Prometheus Alertmanager instance.

## Full Observability Stack

Deploy Hall Monitor with complete observability using Docker Compose:

```bash
# Run Hall Monitor with Docker Compose
docker compose up -d

# Set up your own Prometheus to scrape http://localhost:7878/metrics
# Set up your own Grafana with Prometheus as a data source
```

This includes:
- **Hall Monitor** - Monitoring service
- **Prometheus** - Metrics storage
- **Grafana** - Visualization
- **Loki** - Log aggregation
- **Alertmanager** - Alert routing

Access:
- Hall Monitor: http://localhost:8080
- Grafana: http://localhost:3000 (admin/hallmonitor)
- Prometheus: http://localhost:19090
- Alertmanager: http://localhost:19093

## Kubernetes ServiceMonitor

For Prometheus Operator:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: hallmonitor
  namespace: hallmonitor
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: hallmonitor
  endpoints:
    - port: http
      path: /metrics
      interval: 15s
```

See [Installation Guide](../02-getting-started/installation.md) for Kubernetes deployment.

## Monitoring Best Practices

### Metric Labels

Use consistent labels across monitors:

```yaml
monitors:
  - type: "http"
    name: "api"
    url: "https://api.example.com"
    labels:
      environment: "production"
      team: "platform"
      criticality: "high"
```

### Alert Thresholds

Set appropriate thresholds:

- **ServiceDown**: 2-5 minutes (avoid false positives)
- **HighLatency**: Based on SLA requirements
- **SSLExpiring**: 30 days for warning, 7 days for critical

### Dashboard Organization

Create dashboards by:
- Environment (production, staging, development)
- Team ownership
- Service criticality
- Geographic region

## Next Steps

- [Metrics Reference](./metrics.md) - Complete metric documentation
- [Dashboard Guide](./dashboard.md) - Built-in dashboard features
- Set up your own Grafana with Prometheus data source
- Configure alerts in your Prometheus Alertmanager
