# Use Cases

Hall Monitor is designed for a variety of monitoring scenarios. This guide showcases common use cases and configuration examples.

## Home Lab Monitoring

Monitor your self-hosted infrastructure and services with minimal resource overhead.

### Basic Home Lab Setup

```yaml
monitoring:
  defaultInterval: "30s"
  defaultTimeout: "10s"

  groups:
    - name: "network-infrastructure"
      interval: "15s"
      monitors:
        # Router connectivity
        - type: "ping"
          name: "router"
          target: "192.168.1.1"
          count: 3

        # DNS server
        - type: "dns"
          name: "pihole-dns"
          target: "192.168.1.2:53"
          query: "google.com"
          queryType: "A"

    - name: "home-servers"
      interval: "30s"
      monitors:
        # NAS web interface
        - type: "http"
          name: "nas-web"
          url: "http://192.168.1.10:5000"
          expectedStatus: 200

        # SSH access to server
        - type: "tcp"
          name: "server-ssh"
          target: "192.168.1.20:22"

    - name: "media-services"
      interval: "60s"
      monitors:
        # Plex Media Server
        - type: "http"
          name: "plex"
          url: "http://192.168.1.10:32400/web"
          expectedStatus: 200

        # Sonarr
        - type: "http"
          name: "sonarr"
          url: "http://192.168.1.10:8989"
          expectedStatus: 200

        # Radarr
        - type: "http"
          name: "radarr"
          url: "http://192.168.1.10:7878"
          expectedStatus: 200
```

### Home Automation Monitoring

```yaml
monitoring:
  groups:
    - name: "home-automation"
      interval: "30s"
      monitors:
        # Home Assistant
        - type: "http"
          name: "home-assistant"
          url: "http://192.168.1.30:8123"
          expectedStatus: 200

        # MQTT Broker
        - type: "tcp"
          name: "mosquitto"
          target: "192.168.1.30:1883"

        # Zigbee2MQTT
        - type: "http"
          name: "zigbee2mqtt"
          url: "http://192.168.1.30:8080"
          expectedStatus: 200
```

## Kubernetes Cluster Monitoring

Monitor cluster components and workloads running in Kubernetes.

### Kubernetes Internal Services

```yaml
monitoring:
  groups:
    - name: "k8s-control-plane"
      interval: "15s"
      monitors:
        # Kubernetes API server
        - type: "https"
          name: "k8s-api"
          url: "https://kubernetes.default.svc.cluster.local:443/healthz"
          expectedStatus: 200

        # CoreDNS
        - type: "dns"
          name: "coredns"
          target: "10.96.0.10:53"
          query: "kubernetes.default.svc.cluster.local"
          queryType: "A"

    - name: "k8s-workloads"
      interval: "30s"
      monitors:
        # Application service
        - type: "http"
          name: "webapp"
          url: "http://webapp.default.svc.cluster.local:8080/health"
          expectedStatus: 200

        # Database service
        - type: "tcp"
          name: "postgres"
          target: "postgres.default.svc.cluster.local:5432"

        # Redis cache
        - type: "tcp"
          name: "redis"
          target: "redis.default.svc.cluster.local:6379"
```

### Ingress and External Access

```yaml
monitoring:
  groups:
    - name: "ingress-endpoints"
      interval: "30s"
      monitors:
        # Ingress controller
        - type: "http"
          name: "ingress-nginx"
          url: "http://ingress-nginx-controller.ingress-nginx.svc.cluster.local"
          expectedStatus: 404  # 404 is expected for root path

        # External application
        - type: "http"
          name: "external-app"
          url: "https://myapp.example.com"
          expectedStatus: 200
```

## Microservices Monitoring

Monitor distributed microservices architecture.

### Service Mesh

```yaml
monitoring:
  groups:
    - name: "api-gateway"
      interval: "10s"
      monitors:
        - type: "http"
          name: "gateway-health"
          url: "http://gateway:8080/health"
          expectedStatus: 200
          labels:
            tier: "gateway"
            criticality: "high"

    - name: "backend-services"
      interval: "15s"
      monitors:
        # User service
        - type: "http"
          name: "user-service"
          url: "http://user-service:8080/health"
          expectedStatus: 200
          labels:
            tier: "backend"
            domain: "identity"

        # Order service
        - type: "http"
          name: "order-service"
          url: "http://order-service:8080/health"
          expectedStatus: 200
          labels:
            tier: "backend"
            domain: "commerce"

        # Payment service
        - type: "http"
          name: "payment-service"
          url: "http://payment-service:8080/health"
          expectedStatus: 200
          labels:
            tier: "backend"
            domain: "commerce"
            criticality: "critical"

    - name: "data-layer"
      interval: "30s"
      monitors:
        # PostgreSQL
        - type: "tcp"
          name: "postgres"
          target: "postgres:5432"
          labels:
            tier: "data"

        # Redis
        - type: "tcp"
          name: "redis"
          target: "redis:6379"
          labels:
            tier: "cache"

        # Elasticsearch
        - type: "http"
          name: "elasticsearch"
          url: "http://elasticsearch:9200/_cluster/health"
          expectedStatus: 200
          labels:
            tier: "search"
```

