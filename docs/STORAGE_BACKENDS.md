# Storage Backend Guide

HallMonitor supports multiple storage backends for persisting monitor results and historical data. This guide covers all available backends and how to configure them.

## Available Backends

### 1. BadgerDB (Embedded - Default)

**Best for:** Small to medium deployments, single-node setups, simple installations

BadgerDB is an embedded key-value database written in pure Go. It stores all data locally on disk and requires no external dependencies.

**Advantages:**
- No external dependencies
- Simple setup - just configure a file path
- Good performance for most use cases
- Built-in TTL (retention) management
- Zero configuration database server

**Configuration:**

```yaml
storage:
  backend: "badger"
  badger:
    enabled: true
    path: "./data/hallmonitor.db"
    retentionDays: 30
    enableAggregation: true
```

### 2. PostgreSQL / TimescaleDB

**Best for:** Enterprise deployments, existing PostgreSQL infrastructure, SQL queries

PostgreSQL is a powerful relational database with ACID compliance. TimescaleDB (optional) adds time-series optimizations.

**Advantages:**
- Familiar SQL interface
- Proven enterprise reliability
- Easy integration with existing PostgreSQL infrastructure
- Support for complex queries
- Optional TimescaleDB extension for time-series optimizations
- Centralized storage for multiple HallMonitor instances

**Configuration:**

```yaml
storage:
  backend: "postgres"
  postgres:
    host: "localhost"
    port: 5432
    database: "hallmonitor"
    user: "hallmonitor"
    password: "${POSTGRES_PASSWORD}"  # Use environment variable
    sslmode: "require"                # Use "disable" for local dev only
    retentionDays: 90
```

**Setup:**

```bash
# 1. Install dependencies
go get github.com/jackc/pgx/v5/pgxpool

# 2. Create database
createdb hallmonitor

# 3. Configure environment
export POSTGRES_PASSWORD="your-secure-password"

# 4. Start HallMonitor (schema created automatically)
./hallmonitor -config config.yml
```

**Optional: Enable TimescaleDB**

For deployments with >100 monitors or high-frequency checks, TimescaleDB provides automatic partitioning and compression:

```sql
-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Convert tables to hypertables
SELECT create_hypertable('monitor_results', 'timestamp', if_not_exists => TRUE);
SELECT create_hypertable('monitor_aggregates', 'period_start', if_not_exists => TRUE);

-- Optional: Enable compression for older data
ALTER TABLE monitor_results SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'monitor'
);

SELECT add_compression_policy('monitor_results', INTERVAL '7 days');
```

### 3. InfluxDB

**Best for:** Time-series focused deployments, high write throughput, existing InfluxDB infrastructure

InfluxDB is purpose-built for time-series data with efficient storage and powerful query capabilities.

**Advantages:**
- Optimized for time-series data
- Excellent compression
- Built-in retention policies
- Powerful Flux query language
- Continuous aggregation support
- Downsampling for long-term storage

**Configuration:**

```yaml
storage:
  backend: "influxdb"
  influxdb:
    url: "http://localhost:8086"
    token: "${INFLUXDB_TOKEN}"
    org: "hallmonitor"
    bucket: "monitor_results"
```

**Setup:**

```bash
# 1. Install dependencies
go get github.com/influxdata/influxdb-client-go/v2

# 2. Install InfluxDB (using Docker)
docker run -d --name influxdb \
  -p 8086:8086 \
  -e DOCKER_INFLUXDB_INIT_MODE=setup \
  -e DOCKER_INFLUXDB_INIT_USERNAME=hallmonitor \
  -e DOCKER_INFLUXDB_INIT_PASSWORD=hallmonitor \
  -e DOCKER_INFLUXDB_INIT_ORG=hallmonitor \
  -e DOCKER_INFLUXDB_INIT_BUCKET=monitor_results \
  -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=your-secure-token \
  influxdb:2.7

# 3. Configure environment
export INFLUXDB_TOKEN="your-secure-token"

# 4. Start HallMonitor
./hallmonitor -config config.yml
```

**Optional: Configure Retention Policies**

InfluxDB supports automatic data retention at the bucket level:

```bash
# Create bucket with 30-day retention
influx bucket create \
  --name monitor_results \
  --org hallmonitor \
  --retention 30d
```

### 4. None (Metrics Only)

**Best for:** Prometheus-native deployments, ephemeral data

