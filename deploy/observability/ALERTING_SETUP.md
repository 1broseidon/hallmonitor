# Alerting Setup Guide - Discord Integration

## Overview

Hall Monitor includes a comprehensive alerting system that sends notifications to Discord via Alertmanager. This guide walks you through the complete setup.

## Architecture

```
Hall Monitor → Prometheus → Alertmanager → Discord Adapter → Discord
               (metrics)     (alerts)       (translator)      (notifications)
```

**Components:**
- **Prometheus**: Evaluates alert rules against metrics
- **Alertmanager**: Routes and groups alerts
- **Discord Adapter**: Translates Alertmanager webhooks to Discord format
- **Discord**: Receives formatted alert notifications

## Quick Start

### 1. Get a Discord Webhook URL

#### Creating a Discord Webhook:

1. Open Discord and navigate to your server
2. Right-click on the channel where you want alerts
3. Select **Edit Channel** → **Integrations** → **Webhooks**
4. Click **New Webhook** or **Create Webhook**
5. Configure the webhook:
   - **Name**: `Hall Monitor Alerts`
   - **Avatar**: (optional) Upload a custom icon
6. Click **Copy Webhook URL**
7. Save this URL - you'll need it shortly

**Example webhook URL format:**
```
https://discord.com/api/webhooks/123456789012345678/AbCdEfGhIjKlMnOpQrStUvWxYz
```

### 2. Configure the Webhook

Add the webhook URL to your `.env` file:

```bash
# Open .env file
nano .env

# Add these lines (replace with your actual webhook URL)
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN

# Optional: Separate webhooks for different severity levels
DISCORD_CRITICAL_WEBHOOK_URL=https://discord.com/api/webhooks/YOUR_CRITICAL_WEBHOOK
DISCORD_WARNING_WEBHOOK_URL=https://discord.com/api/webhooks/YOUR_WARNING_WEBHOOK
```

**Security Note**: Never commit `.env` files with webhook URLs to version control!

### 3. Start the Full Stack

```bash
# Restart the observability stack to pick up the new configuration
make docker-down
make docker-full
```

### 4. Verify the Setup

Check that all services are running:

```bash
# Check all containers
docker compose -f docker-compose.yml -f docker-compose.full.yml ps

# Check Discord adapter health
curl http://localhost:5001/

# Check Alertmanager
curl http://localhost:19093/api/v1/status

# Check Prometheus alerts
curl http://localhost:19090/api/v1/rules
```

## Alert Rules

Hall Monitor includes **60+ alert rules** covering:

### 1. Health Monitoring
- **MonitorDown**: Monitor has been down for >1 minute
- **MonitorFlapping**: Monitor changing state frequently
- **HighCheckFailureRate**: >50% of checks failing

### 2. Performance
- **HighResponseTime**: P95 response time >1s
- **VeryHighResponseTime**: P95 response time >5s (critical)
- **HighHTTPResponseTime**: HTTP-specific response time alerts

### 3. SSL/TLS
- **SSLCertificateExpiringSoon**: Certificate expires in <30 days
- **SSLCertificateExpiringCritical**: Certificate expires in <7 days

### 4. Network
- **HighPacketLoss**: >5% packet loss
- **CriticalPacketLoss**: >20% packet loss
- **HighPingRTT**: P95 ping RTT >500ms

### 5. DNS
- **DNSQueryFailures**: >10% failed DNS queries
- **SlowDNSQueries**: P95 query time >1s

### 6. HTTP Errors
- **HighHTTP4xxRate**: >10% 4xx responses
- **HighHTTP5xxRate**: >5% 5xx responses

### 7. System Health
- **NoChecksRunning**: Hall Monitor stopped performing checks
- **HighErrorRate**: >20% error rate across all monitors
- **ManyMonitorsDown**: >30% of monitors down

## Testing Alerts

### Test 1: Manual Alert Trigger

Create a temporary monitor that will fail:

```bash
# Add to config.yml
monitors:
  - type: http
    name: test-alert
    target: "http://localhost:99999"  # Invalid port
    interval: "10s"
    timeout: "5s"
```

This will trigger a `MonitorDown` alert after 1 minute.

### Test 2: Use Alertmanager API

