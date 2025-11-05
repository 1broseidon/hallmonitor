# Introduction to Hall Monitor

Hall Monitor is a lightweight, cloud-native network monitoring solution designed for home labs, Kubernetes clusters, and production environments. Built in Go, it provides comprehensive monitoring capabilities with minimal resource overhead.

## What is Hall Monitor?

Hall Monitor is a modern monitoring tool that checks the health and availability of your infrastructure components. It supports multiple monitoring protocols, exports metrics to Prometheus, and includes a built-in dashboard for quick visibility.

### Key Features

- **Multiple Monitor Types**: HTTP/HTTPS, TCP, DNS, and Ping monitoring
- **Cloud Native**: Designed for containers and Kubernetes with minimal footprint
- **Built-in Dashboard**: Lightweight web UI with dark mode support
- **Prometheus Integration**: Full metrics export for Grafana visualization
- **Flexible Configuration**: YAML-based with environment variable substitution
- **Lightweight**: ~12MB binary, minimal CPU and memory usage
- **Multi-Architecture**: Supports AMD64 and ARM64 platforms

## Why Hall Monitor?

### For Home Lab Enthusiasts

- Monitor your self-hosted services and infrastructure
- Track uptime of NAS, media servers, and home automation
- Lightweight enough to run on Raspberry Pi
- No complex dependencies or databases required

### For Kubernetes Users

- Native Kubernetes support with Helm charts
- ServiceMonitor for Prometheus Operator integration
- Horizontal auto-scaling with HPA
- High availability deployment options

### For DevOps Teams

- Simple yet powerful monitoring for microservices
- Easy integration with existing Prometheus/Grafana stack
- Webhook notifications for Discord and Slack
- Production-ready with security best practices

## Architecture Overview

Hall Monitor follows a simple architecture:

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
          │                    │
          ▼                    ▼
    Web Browser           Prometheus
```

### Core Components

1. **Scheduler**: Manages monitor execution using worker pools
2. **Monitors**: Execute checks for different protocols (HTTP, TCP, DNS, Ping)
3. **Result Store**: In-memory storage of recent check results
4. **Metrics**: Prometheus metrics exporter
5. **Dashboard API**: REST API and web dashboard
6. **Webhooks**: Event notifications to external services

## Performance Characteristics

- **Memory**: ~30-50MB for typical workloads
- **CPU**: Minimal (< 5% on single core)
- **Latency**: Sub-millisecond internal overhead
- **Scalability**: Handles 1000+ monitors on modest hardware
- **Startup**: < 2 seconds cold start

## Comparison with Alternatives

| Feature | Hall Monitor | Prometheus Blackbox | Uptime Kuma | Pingdom |
|---------|--------------|---------------------|-------------|---------|
| Open Source | Yes | Yes | Yes | No |
| Self-Hosted | Yes | Yes | Yes | No |
| Container Native | Yes | Yes | Yes | N/A |
| Built-in Dashboard | Yes | No | Yes | Yes |
| Prometheus Metrics | Yes | Yes | No | No |
| Multi-Architecture | Yes | Yes | Yes | N/A |
| Resource Usage | Low | Low | Medium | N/A |
| Configuration | YAML | YAML | Web UI | Web UI |

## Getting Started

Ready to start monitoring? Check out our [Getting Started Guide](../02-getting-started/index.md) for installation instructions and quick setup.

## Documentation Structure

This documentation is organized into the following sections:

1. **Introduction** - Overview and concepts (you are here)
2. **Getting Started** - Quick installation and first monitor
3. **Installation** - Detailed installation methods
4. **Configuration** - Complete configuration reference
5. **Monitors** - Monitor types and examples
6. **Observability** - Metrics, dashboards, and alerting
7. **Deployment** - Production deployment strategies
8. **API Reference** - REST API documentation
9. **Development** - Contributing and development guide
10. **Reference** - CLI, troubleshooting, and FAQ

## Next Steps

- [Core Concepts](./concepts.md) - Understand monitors, groups, and metrics
- [Use Cases](./use-cases.md) - Common monitoring scenarios
- [Getting Started](../02-getting-started/index.md) - Install and configure Hall Monitor
