# Persistent Storage

Hall Monitor supports persistent storage of monitoring results using BadgerDB, enabling historical data analysis and uptime tracking across restarts.

## Overview

By default, Hall Monitor stores monitoring results in memory. When persistent storage is enabled, results are saved to disk using BadgerDB, allowing you to:

- View historical uptime data
- Track trends over time
- Analyze past incidents
- Generate uptime reports
- Maintain data across restarts

## Configuration

### Enable Storage

Add the storage section to your `config.yml`:

```yaml
storage:
  enabled: true
  path: "./data/hallmonitor.db"
  retentionDays: 30
  enableAggregation: true
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `true` | Enable persistent storage |
| `path` | string | `"./data/hallmonitor.db"` | Database file path |
| `retentionDays` | integer | `30` | Days to retain raw check results |
| `enableAggregation` | boolean | `true` | Enable hourly/daily aggregation |

### Retention Policy

- **Raw results**: Stored for the configured `retentionDays` period
- **Hourly aggregates**: Stored for 2x the retention period
- **Daily aggregates**: Stored for 365 days

Data is automatically deleted using BadgerDB's built-in TTL (time-to-live) mechanism.

## Storage Architecture

### Data Types

1. **Raw Check Results** - Every monitor check stored with full details
2. **Hourly Aggregates** - Statistics computed every hour
3. **Daily Aggregates** - Daily uptime summaries

### Key Schema

```
result:{monitor}:{timestamp}     # Raw check result
agg:hour:{monitor}:{timestamp}   # Hourly aggregate
agg:day:{monitor}:{timestamp}    # Daily aggregate
latest:{monitor}                 # Latest result (cached)
```

### Aggregation

When enabled, Hall Monitor automatically generates aggregates containing:

- Total checks performed
- Number of up/down checks
- Uptime percentage
- Min/max/average response times

Aggregation runs hourly in the background without impacting monitoring.

## API Endpoints

### Historical Results

Query raw check results for a time range:

```bash
GET /api/v1/monitors/:name/history?start=<RFC3339>&end=<RFC3339>&limit=<number>
```

**Example:**

```bash
curl "http://localhost:7878/api/v1/monitors/gitlab/history?start=2025-11-01T00:00:00Z&end=2025-11-07T23:59:59Z&limit=1000"
```

**Response:**

```json
{
  "monitor": "gitlab",
  "start": "2025-11-01T00:00:00Z",
  "end": "2025-11-07T23:59:59Z",
  "results": [
    {
      "monitor": "gitlab",
      "type": "http",
      "status": "up",
      "duration": 150000000,
      "timestamp": "2025-11-07T10:30:00Z"
    }
  ],
  "total": 1234
}
```

### Uptime Statistics

Get uptime percentage for a period:

```bash
GET /api/v1/monitors/:name/uptime?period=<duration>
```

**Examples:**

```bash
# 24 hours
curl "http://localhost:7878/api/v1/monitors/gitlab/uptime?period=24h"

# 7 days
curl "http://localhost:7878/api/v1/monitors/gitlab/uptime?period=168h"

# 30 days
curl "http://localhost:7878/api/v1/monitors/gitlab/uptime?period=720h"
```

**Response:**

```json
{
  "monitor": "gitlab",
  "period": "24h",
  "start": "2025-11-06T10:30:00Z",
  "end": "2025-11-07T10:30:00Z",
  "total_checks": 2880,
  "up_checks": 2875,
  "down_checks": 5,
  "uptime_percent": 99.826
}
```

## Dashboard Integration

When storage is enabled, the built-in dashboards automatically display historical data:

### Uptime Heatmap

- Shows daily uptime for the past 7/30/90 days
- Color-coded cells indicate uptime percentage
- Gray cells indicate missing data (Hall Monitor not running)

### Statistics Panel

The dashboard loads uptime statistics for multiple periods:

- Last 24 hours
- Last 7 days
- Last 30 days

### Time Range Selection

Use the time range buttons to view different historical periods:

- 7 days
- 30 days
- 90 days

## Storage Management

### Disk Space Requirements

Calculate storage needs based on your monitoring frequency:

```
Storage (GB) = (Monitors × Checks/Day × Retention Days × Bytes/Check) / 1,000,000,000

