# Hall Monitor Documentation

Complete documentation for Hall Monitor - a lightweight network monitoring solution for home labs and Kubernetes clusters.

## Quick Links

- **New to Hall Monitor?** Start with [Getting Started](./02-getting-started/index.md)
- **Need to install?** See [Installation Guide](./02-getting-started/installation.md)
- **Looking for examples?** Check [Use Cases](./01-introduction/use-cases.md)
- **Having issues?** Visit [Troubleshooting](./05-reference/troubleshooting.md)

## Documentation Structure

### 1. Introduction
- [Overview](./01-introduction/index.md) - What is Hall Monitor
- [Core Concepts](./01-introduction/concepts.md) - Monitors, groups, and metrics
- [Use Cases](./01-introduction/use-cases.md) - Real-world scenarios

### 2. Getting Started
- [Quick Start](./02-getting-started/index.md) - Get running in minutes
- [Installation](./02-getting-started/installation.md) - All installation methods
- [Configuration Basics](./02-getting-started/configuration-basics.md) - Configuration fundamentals
- [First Monitor](./02-getting-started/first-monitor.md) - Step-by-step tutorial

### 3. Monitor Types
- [Monitor Overview](./03-monitors/index.md) - All available monitor types

### 4. Observability
- [Observability Overview](./04-observability/index.md) - Metrics, dashboards, and integrations

### 5. Reference
- [Troubleshooting](./05-reference/troubleshooting.md) - Common issues and solutions

## Common Tasks

### Installation

**Docker Compose** (Easiest):
```bash
docker compose up -d
```

**Kubernetes (Helm)**:
```bash
helm install hallmonitor ./k8s/helm/hallmonitor -n hallmonitor --create-namespace
```

**Binary**:
```bash
./hallmonitor --config config.yml
```

See [Installation Guide](./02-getting-started/installation.md) for details.

### Configuration

**Basic Configuration**:
```yaml
server:
  port: "7878"
  enableDashboard: true

monitoring:
  defaultInterval: "30s"
  groups:
    - name: "my-services"
      monitors:
        - type: "http"
          name: "web-app"
          url: "https://example.com"
```

See [Configuration Basics](./02-getting-started/configuration-basics.md) for details.

### Adding Monitors

**HTTP Monitor**:
```yaml
- type: "http"
  name: "api-server"
  url: "https://api.example.com"
  expectedStatus: 200
```

**TCP Monitor**:
```yaml
- type: "tcp"
  name: "database"
  target: "db.example.com:5432"
```

**DNS Monitor**:
```yaml
- type: "dns"
  name: "dns-server"
  target: "8.8.8.8:53"
  query: "example.com"
  queryType: "A"
```

**Ping Monitor**:
```yaml
- type: "ping"
  name: "gateway"
  target: "192.168.1.1"
```

See [Monitor Types](./03-monitors/index.md) for details.

## Key Features

- **Multiple Monitor Types**: HTTP, TCP, DNS, Ping
- **Built-in Dashboard**: Web UI with dark mode
- **Prometheus Integration**: Full metrics export
- **Kubernetes Native**: Helm charts and manifests
- **Lightweight**: ~12MB binary, minimal resources
- **Cloud Native**: Docker, Kubernetes, multi-arch support

## Architecture

```
┌─────────────────────────────────────────┐
│          Hall Monitor Server            │
│                                          │
│  ┌────────────┐      ┌──────────────┐  │
│  │  Scheduler │─────▶│   Monitors   │  │
│  │  (Worker   │      │  (HTTP/TCP/  │  │
│  │   Pools)   │      │   DNS/Ping)  │  │
│  └────────────┘      └──────────────┘  │
│         │                    │          │
│         ▼                    ▼          │
│  ┌────────────┐      ┌──────────────┐  │
│  │   Result   │      │   Metrics    │  │
│  │   Store    │      │  (Prometheus)│  │
│  └────────────┘      └──────────────┘  │
│         │                    │          │
│         ▼                    ▼          │
│  ┌────────────┐      ┌──────────────┐  │
│  │ Dashboard  │      │   /metrics   │  │
│  │    API     │      │   Endpoint   │  │
│  └────────────┘      └──────────────┘  │
└─────────────────────────────────────────┘
```

## Support

- **GitHub**: https://github.com/1broseidon/hallmonitor
- **Issues**: https://github.com/1broseidon/hallmonitor/issues
- **Documentation**: You're reading it!

## Contributing

Contributions are welcome! See [Contributing Guide](./09-development/contributing.md).

## License

MIT License - See [LICENSE](../LICENSE) file for details.

---

**Navigation**: [Getting Started](./02-getting-started/index.md) | [Installation](./02-getting-started/installation.md) | [Monitors](./03-monitors/index.md) | [Troubleshooting](./05-reference/troubleshooting.md)