Send a test alert directly to Alertmanager:

```bash
curl -X POST http://localhost:19093/api/v1/alerts -H 'Content-Type: application/json' -d '
[
  {
    "labels": {
      "alertname": "TestAlert",
      "severity": "warning",
      "monitor": "test",
      "component": "testing"
    },
    "annotations": {
      "summary": "This is a test alert",
      "description": "Testing Discord webhook integration"
    }
  }
]
'
```

### Test 3: Simulate High Response Time

Modify a monitor's timeout to be very short:

```yaml
- type: http
  name: gitlab
  target: "https://gitlab.com"
  interval: "10s"
  timeout: "100ms"  # Very short timeout
```

This should trigger `HighResponseTime` alerts.

## Discord Alert Format

Alerts appear in Discord with rich formatting:

**Critical Alert Example:**
```
🔥 CRITICAL: MonitorDown

Monitor: gitlab
Component: http
Severity: CRITICAL

Details: Monitor gitlab (type: http, group: critical-services) 
has been down for more than 1 minute.

Dashboard: [View Dashboard](http://localhost:3000/d/hallmonitor-overview)
```

**Resolved Alert Example:**
```
✅ RESOLVED: MonitorDown

Monitor: gitlab
Component: http
```

## Alert Routing

Alertmanager groups and routes alerts intelligently:

### Grouping Rules
- Alerts grouped by: `alertname`, `severity`, `component`
- **Group Wait**: 30s (critical: 10s) - Wait before sending first alert
- **Group Interval**: 5m - Wait between sending grouped updates
- **Repeat Interval**: 12h (critical: 4h) - Resend unresolved alerts

### Routing Logic
1. **Critical alerts** → Immediate notification (10s wait)
2. **Warning alerts** → Standard notification (30s wait)
3. **Info alerts** → Default receiver

### Inhibition Rules

Prevents alert spam by suppressing redundant alerts:

1. **Critical inhibits Warning**: If critical alert fires, warning is suppressed
2. **System-wide inhibits individual**: If many monitors down, individual alerts suppressed
3. **No checks inhibits all**: If system not running, all monitor alerts suppressed

## Customization

### Modifying Alert Rules

Edit alert rules:
```bash
nano deploy/observability/prometheus/rules/hallmonitor_alerts.yml
```

**Example: Change response time threshold:**
```yaml
- alert: HighResponseTime
  expr: |
    histogram_quantile(0.95,
      sum(rate(hallmonitor_check_duration_seconds_bucket[5m])) by (le, monitor, type, group)
    ) > 2  # Changed from 1 to 2 seconds
  for: 10m  # Changed from 5m to 10m
```

Reload Prometheus:
```bash
curl -X POST http://localhost:19090/-/reload
```

### Creating Custom Alerts

Add new alert rules to the appropriate group:

```yaml
- alert: CustomAlert
  expr: your_promql_expression > threshold
  for: 5m
  labels:
    severity: warning
    component: custom
  annotations:
    summary: "Custom alert summary"
    description: "Detailed description with {{ $value }}"
    dashboard: "http://localhost:3000/d/hallmonitor-overview"
```

### Multiple Discord Channels

Configure separate webhooks for different alert types:

```bash
# In .env
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/general/token
DISCORD_CRITICAL_WEBHOOK_URL=https://discord.com/api/webhooks/critical/token
DISCORD_WARNING_WEBHOOK_URL=https://discord.com/api/webhooks/warnings/token
```

Update `alertmanager.yml` to use different receivers.

## Troubleshooting

### Alerts Not Appearing in Discord

1. **Check Discord Adapter**:
   ```bash
   docker logs discord-adapter
   ```

2. **Verify Webhook URL**:
   ```bash
   # Test webhook directly
   curl -X POST "${DISCORD_WEBHOOK_URL}" \
     -H "Content-Type: application/json" \
     -d '{"content": "Test message"}'
   ```

3. **Check Alertmanager**:
   ```bash
   # View active alerts
   curl http://localhost:19093/api/v1/alerts
   
   # View Alertmanager status
   docker logs alertmanager
   ```

