# Troubleshooting

Common issues and solutions for Hall Monitor.

## Installation Issues

### Port Already in Use

**Symptom**: `bind: address already in use`

**Solution**:
```bash
# Find what's using the port
lsof -i :7878

# Change Hall Monitor port
# Edit config.yml or .env
server:
  port: "8081"

# Or use environment variable
export SERVER_PORT=8081
```

### Permission Denied (Docker)

**Symptom**: Permission errors accessing config files

**Solution**:
```bash
# Fix file permissions
chmod 644 config.yml

# Or run Docker as current user
docker run --user $(id -u):$(id -g) \
  -v $(pwd)/config.yml:/etc/hallmonitor/config.yml:ro \
  ghcr.io/1broseidon/hallmonitor:latest
```

### Image Pull Errors (Kubernetes)

**Symptom**: `ImagePullBackOff` or `ErrImagePull`

**Solution**:
```bash
# For local clusters (K3s, kind, minikube)
# Build and load image locally
make docker-build
docker save hallmonitor:latest | sudo k3s ctr images import -

# Or push to a registry
make docker-push REGISTRY=your-registry.com
# Update deployment to use your registry
```

## Configuration Issues

### Configuration Not Loading

**Symptom**: Monitors not appearing or old configuration persists

**Solution**:

**Docker**:
```bash
# Force recreate containers
docker compose down
docker compose up -d --force-recreate
```

**Kubernetes**:
```bash
# Update ConfigMap
kubectl create configmap hallmonitor-config \
  --from-file=config.yml \
  --dry-run=client -o yaml | kubectl apply -f -

# Force pod restart
kubectl rollout restart deployment/hallmonitor -n hallmonitor
```

**Binary**:
```bash
# Verify config path
hallmonitor --config /path/to/config.yml

# Check file permissions
ls -la /path/to/config.yml
```

### Invalid YAML Syntax

**Symptom**: `yaml: unmarshal errors` or startup failure

**Solution**:
```bash
# Validate YAML syntax
python3 -c "import yaml; yaml.safe_load(open('config.yml'))"

# Or use online validator
# https://www.yamllint.com/

# Common issues:
# - Incorrect indentation (use spaces, not tabs)
# - Missing quotes around strings with special characters
# - Missing colons after keys
```

### Environment Variables Not Substituting

**Symptom**: Literal `${VAR_NAME}` in configuration instead of value

**Solution**:
```bash
# Ensure variables are exported
export API_TOKEN="your-token-here"

# Verify variable is set
echo $API_TOKEN

# Check Hall Monitor logs for substitution
# Variables are replaced at startup
```

## Monitor Issues

### HTTP Monitor Shows Down

**Possible Causes**:

1. **Wrong Expected Status**
```yaml
# Check actual response
curl -I https://example.com

# Update configuration
- type: "http"
  url: "https://example.com"
  expectedStatus: 200  # Match actual status
```

2. **SSL Certificate Issues**
```yaml
# Check certificate
openssl s_client -connect example.com:443 -servername example.com

# Hall Monitor automatically validates certificates
# Expired or invalid certs will cause failure
```

3. **Timeout Too Short**
```yaml
- type: "http"
  url: "https://slow-api.example.com"
  timeout: "30s"  # Increase for slow services
```

4. **Network Connectivity**
```bash
# Test from Hall Monitor's network context

# Docker:
docker exec hallmonitor curl -v https://example.com

# Kubernetes:
kubectl exec -it deployment/hallmonitor -n hallmonitor -- curl -v https://example.com
```

### DNS Monitor Failing

**Possible Causes**:

1. **DNS Server Unreachable**
```bash
# Test DNS server connectivity
dig @8.8.8.8 example.com

# Or using nslookup
nslookup example.com 8.8.8.8
```

2. **Wrong Query Type**
```yaml
# Verify record exists
dig -t A example.com  # For A records
dig -t AAAA example.com  # For AAAA records

# Update configuration
- type: "dns"
  target: "8.8.8.8:53"
  query: "example.com"
  queryType: "A"  # Must match existing record type
```