When set to "none", HallMonitor only exposes Prometheus metrics without storing historical data.

**Advantages:**
- Minimal disk usage
- No database management
- Perfect for Prometheus scraping
- Lowest resource overhead

**Configuration:**

```yaml
storage:
  backend: "none"
```

**Note:** Historical queries and uptime calculations will be disabled in this mode.

## Comparison Matrix

| Feature | BadgerDB | PostgreSQL | InfluxDB | None |
|---------|----------|------------|----------|------|
| Setup Complexity | ⭐ Simple | ⭐⭐⭐ Moderate | ⭐⭐ Easy | ⭐ Trivial |
| External Dependencies | None | PostgreSQL | InfluxDB | None |
| SQL Queries | ❌ | ✅ | ❌ | ❌ |
| Time-series Optimization | ⭐⭐ Good | ⭐⭐⭐ Excellent (with TimescaleDB) | ⭐⭐⭐⭐ Best | N/A |
| Clustering Support | ❌ | ✅ | ✅ | N/A |
| Resource Usage | ⭐⭐⭐ Low | ⭐⭐ Medium | ⭐⭐ Medium | ⭐⭐⭐⭐ Minimal |
| Historical Data | ✅ | ✅ | ✅ | ❌ |
| Retention Management | Auto | Auto | Built-in | N/A |

## Migration Between Backends

### Export from BadgerDB

```bash
# Coming soon: Migration tool
./hallmonitor migrate export --output /tmp/export.json
```

### Import to PostgreSQL/InfluxDB

```bash
# Coming soon: Migration tool
./hallmonitor migrate import --input /tmp/export.json --backend postgres
```

## Testing

Run integration tests for storage backends:

```bash
# Start test databases
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test -tags=integration ./internal/storage/...

# Cleanup
docker-compose -f docker-compose.test.yml down -v
```

## Production Best Practices

### Security

1. **Never commit credentials** - Use environment variables
2. **Enable SSL/TLS** - Set `sslmode=require` for PostgreSQL
3. **Rotate tokens** - Regularly rotate InfluxDB API tokens
4. **Least privilege** - Create dedicated database users with minimal permissions
5. **Network isolation** - Use firewalls or VPNs for database access

### Performance

**PostgreSQL:**
- Adjust connection pool size based on monitor count (default: max 10, min 2)
- Use TimescaleDB for >100 monitors or <1min check intervals
- Create proper indexes (done automatically by schema)
- Monitor query performance with `EXPLAIN ANALYZE`

**InfluxDB:**
- Leverage automatic batching (writes are async)
- Use downsampling for long-term data retention
- Configure retention policies at bucket level
- Monitor write queue with InfluxDB metrics

**BadgerDB:**
- Ensure sufficient disk I/O capacity
- Place database on SSD for best performance
- Monitor disk usage growth

### Monitoring

Add health checks for your storage backend:

```bash
# Check storage health via API
curl http://localhost:7878/api/v1/storage/health
```

### Backup and Recovery

**PostgreSQL:**
```bash
# Backup
pg_dump hallmonitor > hallmonitor_backup.sql

# Restore
psql hallmonitor < hallmonitor_backup.sql
```

**InfluxDB:**
```bash
# Backup
influx backup /path/to/backup

# Restore
influx restore /path/to/backup
```

**BadgerDB:**
```bash
# Backup (simple file copy when service is stopped)
tar -czf hallmonitor_backup.tar.gz ./data/hallmonitor.db
```

## Troubleshooting

### PostgreSQL Connection Issues

**Error:** `failed to ping database: connection refused`

**Solution:**
- Verify PostgreSQL is running: `systemctl status postgresql`
- Check firewall rules allow connection
- Verify connection string parameters (host, port)

### InfluxDB Write Errors

**Error:** `InfluxDB write error: unauthorized`

**Solution:**
- Verify token is correct
- Check token has write permissions to bucket
- Confirm organization name matches

### BadgerDB Disk Space

**Error:** `no space left on device`

**Solution:**
- Reduce retention period
- Increase disk space
- Enable aggregation and disable raw results

## Getting Help

- GitHub Issues: https://github.com/1broseidon/hallmonitor/issues
- PostgreSQL Docs: https://www.postgresql.org/docs/
- InfluxDB Docs: https://docs.influxdata.com/
- TimescaleDB Docs: https://docs.timescale.com/