Example:
- 10 monitors
- 2,880 checks/day (30s interval)
- 30 days retention
- ~200 bytes per check
= 10 × 2,880 × 30 × 200 / 1,000,000,000
≈ 1.7 GB
```

### Backup and Restore

#### Backup

```bash
# Stop Hall Monitor
systemctl stop hallmonitor

# Backup database
cp -r ./data/hallmonitor.db ./backup/hallmonitor-$(date +%Y%m%d).db

# Start Hall Monitor
systemctl start hallmonitor
```

#### Restore

```bash
# Stop Hall Monitor
systemctl stop hallmonitor

# Restore database
cp -r ./backup/hallmonitor-20251107.db ./data/hallmonitor.db

# Start Hall Monitor
systemctl start hallmonitor
```

### Clean Start

To delete all historical data:

```bash
# Stop Hall Monitor
systemctl stop hallmonitor

# Delete database
rm -rf ./data/hallmonitor.db

# Start Hall Monitor (fresh database will be created)
systemctl start hallmonitor
```

## Performance

### Write Performance

- Writes are asynchronous and non-blocking
- Monitoring is never impacted by storage operations
- ~10,000 writes/second throughput

### Read Performance

- Latest results: ~1ms (in-memory cache)
- Historical queries: ~10-50ms (disk scan)
- Aggregates: ~5ms (pre-computed)

### Memory Usage

- In-memory cache: ~10MB (1000 results per monitor)
- BadgerDB cache: ~40-50MB
- Total overhead: ~50-60MB typical

## Troubleshooting

### Storage Not Enabled

If historical data is not appearing:

1. Check `storage.enabled` is `true` in config
2. Verify database path is writable
3. Check logs for storage initialization errors

```bash
# Check if storage initialized
grep "storage initialized" /var/log/hallmonitor.log
```

### Database Corruption

If BadgerDB becomes corrupted:

```bash
# Stop Hall Monitor
systemctl stop hallmonitor

# Delete corrupted database
rm -rf ./data/hallmonitor.db

# Start Hall Monitor (fresh database)
systemctl start hallmonitor
```

Historical data will be lost, but monitoring continues normally.

### Disk Space Issues

If running out of disk space:

1. Reduce `retentionDays` in configuration
2. Disable aggregation if not needed
3. Increase monitor check intervals
4. Add disk space monitoring

```yaml
storage:
  enabled: true
  retentionDays: 7  # Reduce from 30 to 7 days
  enableAggregation: false  # Disable if not needed
```

## Disabling Storage

To run Hall Monitor without persistent storage:

```yaml
storage:
  enabled: false
```

With storage disabled:
- Data is stored in memory only
- Historical queries return empty results
- Dashboards fall back to current data
- Lower memory usage
- No disk I/O

Existing data on disk is preserved but not used.

## Best Practices

### Retention Period

Choose retention based on your needs:

- **7 days**: Short-term troubleshooting
- **30 days**: Monthly reporting and trends
- **90 days**: Quarterly analysis and compliance

### Aggregation

Enable aggregation for:
- Long-term trend analysis
- Efficient queries over large time ranges
- Reduced storage for older data

Disable aggregation for:
- Memory-constrained environments
- Short retention periods (< 7 days)
- Minimal historical analysis needs

### Monitoring Storage

Monitor the storage system itself:

```yaml
monitoring:
  groups:
    - name: "hallmonitor-health"
      monitors:
        - type: "http"
          name: "hallmonitor-api"
          url: "http://localhost:7878/health"
```

Check disk usage:

```bash
# Check database size
du -sh ./data/hallmonitor.db

# Monitor disk space
df -h /path/to/data
```

## Migration

### From Previous Versions

Upgrading to v0.2.0+ automatically enables storage. No migration needed:

1. Add `storage` section to config (or use defaults)
2. Restart Hall Monitor
3. Historical data begins accumulating

### Changing Retention

To change retention period:

```yaml
storage:
  retentionDays: 90  # Changed from 30 to 90
```

Restart Hall Monitor. Existing data retention will be updated on next write.

## Next Steps

- [Dashboard Guide](./dashboard.md) - View historical data in dashboards
- [Metrics Reference](./metrics.md) - Prometheus metrics for long-term storage
- [Troubleshooting](../05-reference/troubleshooting.md) - Common issues

