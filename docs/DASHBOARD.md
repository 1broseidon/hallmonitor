# Built-in Dashboard

Hall Monitor includes a lightweight, beautiful web dashboard built with Bulma CSS. Perfect for quick at-a-glance monitoring without needing to set up Grafana.

## Features

- **Zero Configuration**: Works out of the box
- **Lightweight**: Single HTML page with embedded CSS/JS (~200KB)
- **Real-time Updates**: Auto-refreshes every 30 seconds
- **Responsive Design**: Works on desktop, tablet, and mobile
- **Export to Grafana**: One-click export to full Grafana dashboard

## Accessing the Dashboard

Once Hall Monitor is running, visit:

```
http://localhost:7878/
```

or

```
http://localhost:7878/dashboard
```

## Dashboard Sections

### Hero Stats (Top Row)
- **Overall Uptime**: 7-day average uptime percentage
- **Active Incidents**: Number of monitors currently down
- **Avg Response Time**: Median response time across all checks
- **Total Monitors**: Number of configured monitors

### Monitor Health by Type
Interactive bar chart showing UP/DOWN counts per monitor type (HTTP, DNS, Ping, TCP).

### Currently Down Monitors
Red alert section showing which monitors are down right now. Shows "All Systems Operational" when everything is healthy.

### 24h Activity
Quick stats showing recent activity including uptime, check count, and errors.

### Response Time Performance
Line chart showing P50/P95/P99 response time percentiles over time.

### All Monitors Table
Complete table of all monitors with:
- Current status (Up/Down)
- Monitor name and type
- Group
- 7-day uptime percentage
- Average response time

## Configuration

Enable or disable the dashboard in your `config.yml`:

```yaml
server:
  port: "7878"
  host: "0.0.0.0"
  enableDashboard: true  # Set to false to disable
```

You can also set this via environment variable:

```bash
export SERVER_ENABLEDASHBOARD=false
```

## Exporting to Grafana

For advanced users who want the full Grafana experience:

1. Click "Export to Grafana" button in the dashboard
2. Save the `hallmonitor-dashboard.json` file
3. Import it into your Grafana instance:
   - Go to Dashboards → Import
   - Upload the JSON file
   - Select your Prometheus datasource
   - Click Import

The Grafana dashboard includes:
- Advanced drill-down features
- Historical data analysis
- Alerting capabilities
- Custom time ranges
- Panel customization

## API Endpoints

The dashboard queries these endpoints:

- `GET /api/v1/monitors` - List all monitors with current status
- `GET /api/v1/groups` - List monitor groups
- `GET /metrics` - Prometheus metrics (for charts)
- `GET /api/v1/grafana/dashboard` - Export Grafana JSON

## Disabling the Dashboard

If you prefer to use only Grafana or your own monitoring solution:

```yaml
server:
  enableDashboard: false
```

When disabled, the root path (`/`) will return a 404, but all API endpoints remain available.

## Performance

The dashboard is extremely lightweight:
- HTML: ~25KB
- Bulma CSS: ~200KB (CDN)
- Chart.js: ~200KB (CDN)
- Font Awesome: ~100KB (CDN)

Total: ~525KB loaded once, then cached by browser.

## Browser Compatibility

Works in all modern browsers:
- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Opera 76+

## Troubleshooting

### Dashboard not loading
1. Check that `enableDashboard: true` in config
2. Verify Hall Monitor is running: `curl http://localhost:7878/health`
3. Check logs for errors

### Data not showing
1. Verify monitors are configured in `config.yml`
2. Check that monitoring is running: `curl http://localhost:7878/api/v1/monitors`
3. Wait 30 seconds for first check cycle to complete

### Grafana export not working
The export requires the Grafana dashboard JSON file to be present at:
- `deploy/observability/grafana/provisioning/dashboards/hallmonitor-overview.json`

Or manually import from the repository.

