# Getting Started

Get Hall Monitor up and running in minutes. This guide walks you through installation, basic configuration, and your first monitor.

## Quick Start Paths

Choose the installation method that fits your environment:

### Docker Compose (Recommended for Testing)
Perfect for quick evaluation and single-server deployments.
- **Time**: 2 minutes
- **Complexity**: Beginner
- **Prerequisites**: Docker and Docker Compose

[Start with Docker Compose](./installation.md#docker-compose)

### Kubernetes
Ideal for production clusters and cloud-native environments.
- **Time**: 5 minutes
- **Complexity**: Intermediate
- **Prerequisites**: Kubernetes cluster, kubectl

[Start with Kubernetes](./installation.md#kubernetes)

### Binary
Direct installation on your server without containers.
- **Time**: 3 minutes
- **Complexity**: Beginner
- **Prerequisites**: Linux/macOS server

[Start with Binary](./installation.md#binary-installation)

## Installation Overview

### 1. Install Hall Monitor

Choose your preferred installation method from the options above. Each method includes:
- Step-by-step installation instructions
- Configuration file setup
- Service startup commands
- Verification steps

### 2. Create Configuration

Hall Monitor uses YAML configuration files. Start with the example configuration:

```bash
# Copy the example configuration
cp config.example.yml config.yml

# Edit with your monitors
nano config.yml
```

See [Configuration Basics](./configuration-basics.md) for detailed configuration guidance.

### 3. Add Your First Monitor

Add a simple HTTP monitor to test your setup:

```yaml
monitoring:
  defaultInterval: "30s"
  defaultTimeout: "10s"

  groups:
    - name: "test-monitors"
      monitors:
        - type: "http"
          name: "example-website"
          url: "https://example.com"
          expectedStatus: 200
```

See [First Monitor](./first-monitor.md) for a complete walkthrough.

### 4. Start Monitoring

Start Hall Monitor and verify it's working:

```bash
# Access the dashboard
open http://localhost:8080

# Check health endpoint
curl http://localhost:8080/health

# View metrics
curl http://localhost:8080/metrics
```

## What's Next?

Once you have Hall Monitor running:

1. **Add More Monitors**: Configure monitoring for your infrastructure
   - [Monitor Types](../03-monitors/index.md)
   - [Configuration Basics](./configuration-basics.md)

2. **Set Up Observability**: Integrate with Prometheus and Grafana
   - [Observability Overview](../04-observability/index.md)

3. **Production Deployment**: Deploy for production use
   - Use Helm with production values for Kubernetes deployments
   - Follow Docker best practices for container deployments

## Need Help?

- **Installation Issues**: [Installation Guide](./installation.md)
- **Configuration Questions**: [Configuration Basics](./configuration-basics.md)
- **Troubleshooting**: [Troubleshooting Guide](../05-reference/troubleshooting.md)
- **Examples**: [Use Cases](../01-introduction/use-cases.md)

## Quick Reference

### Access Points

| Endpoint | Purpose | URL |
|----------|---------|-----|
| Dashboard | Web UI | http://localhost:8080 |
| Health Check | Service health | http://localhost:8080/health |
| Metrics | Prometheus metrics | http://localhost:8080/metrics |
| API | Monitor status | http://localhost:8080/api/v1/monitors |

### Common Commands

```bash
# Docker Compose
docker compose up -d          # Start
docker compose logs -f        # View logs
docker compose restart        # Restart
docker compose down           # Stop

# Kubernetes
kubectl get pods -n hallmonitor                    # Check status
kubectl logs -f deployment/hallmonitor -n hallmonitor  # View logs
kubectl port-forward svc/hallmonitor 8080:8080 -n hallmonitor  # Access locally

# Binary
./hallmonitor --config config.yml  # Start
./hallmonitor --help               # Show help
```

## Next Steps

- [Installation Guide](./installation.md) - Detailed installation instructions
- [Configuration Basics](./configuration-basics.md) - Learn configuration fundamentals
- [First Monitor](./first-monitor.md) - Set up your first monitor step-by-step