3. **Expected Response Mismatch**
```yaml
# Check actual response
dig @8.8.8.8 example.com

# Remove or update expectedResponse
- type: "dns"
  target: "8.8.8.8:53"
  query: "example.com"
  queryType: "A"
  expectedResponse: "93.184.216.34"  # Must match actual answer
```

### Ping Monitor Not Working

**Possible Causes**:

1. **Insufficient Privileges**
```bash
# ICMP requires elevated privileges

# Grant capability (binary)
sudo setcap cap_net_raw+ep /usr/local/bin/hallmonitor

# Docker with privileges
docker run --cap-add NET_RAW --cap-add NET_ADMIN \
  --network host \
  -v $(pwd)/config.yml:/etc/hallmonitor/config.yml:ro \
  ghcr.io/1broseidon/hallmonitor:latest

# Kubernetes
securityContext:
  capabilities:
    add: ["NET_RAW"]

# Note: Hall Monitor automatically falls back to unprivileged mode (UDP)
```

2. **Host Unreachable**
```bash
# Test connectivity
ping -c 3 192.168.1.1

# Check firewall rules
# ICMP may be blocked
```

3. **Timeout Too Short**
```yaml
- type: "ping"
  target: "192.168.1.1"
  count: 3
  timeout: "5s"  # Increase for slow networks
```

### TCP Monitor Failing

**Possible Causes**:

1. **Port Closed or Filtered**
```bash
# Test port connectivity
telnet db.example.com 5432
# Or
nc -zv db.example.com 5432
```

2. **Firewall Blocking**
```bash
# Check from Hall Monitor's network

# Docker:
docker exec hallmonitor nc -zv db.example.com 5432

# Kubernetes:
kubectl exec -it deployment/hallmonitor -n hallmonitor -- nc -zv db.example.com 5432
```

3. **Wrong Host or Port**
```yaml
- type: "tcp"
  target: "db.example.com:5432"  # Verify host and port
  # Format: "host:port" or "ip:port"
  # IPv6: "[::1]:8080"
```

## Performance Issues

### High Memory Usage

**Symptom**: Hall Monitor using too much memory

**Solution**:
```yaml
# Reduce check frequency
monitoring:
  defaultInterval: "60s"  # Increase from 30s

# Reduce number of monitors
# Comment out non-critical monitors

# For Kubernetes, increase limits
resources:
  limits:
    memory: 256Mi
  requests:
    memory: 128Mi
```

### High CPU Usage

**Symptom**: Hall Monitor consuming excessive CPU

**Solution**:
```yaml
# Reduce check frequency
monitoring:
  defaultInterval: "60s"

# Reduce concurrent checks
# Group monitors to stagger execution
groups:
  - name: "group-1"
    interval: "30s"
  - name: "group-2"
    interval: "35s"  # Offset by 5s

# Check for timeout issues
# Monitors timing out repeatedly consume resources
```

### Slow Response Times

**Symptom**: Dashboard slow to load or API calls timing out

**Solution**:
```bash
# Check if monitors are hanging
curl http://localhost:7878/api/v1/monitors

# Reduce result retention (if available)
# Check monitor timeouts
monitoring:
  defaultTimeout: "10s"  # Ensure reasonable timeouts

# Restart Hall Monitor
docker compose restart
# or
kubectl rollout restart deployment/hallmonitor -n hallmonitor
```

## Dashboard Issues

### Dashboard Not Loading

**Symptom**: 404 or blank page at `/`

**Solution**:
```yaml
# Ensure dashboard is enabled
server:
  enableDashboard: true  # Must be true

# Restart after config change
```

### Data Not Showing

**Symptom**: Dashboard loads but shows no monitors

**Possible Causes**:

1. **Monitors Not Configured**
```yaml
# Verify monitors are in configuration
monitoring:
  groups:
    - name: "my-monitors"
      monitors:
        - type: "http"
          name: "example"
          url: "https://example.com"
```

2. **Monitors Not Yet Checked**
```bash
# Wait 30-60 seconds for first check cycle
# Or check API directly
curl http://localhost:7878/api/v1/monitors
```

3. **JavaScript Errors**
```bash
# Open browser console (F12)
# Check for errors
# Ensure no ad blockers or extensions blocking scripts
```

## Metrics Issues

### Prometheus Not Scraping

