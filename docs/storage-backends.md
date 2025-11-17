# Storage Backends

HallMonitor supports multiple storage backends for persisting monitor results and historical data. Choose the backend that best fits your infrastructure and requirements.

## Available Backends

### 1. BadgerDB (Embedded)
**Best for**: Single-server deployments, homelab setups, small-to-medium monitoring

**Pros**:
- No external dependencies
- Zero configuration needed
- Fast local access
- Built-in support for retention policies
- Suitable for most use cases

**Cons**:
- Single-server only (no horizontal scaling)
- Limited to local disk capacity
- No SQL query capabilities

**Configuration**:
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
**Best for**: Production deployments, enterprise environments, SQL analytics

**Pros**:
- Industry-standard relational database
- SQL query capabilities for custom reporting
- TimescaleDB extension for time-series optimization
- ACID compliance and reliability
- Integrates with existing PostgreSQL infrastructure
- Support for replication and backups

**Cons**:
- Requires external PostgreSQL server
- More complex setup than embedded options
- Higher resource usage

**Configuration**:
```yaml
storage:
  backend: "postgres"
  postgres:
    host: "localhost"
    port: 5432
    database: "hallmonitor"
    user: "hallmonitor"
    password: "${POSTGRES_PASSWORD}"  # Use env var
    sslmode: "require"
    retentionDays: 90
    enableTimescale: true  # Optional
```

**Setup Steps**:

1. Create database and user:
```sql
CREATE DATABASE hallmonitor;
CREATE USER hallmonitor WITH PASSWORD 'your-password';
GRANT ALL PRIVILEGES ON DATABASE hallmonitor TO hallmonitor;
```

2. (Optional) Enable TimescaleDB extension:
```sql
\c hallmonitor
CREATE EXTENSION IF NOT EXISTS timescaledb;
```

3. HallMonitor will automatically create required tables on startup

**TimescaleDB Benefits**:
- Automatic time-series partitioning
- Compression for older data
- Better query performance for large datasets
- Continuous aggregates for pre-computed summaries

### 3. InfluxDB
**Best for**: Time-series focused deployments, high-frequency monitoring, IoT

**Pros**:
- Purpose-built for time-series data
- Excellent compression and query performance
- Built-in retention policies
- Flux query language for complex analytics
- Native downsampling capabilities
- Industry standard for monitoring stacks

**Cons**:
- Requires external InfluxDB server
- Different query language (Flux/InfluxQL)
- More specialized than general-purpose databases

**Configuration**:
```yaml
storage:
  backend: "influxdb"
  influxdb:
    url: "http://localhost:8086"
    token: "${INFLUXDB_TOKEN}"  # Use env var
    org: "hallmonitor"
    bucket: "monitor_results"
    retentionDays: 30
```

**Setup Steps**:

1. Install InfluxDB 2.x

2. Create organization and bucket:
```bash
influx setup \
  --username hallmonitor \
  --password your-password \
  --org hallmonitor \
  --bucket monitor_results \
  --retention 30d
```

3. Create API token:
```bash
influx auth create \
  --org hallmonitor \
  --all-access
```

4. Use the generated token in your configuration

### 4. None (Metrics Only)
**Best for**: Prometheus scraping, external storage solutions

**Pros**:
- Minimal resource usage
- No disk space needed
- Perfect for Prometheus integration

**Cons**:
- No historical data in HallMonitor
- API endpoints for history will return empty results
- Dashboard historical views disabled

**Configuration**:
```yaml
storage:
  backend: "none"
```

## Comparison Matrix

| Feature | BadgerDB | PostgreSQL | InfluxDB | None |
|---------|----------|------------|----------|------|
| **Setup Complexity** | ⭐ Easy | ⭐⭐⭐ Moderate | ⭐⭐⭐ Moderate | ⭐ Easy |
| **Horizontal Scaling** | ❌ No | ✅ Yes | ✅ Yes | N/A |
| **SQL Queries** | ❌ No | ✅ Yes | ⚠️ Flux | N/A |
| **Time-series Optimized** | ⚠️ Partial | ⚠️ With TimescaleDB | ✅ Yes | N/A |
| **Resource Usage** | ⭐ Low | ⭐⭐⭐ High | ⭐⭐ Medium | ⭐ Minimal |
| **Disk Space Efficiency** | ⭐⭐ Good | ⭐⭐ Good | ⭐⭐⭐ Excellent | N/A |
| **Query Performance** | ⭐⭐⭐ Fast | ⭐⭐ Good | ⭐⭐⭐ Fast | N/A |
| **Retention Management** | ✅ Automatic | ✅ Automatic | ✅ Built-in | N/A |

## Choosing a Backend

### Use **BadgerDB** if:
- You're running a single HallMonitor instance
- You want zero external dependencies
- Monitoring < 100 services
- You don't need SQL analytics
- Storage < 10GB is acceptable