## Multi-Region Monitoring

Monitor services across different regions and data centers.

### Global Service Monitoring

```yaml
monitoring:
  groups:
    - name: "us-east-region"
      interval: "30s"
      monitors:
        - type: "http"
          name: "api-us-east"
          url: "https://api-us-east.example.com/health"
          expectedStatus: 200
          labels:
            region: "us-east-1"

        - type: "http"
          name: "web-us-east"
          url: "https://us-east.example.com"
          expectedStatus: 200
          labels:
            region: "us-east-1"

    - name: "eu-west-region"
      interval: "30s"
      monitors:
        - type: "http"
          name: "api-eu-west"
          url: "https://api-eu-west.example.com/health"
          expectedStatus: 200
          labels:
            region: "eu-west-1"

        - type: "http"
          name: "web-eu-west"
          url: "https://eu-west.example.com"
          expectedStatus: 200
          labels:
            region: "eu-west-1"

    - name: "asia-pacific-region"
      interval: "30s"
      monitors:
        - type: "http"
          name: "api-ap-south"
          url: "https://api-ap-south.example.com/health"
          expectedStatus: 200
          labels:
            region: "ap-south-1"
```

## SSL Certificate Monitoring

Track SSL certificate expiration for web services.

### Certificate Monitoring

```yaml
monitoring:
  defaultSSLCertExpiryWarningDays: 30  # Warn 30 days before expiry

  groups:
    - name: "ssl-certificates"
      interval: "3600s"  # Check once per hour
      monitors:
        - type: "http"
          name: "main-website-ssl"
          url: "https://www.example.com"
          expectedStatus: 200
          sslCertExpiryWarningDays: 30
          labels:
            purpose: "ssl-monitoring"

        - type: "http"
          name: "api-ssl"
          url: "https://api.example.com"
          expectedStatus: 200
          sslCertExpiryWarningDays: 14  # More urgent for API
          labels:
            purpose: "ssl-monitoring"

        - type: "http"
          name: "admin-portal-ssl"
          url: "https://admin.example.com"
          expectedStatus: 200
          sslCertExpiryWarningDays: 7
          labels:
            purpose: "ssl-monitoring"
```

## External Service Monitoring

Monitor third-party services and APIs.

### SaaS and External APIs

```yaml
monitoring:
  groups:
    - name: "external-services"
      interval: "60s"
      monitors:
        # Payment gateway
        - type: "http"
          name: "stripe-api"
          url: "https://api.stripe.com"
          expectedStatus: 401  # Expect auth error without token
          timeout: "10s"
          labels:
            vendor: "stripe"

        # Email service
        - type: "tcp"
          name: "smtp-server"
          target: "smtp.sendgrid.net:587"
          timeout: "10s"
          labels:
            vendor: "sendgrid"

        # CDN
        - type: "http"
          name: "cloudflare-cdn"
          url: "https://mysite.cdn.cloudflare.net/health"
          expectedStatus: 200
          timeout: "10s"
          labels:
            vendor: "cloudflare"

    - name: "cloud-services"
      interval: "60s"
      monitors:
        # AWS S3 bucket
        - type: "http"
          name: "s3-bucket"
          url: "https://mybucket.s3.amazonaws.com"
          expectedStatus: 403  # Expect forbidden without auth
          timeout: "10s"

        # GitHub
        - type: "http"
          name: "github-api"
          url: "https://api.github.com"
          expectedStatus: 200
          timeout: "10s"
```

## Database and Data Store Monitoring

Monitor various database systems.

### Database Connectivity