**Symptom**: No Hall Monitor metrics in Prometheus

**Solution**:

1. **Verify metrics endpoint**
```bash
curl http://hallmonitor:7878/metrics
# Should return Prometheus format metrics
```

2. **Check Prometheus configuration**
```yaml
scrape_configs:
  - job_name: 'hallmonitor'
    static_configs:
      - targets: ['hallmonitor:7878']  # Verify hostname/IP
    metrics_path: '/metrics'
    scrape_interval: 15s
```

3. **Check Prometheus logs**
```bash
# Look for scrape errors
docker compose logs prometheus | grep hallmonitor

# Or in Prometheus UI
# Status > Targets
# Check hallmonitor target status
```

### Missing Metrics

**Symptom**: Some metrics not appearing

**Solution**:
```yaml
# Ensure metrics are enabled
metrics:
  enabled: true
  includeProcessMetrics: true
  includeGoMetrics: true

# Restart Hall Monitor
```

## Kubernetes Specific

### CrashLoopBackOff

**Symptom**: Pods repeatedly crashing

**Solution**:
```bash
# Check logs
kubectl logs -f deployment/hallmonitor -n hallmonitor

# Common causes:
# - Invalid configuration (check ConfigMap)
# - Missing environment variables
# - Resource limits too low
# - Image pull failures

# Check events
kubectl describe pod <pod-name> -n hallmonitor
```

### Network Policies Blocking

**Symptom**: Monitors can't reach targets

**Solution**:
```bash
# Check NetworkPolicies
kubectl get networkpolicies -n hallmonitor

# Test from pod
kubectl exec -it deployment/hallmonitor -n hallmonitor -- curl -v https://example.com

# Update NetworkPolicy to allow egress
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: hallmonitor-egress
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: hallmonitor
  policyTypes:
    - Egress
  egress:
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
```

### Service Not Accessible

**Symptom**: Can't access Hall Monitor service

**Solution**:
```bash
# Check service
kubectl get svc -n hallmonitor

# Port forward for testing
kubectl port-forward svc/hallmonitor 7878:7878 -n hallmonitor

# Check Ingress (if configured)
kubectl get ingress -n hallmonitor
kubectl describe ingress hallmonitor -n hallmonitor
```

## Docker Specific

### Container Won't Start

**Symptom**: Container exits immediately

**Solution**:
```bash
# Check logs
docker compose logs hallmonitor

# Run interactively for debugging
docker run -it --entrypoint /bin/sh ghcr.io/1broseidon/hallmonitor:latest

# Check if config file is mounted
docker compose exec hallmonitor ls -la /etc/hallmonitor/config.yml

# Verify environment variables
docker compose exec hallmonitor env | grep SERVER
```

### Volume Mount Issues

**Symptom**: Config file not found or permission denied

**Solution**:
```yaml
# Ensure absolute path in compose file
volumes:
  - ./config.yml:/etc/hallmonitor/config.yml:ro
  # Not: config.yml:/etc/hallmonitor/config.yml

# Check file exists
ls -la config.yml

# Fix permissions
chmod 644 config.yml
```

## Debug Mode

Enable debug logging for troubleshooting:

**Configuration**:
```yaml
logging:
  level: "debug"
  format: "text"  # Human readable
```

**Environment Variable**:
```bash
export LOG_LEVEL=debug
```

**Docker**:
```yaml
# docker-compose.yml or compose.yaml
environment:
  - LOG_LEVEL=debug
```

**Kubernetes**:
```bash
kubectl set env deployment/hallmonitor -n hallmonitor LOG_LEVEL=debug
```

## Getting Help

If you're still experiencing issues:

1. **Check logs with debug enabled**
2. **Review configuration for typos**
3. **Test connectivity manually**
4. **Check GitHub issues**: https://github.com/1broseidon/hallmonitor/issues
5. **Open a new issue with**:
   - Hall Monitor version
   - Deployment method (Docker/K8s/binary)
   - Relevant configuration (sanitized)
   - Error messages and logs
   - Steps to reproduce

## Related Documentation

- [Configuration Reference](./configuration.md)
- [Monitor Types](../03-monitors/index.md)
- [Installation Guide](../02-getting-started/installation.md)
