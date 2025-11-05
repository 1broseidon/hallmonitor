# Kubernetes Deployment

Hall Monitor can be deployed to Kubernetes using Helm.

## Prerequisites

- Kubernetes cluster (1.19+)
- Helm 3.x
- kubectl configured to access your cluster

## Quick Install

The Helm chart automatically pulls the latest multi-architecture image from GitHub Container Registry (`ghcr.io/1broseidon/hallmonitor:latest`).

```bash
# Install with default values
helm install hallmonitor ./helm/hallmonitor -n hallmonitor --create-namespace

# Install with custom values
helm install hallmonitor ./helm/hallmonitor -n hallmonitor --create-namespace -f custom-values.yaml
```

## Configuration

The Helm chart includes:
- Deployment with configurable replicas
- Service (ClusterIP by default)
- ConfigMap for application configuration
- ServiceAccount with minimal permissions
- Optional: Ingress, HPA (Horizontal Pod Autoscaler)

### Production Values

For production deployments, use the included production values:

```bash
helm install hallmonitor ./helm/hallmonitor -n hallmonitor --create-namespace \
  -f ./helm/hallmonitor/values-production.yaml
```

### Custom Configuration

Create a `custom-values.yaml` with your monitors:

```yaml
config:
  monitoring:
    groups:
      - name: "my-services"
        monitors:
          - type: "http"
            name: "my-app"
            url: "https://app.example.com"
```

Then install:

```bash
helm install hallmonitor ./helm/hallmonitor -n hallmonitor --create-namespace -f custom-values.yaml
```

## Upgrade

```bash
helm upgrade hallmonitor ./helm/hallmonitor -n hallmonitor
```

## Uninstall

```bash
helm uninstall hallmonitor -n hallmonitor
```

## Accessing the Dashboard

```bash
# Port forward to access locally
kubectl port-forward -n hallmonitor svc/hallmonitor 7878:7878

# Then open http://localhost:7878 in your browser
```

## Prometheus Metrics

The `/metrics` endpoint is automatically exposed and can be scraped by Prometheus. If you have the Prometheus Operator installed, a ServiceMonitor can be enabled:

```yaml
serviceMonitor:
  enabled: true
```

## Additional Resources

- [Helm Chart Documentation](./helm/hallmonitor/README.md)
- [Values Reference](./helm/hallmonitor/values.yaml)
- [Production Values](./helm/hallmonitor/values-production.yaml)