### Use **PostgreSQL** if:
- You need SQL query capabilities
- You already run PostgreSQL infrastructure
- You want enterprise reliability features
- You need complex reporting and analytics
- You require ACID compliance

### Use **InfluxDB** if:
- You're monitoring > 100 services
- You need high-frequency checks (< 10s intervals)
- You want advanced time-series analytics
- You're already using InfluxDB/TICK stack
- Disk efficiency is critical

### Use **None** if:
- You only need Prometheus metrics
- You're using external storage (e.g., Prometheus)
- Historical data isn't needed in HallMonitor

## Migration Between Backends

### Export from Current Backend
```bash
# Export data (future feature)
hallmonitor export --backend badger --output /tmp/export.json
```

### Import to New Backend
```bash
# Import data (future feature)
hallmonitor import --backend postgres --input /tmp/export.json
```

**Note**: Migration tools are planned for a future release. Currently, switching backends starts with a fresh database.

## Performance Recommendations

### BadgerDB
- Use SSD storage for better performance
- Keep retention period < 90 days
- Monitor disk space usage
- Enable aggregation for faster queries

### PostgreSQL
- Use connection pooling (configured automatically)
- Enable TimescaleDB for > 100 monitors
- Set appropriate retention policies
- Configure regular VACUUM operations
- Use pgBouncer for high-traffic deployments

### InfluxDB
- Use appropriate retention policies
- Configure downsampling for long-term storage
- Monitor shard group duration
- Use batch writes (handled automatically)

## Retention Policies

All backends support automatic data retention:

```yaml
storage:
  backend: "postgres"  # or influxdb, badger
  postgres:
    retentionDays: 90  # Automatically delete data older than 90 days
```

Retention cleanup runs daily at midnight (local time).

## Security Best Practices

### 1. Use Environment Variables for Secrets
```yaml
storage:
  postgres:
    password: "${POSTGRES_PASSWORD}"
```

```bash
export POSTGRES_PASSWORD="your-secure-password"
./hallmonitor
```

### 2. Enable SSL/TLS
```yaml
storage:
  postgres:
    sslmode: "require"  # or verify-ca, verify-full
```

### 3. Restrict Database Access
- Create dedicated database user with minimal permissions
- Use firewall rules to limit access
- Enable authentication on InfluxDB

### 4. Regular Backups
- PostgreSQL: Use pg_dump or continuous archiving
- InfluxDB: Use backup/restore commands
- BadgerDB: Backup the data directory

## Monitoring Storage Health

HallMonitor provides health checks for storage backends:

### API Endpoint
```bash
curl http://localhost:7878/api/v1/storage/health
```

Response:
```json
{
  "healthy": true,
  "backend": "postgres",
  "capabilities": {
    "supportsAggregation": true,
    "supportsRetention": true,
    "supportsRawResults": true,
    "readOnly": false
  }
}
```

### Metrics
Storage backend metrics are exposed via Prometheus:
- `hallmonitor_storage_operations_total` - Total storage operations
- `hallmonitor_storage_errors_total` - Storage operation errors
- `hallmonitor_storage_latency_seconds` - Operation latency

## Troubleshooting

### PostgreSQL Connection Issues
```bash
# Test connection
psql -h localhost -U hallmonitor -d hallmonitor

# Check logs
tail -f /var/log/postgresql/postgresql-15-main.log
```

### InfluxDB Connection Issues
```bash
# Test connection
influx ping

# Check token
influx auth list
```

### BadgerDB Issues
```bash
# Check disk space
df -h

# Verify permissions
ls -la ./data/
```

## Example Deployments

### Docker Compose with PostgreSQL
See: `examples/config-postgres.yml`

### Docker Compose with InfluxDB
See: `examples/config-influxdb.yml`

### Kubernetes with PostgreSQL
```yaml
# See full example in docs/kubernetes/
apiVersion: v1
kind: ConfigMap
metadata:
  name: hallmonitor-config
data:
  config.yml: |
    storage:
      backend: "postgres"
      postgres:
        host: "postgres-service"
        port: 5432
        database: "hallmonitor"
        user: "hallmonitor"
        password: "${POSTGRES_PASSWORD}"
```

## Future Enhancements

Planned storage backend features:
- [ ] SQLite backend (lightweight, embedded with SQL)
- [ ] ClickHouse backend (extreme scale)
- [ ] S3/Object storage for archival
- [ ] Multi-backend support (write to multiple simultaneously)
- [ ] Data export/import tools
- [ ] Automatic migration between backends

## Support

For issues or questions about storage backends:
- GitHub Issues: https://github.com/1broseidon/hallmonitor/issues
- Documentation: https://docs.hallmonitor.io