```yaml
monitoring:
  groups:
    - name: "databases"
      interval: "30s"
      monitors:
        # PostgreSQL
        - type: "tcp"
          name: "postgres-primary"
          target: "postgres-primary.example.com:5432"
          timeout: "5s"
          labels:
            database: "postgresql"
            role: "primary"

        - type: "tcp"
          name: "postgres-replica"
          target: "postgres-replica.example.com:5432"
          timeout: "5s"
          labels:
            database: "postgresql"
            role: "replica"

        # MySQL
        - type: "tcp"
          name: "mysql-db"
          target: "mysql.example.com:3306"
          timeout: "5s"
          labels:
            database: "mysql"

        # MongoDB
        - type: "tcp"
          name: "mongodb"
          target: "mongodb.example.com:27017"
          timeout: "5s"
          labels:
            database: "mongodb"

        # Redis
        - type: "tcp"
          name: "redis-cache"
          target: "redis.example.com:6379"
          timeout: "3s"
          labels:
            database: "redis"
            purpose: "cache"

        # Elasticsearch
        - type: "http"
          name: "elasticsearch"
          url: "http://elasticsearch.example.com:9200/_cluster/health"
          expectedStatus: 200
          timeout: "5s"
          labels:
            database: "elasticsearch"
```

## DNS Infrastructure Monitoring

Monitor DNS servers and record propagation.

### DNS Server Monitoring

```yaml
monitoring:
  groups:
    - name: "dns-servers"
      interval: "30s"
      monitors:
        # Primary DNS (Google)
        - type: "dns"
          name: "google-dns-primary"
          target: "8.8.8.8:53"
          query: "google.com"
          queryType: "A"
          timeout: "3s"

        # Secondary DNS (Cloudflare)
        - type: "dns"
          name: "cloudflare-dns"
          target: "1.1.1.1:53"
          query: "cloudflare.com"
          queryType: "A"
          timeout: "3s"

        # Internal DNS
        - type: "dns"
          name: "internal-dns"
          target: "192.168.1.1:53"
          query: "example.local"
          queryType: "A"
          expectedResponse: "192.168.1.100"
          timeout: "3s"

    - name: "dns-records"
      interval: "300s"  # Check every 5 minutes
      monitors:
        # A records
        - type: "dns"
          name: "domain-a-record"
          target: "8.8.8.8:53"
          query: "www.example.com"
          queryType: "A"

        # MX records
        - type: "dns"
          name: "domain-mx-record"
          target: "8.8.8.8:53"
          query: "example.com"
          queryType: "MX"

        # TXT records (SPF)
        - type: "dns"
          name: "domain-spf-record"
          target: "8.8.8.8:53"
          query: "example.com"
          queryType: "TXT"
```

## Network Infrastructure Monitoring

Monitor routers, switches, and network devices.

### Network Device Monitoring

```yaml
monitoring:
  groups:
    - name: "network-core"
      interval: "15s"
      monitors:
        # Main router
        - type: "ping"
          name: "core-router"
          target: "192.168.1.1"
          count: 3
          timeout: "3s"
          labels:
            device: "router"
            location: "datacenter"

        # Layer 3 switch
        - type: "ping"
          name: "core-switch"
          target: "192.168.1.2"
          count: 3
          timeout: "3s"
          labels:
            device: "switch"
            location: "datacenter"

    - name: "network-edge"
      interval: "30s"
      monitors:
        # Access points
        - type: "ping"
          name: "ap-floor-1"
          target: "192.168.1.10"
          count: 3
          labels:
            device: "access-point"
            location: "floor-1"

        - type: "ping"
          name: "ap-floor-2"
          target: "192.168.1.11"
          count: 3
          labels:
            device: "access-point"
            location: "floor-2"

        # Firewall
        - type: "ping"
          name: "perimeter-firewall"
          target: "192.168.1.254"
          count: 3
          labels:
            device: "firewall"
            location: "dmz"
```

## Best Practices

### Organizing Monitors

1. **Group by function**: Network, applications, databases, external services
2. **Group by criticality**: Critical, important, non-critical
3. **Group by location**: Region, data center, availability zone
4. **Use consistent naming**: `service-type-location` pattern

### Setting Intervals

- **Critical services**: 10-15 seconds
- **Important services**: 30 seconds
- **Regular services**: 60 seconds
- **Background checks**: 300-600 seconds
- **SSL certificates**: 3600 seconds (1 hour)

### Using Labels Effectively

```yaml
labels:
  environment: "production"
  team: "platform"
  criticality: "high"
  region: "us-east-1"
  alert: "pagerduty"
```

Labels help with:
- Filtering in dashboards
- Routing alerts
- Organizing metrics
- Team ownership

## Next Steps

- [Getting Started](../02-getting-started/index.md) - Install Hall Monitor
- [Configuration Basics](../02-getting-started/configuration-basics.md) - Detailed configuration
- [Monitor Types](../03-monitors/index.md) - Learn about each monitor type
