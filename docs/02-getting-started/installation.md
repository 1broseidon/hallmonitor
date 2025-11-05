# Installation

This guide covers all installation methods for Hall Monitor. Choose the method that best fits your environment.

## Docker Compose

The easiest way to get started with Hall Monitor.

### Prerequisites

- Docker 20.10 or higher
- Docker Compose 2.0 or higher
- 512MB RAM minimum
- 1GB disk space

### Simple Installation

Deploy Hall Monitor without additional services:

```bash
# 1. Clone or download the repository
git clone https://github.com/1broseidon/hallmonitor.git
cd hallmonitor

# 2. Copy environment template
cp .env.example .env

# 3. Create your config from the example
cp config.example.yml config.yml

# 4. Edit configuration with your monitors
nano config.yml

# 5. Start Hall Monitor (mounts config.yml into container)
docker compose up -d

# 6. Verify it's running
curl http://localhost:8080/health
```

Hall Monitor is now available at http://localhost:8080

### Full Stack Installation

Deploy Hall Monitor with Prometheus, Grafana, and Loki for complete observability:

```bash
# Start Hall Monitor
docker compose up -d

# Access the dashboard
# Hall Monitor: http://localhost:8080
# Grafana:      http://localhost:3000 (admin/hallmonitor)
# Prometheus:   http://localhost:19090
```

### Configuration

Edit the `.env` file to customize:

```bash
# Server configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Grafana credentials
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=hallmonitor
```

## Kubernetes

Deploy Hall Monitor to your Kubernetes cluster.

### Prerequisites

- Kubernetes 1.20 or higher (or K3s)
- kubectl configured
- 256MB RAM per pod minimum
- Cluster admin access for RBAC

### Quick Deployment

```bash
# Clone the repository
git clone https://github.com/1broseidon/hallmonitor.git
cd hallmonitor

# Deploy to Kubernetes using Helm
helm install hallmonitor ./k8s/helm/hallmonitor -n hallmonitor --create-namespace

# Check status
kubectl get pods -n hallmonitor

# Access via port-forward
kubectl port-forward svc/hallmonitor 8080:8080 -n hallmonitor
```

### Environment-Specific Deployment

Hall Monitor provides three deployment configurations:

**Development**:
```bash
helm install hallmonitor ./k8s/helm/hallmonitor -n hallmonitor --create-namespace
```
- Single replica
- Default settings

**Production**:
```bash
helm install hallmonitor ./k8s/helm/hallmonitor -n hallmonitor --create-namespace \
  -f ./k8s/helm/hallmonitor/values-production.yaml
```
- 2-5 replicas with HPA
- JSON logging
- Ingress with TLS
- Pod anti-affinity for HA
- Optimized resource limits

### Accessing the Service

**Inside the cluster**:
```
http://hallmonitor.hallmonitor.svc.cluster.local:8080
```

**External access with port-forward**:
```bash
kubectl port-forward svc/hallmonitor 8080:8080 -n hallmonitor
```

**Production with Ingress** (after deployment):
```
https://hallmonitor.yourdomain.com
```

### Customizing Configuration

The configuration is stored in a ConfigMap. To update:

```bash
# Edit the ConfigMap
kubectl edit configmap hallmonitor-config -n hallmonitor

# Restart pods to reload
kubectl rollout restart deployment/hallmonitor -n hallmonitor
```

## Helm

Use Helm for managed Kubernetes deployments.

### Prerequisites

- Helm 3.0 or higher
- Kubernetes cluster
- kubectl configured

### Installation

```bash
# Clone the repository
git clone https://github.com/1broseidon/hallmonitor.git
cd hallmonitor

# Install with default values
helm install hallmonitor k8s/helm/hallmonitor -n hallmonitor --create-namespace

# Check status
helm status hallmonitor -n hallmonitor
```

### Custom Values

Create a custom values file:

```yaml
# custom-values.yaml
replicaCount: 3

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi

ingress:
  enabled: true
  hosts:
    - host: hallmonitor.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: hallmonitor-tls
      hosts:
        - hallmonitor.example.com

config:
  server:
    port: "8080"
  monitoring:
    defaultInterval: "30s"
    groups:
      - name: "my-services"
        monitors:
          - type: "http"
            name: "my-app"
            url: "https://myapp.example.com"
            expectedStatus: 200
```

Install with custom values:

```bash
helm install hallmonitor k8s/helm/hallmonitor \
  -n hallmonitor \
  --create-namespace \
  -f custom-values.yaml
```

### Production Installation

Use production-ready values:

```bash
helm install hallmonitor k8s/helm/hallmonitor \
  -n hallmonitor \
  --create-namespace \
  -f k8s/helm/hallmonitor/values-production.yaml
```

### Upgrading

```bash
# Update values
helm upgrade hallmonitor k8s/helm/hallmonitor \
  -n hallmonitor \
  -f custom-values.yaml

# Rollback if needed
helm rollback hallmonitor -n hallmonitor
```

## Binary Installation

Install Hall Monitor directly on your server.

### Prerequisites

- Linux (AMD64 or ARM64) or macOS
- 512MB RAM minimum
- Go 1.21+ (for building from source)

### Download Pre-built Binary

```bash
# Download the latest release
wget https://github.com/1broseidon/hallmonitor/releases/latest/download/hallmonitor-linux-amd64

# Make executable
chmod +x hallmonitor-linux-amd64
mv hallmonitor-linux-amd64 /usr/local/bin/hallmonitor

# Verify installation
hallmonitor --version
```

### Build from Source

```bash
# Clone repository
git clone https://github.com/1broseidon/hallmonitor.git
cd hallmonitor

# Build
make build

# Install
sudo cp hallmonitor /usr/local/bin/

# Verify
hallmonitor --version
```

### Configuration

```bash
# Copy example configuration
cp config.example.yml /etc/hallmonitor/config.yml

# Edit configuration
sudo nano /etc/hallmonitor/config.yml

# Run Hall Monitor
hallmonitor --config /etc/hallmonitor/config.yml
```

### Run as Systemd Service

Create a systemd service file:

```bash
sudo nano /etc/systemd/system/hallmonitor.service
```

```ini
[Unit]
Description=Hall Monitor - Network Monitoring
After=network.target

[Service]
Type=simple
User=hallmonitor
Group=hallmonitor
ExecStart=/usr/local/bin/hallmonitor --config /etc/hallmonitor/config.yml
Restart=always
RestartSec=10

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/hallmonitor

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
# Create user
sudo useradd -r -s /bin/false hallmonitor

# Set permissions
sudo chown -R hallmonitor:hallmonitor /etc/hallmonitor

# Enable service
sudo systemctl daemon-reload
sudo systemctl enable hallmonitor
sudo systemctl start hallmonitor

# Check status
sudo systemctl status hallmonitor

# View logs
sudo journalctl -u hallmonitor -f
```

## Verification

After installation, verify Hall Monitor is working:

### Health Check

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2025-01-04T10:30:00Z"
}
```

### Monitor Status

```bash
curl http://localhost:8080/api/v1/monitors | jq
```

### Metrics

```bash
curl http://localhost:8080/metrics | grep hallmonitor
```

### Dashboard

Open your browser:
```
http://localhost:8080
```

## Next Steps

- [Configuration Basics](./configuration-basics.md) - Learn how to configure monitors
- [First Monitor](./first-monitor.md) - Set up your first monitor
- [Monitor Types](../03-monitors/index.md) - Explore different monitor types

## Troubleshooting

### Port Already in Use

```bash
# Change the port in configuration
# Docker: Edit .env file
SERVER_PORT=8081

# Binary: Edit config.yml
server:
  port: "8081"
```

### Permission Denied (Ping)

ICMP ping requires elevated privileges. Hall Monitor automatically falls back to unprivileged mode (UDP), but you can grant capabilities:

```bash
# Grant NET_RAW capability to binary
sudo setcap cap_net_raw+ep /usr/local/bin/hallmonitor

# For Docker, run with privilege
docker run --cap-add NET_RAW --cap-add NET_ADMIN --network host \
  -v $(pwd)/config.yml:/etc/hallmonitor/config.yml:ro \
  ghcr.io/1broseidon/hallmonitor:latest
```

### Configuration Not Loading

```bash
# Docker: Force recreate
docker compose up -d --force-recreate

# Kubernetes: Restart pods
kubectl rollout restart deployment/hallmonitor -n hallmonitor

# Binary: Check file path
hallmonitor --config /path/to/config.yml
```

### Memory Issues

If Hall Monitor is using too much memory:

```bash
# Reduce check intervals
monitoring:
  defaultInterval: "60s"  # Increase from 30s

# Reduce monitor count
# Comment out non-critical monitors

# For Kubernetes, increase memory limits
resources:
  limits:
    memory: 256Mi
```

## Support

For additional help:
- [Troubleshooting Guide](../05-reference/troubleshooting.md)
- [GitHub Issues](https://github.com/1broseidon/hallmonitor/issues)
- [Configuration Basics](./configuration-basics.md)
