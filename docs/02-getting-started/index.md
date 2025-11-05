# Getting Started

Get Hall Monitor running in **under 2 minutes** - no git clone required!

## Quick Start (2 Minutes)

**1. Create a minimal config file:**

```bash
cat > config.yml << 'EOF'
server:
  port: "7878"
  host: "0.0.0.0"
  enableDashboard: true

monitoring:
  defaultInterval: "30s"
  defaultTimeout: "10s"
  groups:
    - name: "my-services"
      monitors:
        - type: "http"
          name: "example"
          url: "https://example.com"
          expectedStatus: 200
EOF
```

**2. Run with Docker:**

```bash
docker run -d \
  --name hallmonitor \
  --network host \
  --cap-add NET_RAW \
  --cap-add NET_ADMIN \
  -v $(pwd)/config.yml:/etc/hallmonitor/config.yml:ro \
  ghcr.io/1broseidon/hallmonitor:latest
```

**3. Access the dashboard:**

Open http://localhost:7878 in your browser.

**Done!** Edit `config.yml` to add your own monitors, then restart the container.

---

## Advanced Installation Methods

For production deployments, Docker Compose, Kubernetes, or full observability stacks, see:
- [Installation Guide](./installation.md) - All installation methods
- [Configuration Basics](./configuration-basics.md) - Detailed configuration


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
| Dashboard | Web UI | http://localhost:7878 |
| Health Check | Service health | http://localhost:7878/health |
| Metrics | Prometheus metrics | http://localhost:7878/metrics |
| API | Monitor status | http://localhost:7878/api/v1/monitors |

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
kubectl port-forward svc/hallmonitor 7878:7878 -n hallmonitor  # Access locally

# Binary
./hallmonitor --config config.yml  # Start
./hallmonitor --help               # Show help
```

## Next Steps

- [Installation Guide](./installation.md) - Detailed installation instructions
- [Configuration Basics](./configuration-basics.md) - Learn configuration fundamentals
- [First Monitor](./first-monitor.md) - Set up your first monitor step-by-step