4. **Check Prometheus Rules**:
   ```bash
   # View alert rules status
   curl http://localhost:19090/api/v1/rules | jq '.data.groups[].rules[] | select(.type=="alerting")'
   ```

### Webhook 401 Unauthorized

- Discord webhook URLs expire if the webhook is deleted
- Regenerate webhook in Discord and update `.env`

### Too Many Alerts

Adjust thresholds or increase `for` duration:
```yaml
for: 10m  # Wait 10 minutes before alerting
```

Adjust repeat interval:
```yaml
repeat_interval: 24h  # Repeat only once per day
```

### Missing Alerts

Check Prometheus evaluation:
```bash
# Check if rule is evaluating
curl 'http://localhost:19090/api/v1/query?query=ALERTS{alertname="MonitorDown"}'
```

Verify alert is reaching Alertmanager:
```bash
curl http://localhost:19093/api/v1/alerts | jq
```

## Advanced Configuration

### Silence Alerts

Temporarily silence specific alerts:

```bash
# Via Alertmanager UI
open http://localhost:19093

# Or via API
curl -X POST http://localhost:19093/api/v1/silences -d '{
  "matchers": [
    {"name": "alertname", "value": "MonitorDown", "isRegex": false},
    {"name": "monitor", "value": "gitlab", "isRegex": false}
  ],
  "startsAt": "2025-11-04T00:00:00Z",
  "endsAt": "2025-11-05T00:00:00Z",
  "createdBy": "admin",
  "comment": "Scheduled maintenance"
}'
```

### Alert Dependencies

Create alert dependencies using inhibition rules:

```yaml
inhibit_rules:
  - source_match:
      alertname: 'DatabaseDown'
    target_match:
      alertname: 'APIDown'
    equal: ['environment']
```

### Time-Based Routing

Route alerts differently based on time of day (requires Alertmanager time-based routing):

```yaml
routes:
  - match:
      severity: critical
    receiver: pagerduty-oncall
    active_time_intervals:
      - business_hours
  - match:
      severity: warning
    receiver: discord-warnings
```

## Files Reference

- **Alert Rules**: `deploy/observability/prometheus/rules/hallmonitor_alerts.yml`
- **Alertmanager Config**: `deploy/observability/alertmanager/alertmanager.yml`
- **Discord Adapter**: `deploy/observability/alertmanager/discord-webhook-adapter.py`
- **Prometheus Config**: `deploy/observability/prometheus/prometheus.yml`
- **Environment**: `.env`

## Monitoring the Alerting System

- **Prometheus Targets**: http://localhost:19090/targets
- **Prometheus Rules**: http://localhost:19090/rules
- **Alertmanager Status**: http://localhost:19093/#/status
- **Alertmanager Alerts**: http://localhost:19093/#/alerts
- **Discord Adapter Health**: http://localhost:5001/

## Best Practices

1. **Test Alerts Regularly**: Use the testing methods above
2. **Tune Thresholds**: Adjust based on your environment
3. **Use Meaningful Names**: Make alert names descriptive
4. **Add Context**: Include monitor name, component, description
5. **Set Appropriate Severity**: Don't over-use critical
6. **Monitor Alert Volume**: Use Grafana to track alert rates
7. **Document Custom Rules**: Add comments explaining thresholds
8. **Version Control**: Commit alert rules to git
9. **Review Regularly**: Audit and update alert rules monthly
10. **Silence Maintenance**: Use silences for planned maintenance

## Next Steps

- [ ] Configure Discord webhook URL
- [ ] Start the full stack
- [ ] Test alerts with a failing monitor
- [ ] Customize thresholds for your environment
- [ ] Set up additional notification channels (PagerDuty, Slack, etc.)
- [ ] Create custom alert rules for your specific needs
- [ ] Configure alert silences for maintenance windows
- [ ] Document your alerting strategy

## Support

For issues:
1. Check Discord adapter logs: `docker logs discord-adapter`
2. Check Alertmanager logs: `docker logs alertmanager`
3. Verify Prometheus rules: `curl http://localhost:19090/api/v1/rules`
4. Test webhook manually with curl
5. Review this guide for troubleshooting steps

---

**Last Updated**: 2025-11-04  
**Version**: 1.0  
**Maintainer**: Hall Monitor Team

